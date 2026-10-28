package evolve

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/format/packfile"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp/capability"
	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/filesystem"

	"github.com/zeithrold/tools/internal/githubauth"
)

const (
	// maxUpstreamPackBytes bounds the packfile bytes read from the upload-pack
	// response, so the object payload one fetch ingests cannot be unbounded. It is
	// a transfer cap, not a decoded-resource guarantee: go-git streams the pack to
	// the repository's pack directory and keeps one index entry per object in
	// memory as it parses. Decompressed object, tree and total-size bounds belong
	// to the caller's verification of the finished baseline.
	maxUpstreamPackBytes = 128 << 20

	// upstreamFetchTimeout bounds discovery and transfer together. The caller's
	// context shortens it; it never extends it.
	upstreamFetchTimeout = 5 * time.Minute

	// upstreamRefName is the one ref that keeps the fetched commit reachable for
	// Git's own garbage collection and integrity checks. It is a private
	// namespace, never an upstream branch or tag name.
	upstreamRefName plumbing.ReferenceName = "refs/zt/upstream"
)

var (
	// errUpstreamRedirect is returned instead of ever following a redirect. The
	// credential is bound to one host and repository, so a second hop must never
	// be asked to send it.
	errUpstreamRedirect = errors.New("upstream fetch refused an HTTP redirect")

	// errUpstreamPackBound is returned once the encoded response grows past
	// maxUpstreamPackBytes.
	errUpstreamPackBound = fmt.Errorf("upstream baseline response exceeds %d bytes", maxUpstreamPackBytes)
)

// fetchGitBaseline acquires exactly one pinned commit at depth one into a new
// bare repository at directory, which must not already exist. It drives the
// stable go-git v5 smart HTTP upload-pack client and its object storage, so the
// adapter only decides what to ask for and what to accept; it implements no
// wire protocol, proxy, ACK or delta handling of its own.
//
// The remote is the canonical https://<host>/<repository>.git the caller was
// handed; a plain-HTTP loopback URL is accepted only so library tests can serve
// a native git http-backend fixture. endpoint.Transport, which attaches the Git
// credential, is the HTTP client's only transport, and both the client and a
// thin wrapper refuse redirects, so no second host is ever asked for the
// credential. The pinned commit must be advertised or the server must advertise
// that it accepts an unadvertised want; a depth-one boundary other than the
// pinned commit, an incomplete pack and a response above the encoded-size bound
// are refused.
//
// The packfile bytes are capped at maxUpstreamPackBytes. That cap limits the
// object payload transferred and the pack written to disk, not decompressed
// object sizes or the number of index entries the library retains while parsing;
// the complete decoded baseline is bounded by the caller's own verification.
//
// A successful call leaves exactly what a native reader needs: the packfile and
// its index, the response's shallow boundary and one ref pointing at the pinned
// commit. Nothing is checked out, no submodule or LFS object is resolved, and no
// credential is written to a URL, argument, environment variable or file. The
// repository is not verified here; the caller verifies the complete baseline
// before it publishes the cache entry. A failed fetch can leave a partial
// repository in directory, so the caller owns that staging directory and removes
// it.
func fetchGitBaseline(ctx context.Context, directory string, target Target, endpoint githubauth.GitEndpoint) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// GitHub's Git transport serves SHA-1 repositories. The caller rejects a
	// SHA-256 policy base already; the adapter never guesses a format.
	if length, ok := fullCommitID(target.BaseCommit); !ok || length != 40 {
		return fmt.Errorf("upstream fetch requires one complete lowercase SHA-1 commit, not %s", safeQuote(target.BaseCommit))
	}
	if err := validateUpstreamEndpoint(target, endpoint); err != nil {
		return err
	}
	storage, err := createBareUpstreamRepository(directory)
	if err != nil {
		return err
	}
	defer storage.Close()

	fetchCtx, cancel := context.WithTimeout(ctx, upstreamFetchTimeout)
	defer cancel()

	httpClient := &http.Client{
		// The credential is attached per request by the authenticated transport;
		// the adapter never sees it and cannot send it elsewhere.
		Transport:     redirectRefusingTransport{base: endpoint.Transport},
		Timeout:       upstreamFetchTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return errUpstreamRedirect },
	}
	client := githttp.NewClientWithOptions(httpClient, &githttp.ClientOptions{
		RedirectPolicy: githttp.NoFollowRedirects,
	})
	ep, err := transport.NewEndpoint(endpoint.URL)
	if err != nil {
		return fmt.Errorf("upstream endpoint is not a Git URL")
	}
	session, err := client.NewUploadPackSession(ep, nil)
	if err != nil {
		return upstreamTransferError("session setup", err, fetchCtx)
	}
	defer session.Close()

	advertised, err := session.AdvertisedReferencesContext(fetchCtx)
	if err != nil {
		return upstreamTransferError("reference discovery", err, fetchCtx)
	}
	want := plumbing.NewHash(target.BaseCommit)
	if err := checkWantedCommitAdvertised(advertised, want); err != nil {
		return err
	}
	if !advertised.Capabilities.Supports(capability.Shallow) {
		return fmt.Errorf("upstream server does not advertise shallow fetches, so zt cannot request only the pinned commit at depth one")
	}

	request := packp.NewUploadPackRequest()
	request.Wants = []plumbing.Hash{want}
	request.Depth = packp.DepthCommits(1)
	if err := request.Capabilities.Set(capability.Shallow); err != nil {
		return fmt.Errorf("upstream request: %w", err)
	}
	// The response is read as a plain packfile. Sideband, thin-pack and
	// multi_ack are deliberately not requested: the adapter wants one complete
	// pack and no progress or negotiation channel to demultiplex.
	if advertised.Capabilities.Supports(capability.NoProgress) {
		if err := request.Capabilities.Set(capability.NoProgress); err != nil {
			return fmt.Errorf("upstream request: %w", err)
		}
	}
	response, err := session.UploadPack(fetchCtx, request)
	if err != nil {
		return upstreamTransferError("pack transfer", err, fetchCtx)
	}
	if response == nil {
		return fmt.Errorf("upstream pack transfer returned no response")
	}
	defer response.Close()

	if len(response.ShallowUpdate.Unshallows) != 0 {
		return fmt.Errorf("upstream server sent an unshallow boundary for a fresh depth-one fetch")
	}
	packed := &boundedUpstreamPack{reader: response, limit: maxUpstreamPackBytes}
	if err := packfile.UpdateObjectStorage(storage, packed); err != nil {
		if ctxErr := fetchCtx.Err(); ctxErr != nil {
			return fmt.Errorf("upstream pack transfer: %w", ctxErr)
		}
		if errors.Is(err, errUpstreamPackBound) {
			return errUpstreamPackBound
		}
		return &upstreamTransferFailure{action: "pack decoding", cause: err}
	}
	if err := fetchCtx.Err(); err != nil {
		return fmt.Errorf("upstream pack transfer: %w", err)
	}
	return recordUpstreamBaseline(storage, want, response.ShallowUpdate)
}

// validateUpstreamEndpoint binds the endpoint to the policy target before any
// request is built. The URL must name exactly the target repository; only a
// loopback host may use plain HTTP, and only for native test fixtures.
func validateUpstreamEndpoint(target Target, endpoint githubauth.GitEndpoint) error {
	if endpoint.Transport == nil {
		return fmt.Errorf("upstream fetch requires the authenticated HTTP transport, so an unauthenticated connection is refused")
	}
	if !repositoryName.MatchString(target.Repository) {
		return fmt.Errorf("upstream target repository must be OWNER/NAME, not %s", safeQuote(target.Repository))
	}
	if !hostname.MatchString(target.Host) {
		return fmt.Errorf("upstream target host is not one hostname")
	}
	parsed, err := url.Parse(endpoint.URL)
	if err != nil {
		return fmt.Errorf("upstream endpoint is not a URL")
	}
	if parsed.User != nil || parsed.Fragment != "" || parsed.RawQuery != "" || parsed.ForceQuery || parsed.RawPath != "" || parsed.Opaque != "" {
		return fmt.Errorf("upstream endpoint must be the canonical target URL without credentials, escaping, query or fragment")
	}
	if parsed.Path != "/"+target.Repository+".git" {
		return fmt.Errorf("upstream endpoint does not name the policy target repository")
	}
	switch parsed.Scheme {
	case "https":
		if parsed.Host != target.Host {
			return fmt.Errorf("upstream endpoint host is not the policy target host")
		}
	case "http":
		// A native git http-backend fixture is the only plain-HTTP service the
		// adapter accepts; it must still be this machine and the policy host.
		if !isLoopback(parsed.Hostname()) || parsed.Hostname() != target.Host {
			return fmt.Errorf("upstream endpoint must use HTTPS unless a loopback host serves the target")
		}
	default:
		return fmt.Errorf("upstream endpoint must use HTTPS")
	}
	return nil
}

// createBareUpstreamRepository initializes the staging repository with go-git's
// filesystem object, reference and config storage, without the high-level
// repository package and its unrelated protocol registrations. The directory
// must be new, so the caller's staging area is the only place a failed fetch can
// have written. HEAD starts as Git's default unborn branch and is detached at the
// pinned commit once the baseline is recorded: a symbolic HEAD may only name a
// branch, and zt never claims the pinned commit is one.
func createBareUpstreamRepository(directory string) (*filesystem.Storage, error) {
	_, err := os.Lstat(directory)
	switch {
	case err == nil:
		return nil, fmt.Errorf("upstream staging repository already exists: %s", safeQuote(directory))
	case !errors.Is(err, os.ErrNotExist):
		return nil, fmt.Errorf("upstream staging repository: %w", err)
	}
	if err := os.Mkdir(directory, 0o700); err != nil {
		return nil, fmt.Errorf("upstream staging repository: %w", err)
	}
	storage := filesystem.NewStorage(osfs.New(directory), cache.NewObjectLRUDefault())
	if err := storage.Init(); err != nil {
		storage.Close()
		return nil, fmt.Errorf("initialize bare upstream repository: %w", err)
	}
	configuration := config.NewConfig()
	configuration.Core.IsBare = true
	if err := storage.SetConfig(configuration); err != nil {
		storage.Close()
		return nil, fmt.Errorf("configure bare upstream repository: %w", err)
	}
	if err := storage.SetReference(plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.Master)); err != nil {
		storage.Close()
		return nil, fmt.Errorf("initialize upstream HEAD: %w", err)
	}
	return storage, nil
}

// checkWantedCommitAdvertised refuses a want the server has not agreed to send.
// A commit that is an advertised head or reference tip is always allowed;
// otherwise the server must advertise that it accepts an unadvertised tip or any
// reachable commit. A peeled tag target is deliberately not enough: upload-pack
// advertises the tag object, not that peeled commit, as a want. Without one of
// those, upload-pack would reject the request anyway, and checking here keeps
// the refusal inside zt's own message.
func checkWantedCommitAdvertised(advertised *packp.AdvRefs, want plumbing.Hash) error {
	if advertised != nil {
		if advertised.Head != nil && *advertised.Head == want {
			return nil
		}
		for _, hash := range advertised.References {
			if hash == want {
				return nil
			}
		}
		if advertised.Capabilities != nil &&
			(advertised.Capabilities.Supports(capability.AllowTipSHA1InWant) ||
				advertised.Capabilities.Supports(capability.AllowReachableSHA1InWant)) {
			return nil
		}
	}
	return fmt.Errorf("upstream server does not advertise the pinned commit %s and does not allow fetching an unadvertised commit", want)
}

// recordUpstreamBaseline confirms the response really delivered the pinned
// commit before it writes the metadata a native reader uses. The shallow boundary
// must be exactly the pinned commit when it has a parent, because that is the
// only boundary a depth-one fetch of this one want can have. A root commit has no
// parent to hide, so an empty boundary is complete too.
func recordUpstreamBaseline(storage *filesystem.Storage, want plumbing.Hash, update packp.ShallowUpdate) error {
	hasParents, err := commitHeaderHasParent(storage, want)
	if err != nil {
		return fmt.Errorf("upstream response did not deliver the pinned commit %s, so the incomplete baseline is refused", want)
	}
	if len(update.Shallows) > 1 {
		return fmt.Errorf("upstream server reported more than one shallow boundary for one depth-one want")
	}
	for _, hash := range update.Shallows {
		if hash != want {
			return fmt.Errorf("upstream server reported shallow boundary %s instead of the pinned commit", hash)
		}
	}
	if hasParents && len(update.Shallows) == 0 {
		return fmt.Errorf("upstream server did not mark the pinned commit as a depth-one shallow boundary")
	}
	if len(update.Shallows) != 0 {
		if err := storage.SetShallow(update.Shallows); err != nil {
			return fmt.Errorf("record upstream shallow boundary: %w", err)
		}
	}
	if err := storage.SetReference(plumbing.NewHashReference(upstreamRefName, want)); err != nil {
		return fmt.Errorf("record pinned upstream ref: %w", err)
	}
	// HEAD is kept detached at the pinned commit rather than on an invented
	// branch, so the bare cache is readable as-is and Git's integrity check does
	// not see a symbolic HEAD outside refs/heads.
	if err := storage.SetReference(plumbing.NewHashReference(plumbing.HEAD, want)); err != nil {
		return fmt.Errorf("record upstream HEAD: %w", err)
	}
	return nil
}

// commitHeaderHasParent reports whether the delivered object is a commit with
// at least one parent. It reads only the object header, so the adapter does not
// need the high-level object package and its signature verification dependencies
// just to decide whether a depth-one boundary is required.
func commitHeaderHasParent(storage *filesystem.Storage, want plumbing.Hash) (bool, error) {
	encoded, err := storage.EncodedObject(plumbing.CommitObject, want)
	if err != nil {
		return false, err
	}
	reader, err := encoded.Reader()
	if err != nil {
		return false, err
	}
	defer reader.Close()
	header := bufio.NewScanner(reader)
	for header.Scan() {
		line := header.Text()
		if line == "" {
			return false, nil
		}
		if strings.HasPrefix(line, "parent ") {
			return true, nil
		}
	}
	if err := header.Err(); err != nil {
		return false, err
	}
	return false, fmt.Errorf("commit object ended before its header")
}

// redirectRefusingTransport turns any 3xx response into an error before the
// HTTP client can consider following it. It delegates every other request and
// response unchanged to the authenticated transport the caller supplied.
type redirectRefusingTransport struct {
	base http.RoundTripper
}

func (t redirectRefusingTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := t.base.RoundTrip(request)
	if err != nil {
		return nil, err
	}
	if response == nil || response.Body == nil {
		return nil, errors.New("upstream transport returned no response body")
	}
	if response.StatusCode >= http.StatusMultipleChoices && response.StatusCode < http.StatusBadRequest {
		_ = response.Body.Close()
		return nil, errUpstreamRedirect
	}
	// Bound responses before go-git parses reference advertisements and shallow
	// headers as well as pack bytes. The pack parser's separate cap below still
	// enforces exactly maxUpstreamPackBytes on the object stream.
	limit := int64(maxGitOutputBytes)
	if request.Method == http.MethodPost {
		limit += maxUpstreamPackBytes
	}
	response.Body = &boundedUpstreamBody{
		boundedUpstreamPack: boundedUpstreamPack{reader: response.Body, limit: limit},
		closer:              response.Body,
	}
	return response, nil
}

type boundedUpstreamBody struct {
	boundedUpstreamPack
	closer io.Closer
}

func (b *boundedUpstreamBody) Close() error { return b.closer.Close() }

// boundedUpstreamPack refuses to read more than limit bytes from the encoded
// packfile response. It lets one probe read happen exactly at the bound, so a
// pack that ends precisely there is still accepted.
type boundedUpstreamPack struct {
	reader io.Reader
	limit  int64
	read   int64
}

func (r *boundedUpstreamPack) Read(p []byte) (int, error) {
	if r.read > r.limit {
		return 0, errUpstreamPackBound
	}
	if r.read == r.limit {
		var probe [1]byte
		n, err := r.reader.Read(probe[:])
		if n > 0 {
			return 0, errUpstreamPackBound
		}
		return 0, err
	}
	if int64(len(p)) > r.limit-r.read {
		p = p[:r.limit-r.read]
	}
	n, err := r.reader.Read(p)
	r.read += int64(n)
	if r.read > r.limit {
		return 0, errUpstreamPackBound
	}
	return n, err
}

// upstreamTransferFailure reports one failed request without repeating the
// transport's own message, which can contain the endpoint or response content.
// The cause stays available for errors.Is/errors.As.
type upstreamTransferFailure struct {
	action string
	cause  error
}

func (e *upstreamTransferFailure) Error() string {
	return "upstream " + e.action + " failed; the transport message is not repeated because it can contain endpoint or response content"
}

func (e *upstreamTransferFailure) Unwrap() error { return e.cause }

// upstreamTransferError classifies a request failure. Cancellation is reported as
// the caller's own context error; the recognized protocol failures get a fixed
// message; anything else becomes an upstreamTransferFailure.
func upstreamTransferError(action string, cause error, ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("upstream %s: %w", action, err)
	}
	switch {
	case errors.Is(cause, errUpstreamRedirect):
		return fmt.Errorf("upstream %s: %w; the credential is sent only to the canonical target", action, errUpstreamRedirect)
	case errors.Is(cause, transport.ErrRepositoryNotFound):
		return fmt.Errorf("upstream %s: the repository was not found or is not readable with the stored credential", action)
	case errors.Is(cause, transport.ErrAuthenticationRequired):
		return fmt.Errorf("upstream %s: the server requires authentication for this repository", action)
	case errors.Is(cause, transport.ErrAuthorizationFailed):
		return fmt.Errorf("upstream %s: the stored credential is not authorized to read this repository", action)
	case errors.Is(cause, transport.ErrEmptyRemoteRepository):
		return fmt.Errorf("upstream %s: the repository advertises no references", action)
	}
	return &upstreamTransferFailure{action: action, cause: cause}
}
