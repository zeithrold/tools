package evolve

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zeithrold/tools/internal/githubauth"
)

const (
	maxRemoteResponseBytes = 1 << 20
	remoteTimeout          = 30 * time.Second
)

// errRemoteUncertain marks a failure whose remote effect is unknown: the request
// may have been applied even though no response arrived. A caller must reconcile by
// reading before it writes again.
var errRemoteUncertain = errors.New("remote outcome unknown")

// githubEndpoint is one authenticated GitHub REST endpoint, bound to the host and
// repository the operator policy names. The credential is added per request by the
// githubauth transport; this type never holds or logs a token.
type githubEndpoint struct {
	base   *url.URL
	client *http.Client
	owner  string
	repo   string
}

// newGitHubEndpoint validates the endpoint provided by the authentication seam and
// binds it to the policy's repository.
func newGitHubEndpoint(api githubauth.API, target Target) (*githubEndpoint, error) {
	if strings.TrimSpace(api.BaseURL) == "" || api.Transport == nil {
		return nil, fmt.Errorf("GitHub API endpoint is not configured; `zt auth login` is required before a remote operation")
	}
	base, err := url.Parse(strings.TrimRight(api.BaseURL, "/"))
	if err != nil || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return nil, fmt.Errorf("GitHub API endpoint is invalid")
	}
	if base.Scheme != "https" {
		// A plain-HTTP endpoint is accepted only for a loopback test service; a real
		// submission always speaks HTTPS to the policy's host.
		if base.Scheme != "http" || !isLoopback(base.Hostname()) {
			return nil, fmt.Errorf("GitHub API endpoint must use HTTPS")
		}
	}
	owner, repo, err := splitRepository(target.Repository)
	if err != nil {
		return nil, err
	}
	return &githubEndpoint{
		base:  base,
		owner: owner,
		repo:  repo,
		client: &http.Client{
			Transport: api.Transport,
			Timeout:   remoteTimeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return fmt.Errorf("refusing to follow a GitHub redirect")
			},
		},
	}, nil
}

func splitRepository(value string) (string, string, error) {
	parts := strings.Split(value, "/")
	if len(parts) != 2 || !repositoryName.MatchString(value) {
		return "", "", fmt.Errorf("target repository must be OWNER/NAME")
	}
	return parts[0], parts[1], nil
}

// isLoopback reports whether host names this machine, so a plain-HTTP test service
// cannot stand in for a remote host.
func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	address := net.ParseIP(host)
	return address != nil && address.IsLoopback()
}

// do sends one request inside the repository subtree and returns the status. A
// transport failure, a rate limit or a server error is reported as
// errRemoteUncertain, because from the caller's point of view the request may
// already have been applied.
func (e *githubEndpoint) do(ctx context.Context, method, suffix string, query url.Values, body, out any) (int, error) {
	target := *e.base
	target.Path = strings.TrimRight(e.base.Path, "/") + suffix
	target.RawQuery = ""
	if query != nil {
		target.RawQuery = query.Encode()
	}
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return 0, fmt.Errorf("encode GitHub request: %w", err)
		}
		payload = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, target.String(), payload)
	if err != nil {
		return 0, fmt.Errorf("build GitHub request: %w", err)
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := e.client.Do(request)
	if err != nil {
		if errors.Is(err, githubauth.ErrReloginRequired) {
			return 0, err
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return 0, fmt.Errorf("%w: %s %s: %v", errRemoteUncertain, method, e.route(suffix), ctxErr)
		}
		return 0, fmt.Errorf("%w: %s %s failed (transport details omitted)", errRemoteUncertain, method, e.route(suffix))
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, maxRemoteResponseBytes+1))
	if err != nil {
		return 0, fmt.Errorf("%w: reading the GitHub response for %s %s failed", errRemoteUncertain, method, e.route(suffix))
	}
	if len(data) > maxRemoteResponseBytes {
		return 0, fmt.Errorf("GitHub response for %s %s exceeded %d bytes", method, e.route(suffix), maxRemoteResponseBytes)
	}
	switch {
	case response.StatusCode >= 500, response.StatusCode == http.StatusTooManyRequests, response.StatusCode == http.StatusRequestTimeout, response.StatusCode == http.StatusGatewayTimeout:
		return response.StatusCode, fmt.Errorf("%w: GitHub returned HTTP %d for %s %s", errRemoteUncertain, response.StatusCode, method, e.route(suffix))
	}
	if out != nil && len(data) > 0 && response.StatusCode >= 200 && response.StatusCode < 300 {
		if err := json.Unmarshal(data, out); err != nil {
			return response.StatusCode, fmt.Errorf("GitHub returned an unreadable response for %s %s", method, e.route(suffix))
		}
	}
	return response.StatusCode, nil
}

// route names a request in an error without echoing the query or the host.
func (e *githubEndpoint) route(suffix string) string {
	return "/repos/" + e.owner + "/" + e.repo + suffix
}

func (e *githubEndpoint) repoPath(suffix string) string {
	return "/repos/" + url.PathEscape(e.owner) + "/" + url.PathEscape(e.repo) + suffix
}

// refPath escapes one branch name segment by segment, so a branch cannot introduce
// a path separator or traversal of its own.
func refPath(name string) string {
	segments := strings.Split(name, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}
	return strings.Join(segments, "/")
}

type remoteRepository struct {
	FullName      string `json:"full_name"`
	DefaultBranch string `json:"default_branch"`
	Private       bool   `json:"private"`
}

type remoteRef struct {
	Ref    string `json:"ref"`
	Object struct {
		Type string `json:"type"`
		SHA  string `json:"sha"`
	} `json:"object"`
}

type remoteIdentity struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Date  string `json:"date"`
}

type remoteCommit struct {
	SHA       string         `json:"sha"`
	Message   string         `json:"message"`
	Tree      remoteObject   `json:"tree"`
	Parents   []remoteObject `json:"parents"`
	Author    remoteIdentity `json:"author"`
	Committer remoteIdentity `json:"committer"`
}

type remoteObject struct {
	SHA string `json:"sha"`
}

// remoteTree is GitHub's recursive tree response. The listing key is "tree": the
// wire contract is GitHub's, not ours, and a wrong key would silently decode as an
// empty listing and make every non-empty base look unmaterialisable.
type remoteTree struct {
	SHA       string            `json:"sha"`
	Truncated bool              `json:"truncated"`
	Entries   []remoteTreeEntry `json:"tree"`
}

type remoteTreeEntry struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
	Type string `json:"type"`
	SHA  string `json:"sha"`
}

type remotePullRepo struct {
	FullName string `json:"full_name"`
}

type remotePullRef struct {
	Ref  string          `json:"ref"`
	SHA  string          `json:"sha"`
	Repo *remotePullRepo `json:"repo"`
}

type remotePull struct {
	Number  int           `json:"number"`
	Title   string        `json:"title"`
	Body    string        `json:"body"`
	State   string        `json:"state"`
	Draft   bool          `json:"draft"`
	HTMLURL string        `json:"html_url"`
	Head    remotePullRef `json:"head"`
	Base    remotePullRef `json:"base"`
}

type remoteCheckRun struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

type remoteCheckRuns struct {
	TotalCount int              `json:"total_count"`
	CheckRuns  []remoteCheckRun `json:"check_runs"`
}

type remoteBlobRequest struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

type remoteTreeRequest struct {
	BaseTree string            `json:"base_tree,omitempty"`
	Tree     []remoteTreeEntry `json:"tree"`
}

type remoteCommitRequest struct {
	Message   string         `json:"message"`
	Tree      string         `json:"tree"`
	Parents   []string       `json:"parents"`
	Author    remoteIdentity `json:"author"`
	Committer remoteIdentity `json:"committer"`
}

type remoteRefRequest struct {
	Ref string `json:"ref"`
	SHA string `json:"sha"`
}

type remotePullRequest struct {
	Title string `json:"title"`
	Head  string `json:"head"`
	Base  string `json:"base"`
	Body  string `json:"body"`
	Draft bool   `json:"draft"`
}

func (e *githubEndpoint) repository(ctx context.Context) (remoteRepository, error) {
	var repository remoteRepository
	status, err := e.do(ctx, http.MethodGet, e.repoPath(""), nil, nil, &repository)
	if err != nil {
		return remoteRepository{}, err
	}
	switch status {
	case http.StatusOK:
		if !strings.EqualFold(repository.FullName, e.owner+"/"+e.repo) {
			return remoteRepository{}, fmt.Errorf("GitHub returned a different repository than the policy names; refusing to act on it")
		}
		if repository.DefaultBranch == "" {
			return remoteRepository{}, fmt.Errorf("GitHub did not report a default branch for the target repository")
		}
		return repository, nil
	case http.StatusNotFound:
		return remoteRepository{}, fmt.Errorf("GitHub cannot see repository %s with this credential", e.route(""))
	default:
		return remoteRepository{}, fmt.Errorf("GitHub returned HTTP %d for %s", status, e.route(""))
	}
}

func (e *githubEndpoint) ref(ctx context.Context, fullRef string) (*remoteRef, error) {
	name, ok := strings.CutPrefix(fullRef, "refs/heads/")
	if !ok {
		return nil, fmt.Errorf("unsupported ref %s", safeQuote(fullRef))
	}
	var value remoteRef
	status, err := e.do(ctx, http.MethodGet, e.repoPath("/git/ref/heads/"+refPath(name)), nil, nil, &value)
	if err != nil {
		return nil, err
	}
	switch status {
	case http.StatusOK:
		return &value, nil
	case http.StatusNotFound:
		return nil, nil
	default:
		return nil, fmt.Errorf("GitHub returned HTTP %d reading %s", status, e.route("/git/ref/heads/"+refPath(name)))
	}
}

func (e *githubEndpoint) commitObject(ctx context.Context, sha string) (remoteCommit, error) {
	var commit remoteCommit
	status, err := e.do(ctx, http.MethodGet, e.repoPath("/git/commits/"+url.PathEscape(sha)), nil, nil, &commit)
	if err != nil {
		return remoteCommit{}, err
	}
	if status != http.StatusOK {
		return remoteCommit{}, fmt.Errorf("GitHub returned HTTP %d reading commit %s", status, safeQuote(sha))
	}
	return commit, nil
}

// treeEntries reads one complete remote tree. A truncated listing is refused: a
// partial tree cannot prove that a head contains exactly the expected files.
func (e *githubEndpoint) treeEntries(ctx context.Context, sha string) (map[string]remoteTreeEntry, error) {
	var tree remoteTree
	status, err := e.do(ctx, http.MethodGet, e.repoPath("/git/trees/"+url.PathEscape(sha)), url.Values{"recursive": []string{"1"}}, nil, &tree)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GitHub returned HTTP %d reading tree %s", status, safeQuote(sha))
	}
	if tree.Truncated {
		return nil, fmt.Errorf("GitHub truncated the tree listing for %s, so zt cannot prove what the head contains; refusing", safeQuote(sha))
	}
	entries := make(map[string]remoteTreeEntry, len(tree.Entries))
	for _, entry := range tree.Entries {
		if entry.Type == "tree" {
			continue
		}
		if len(entries) >= 2*maxBaseEntries {
			return nil, fmt.Errorf("remote tree holds more than %d entries", 2*maxBaseEntries)
		}
		entries[entry.Path] = entry
	}
	return entries, nil
}

func (e *githubEndpoint) createBlob(ctx context.Context, content []byte) (string, error) {
	var created remoteObject
	status, err := e.do(ctx, http.MethodPost, e.repoPath("/git/blobs"), nil, remoteBlobRequest{Content: string(content), Encoding: "utf-8"}, &created)
	if err != nil {
		return "", err
	}
	if status != http.StatusCreated {
		return "", fmt.Errorf("GitHub returned HTTP %d creating a blob", status)
	}
	if created.SHA == "" {
		return "", fmt.Errorf("GitHub created a blob without reporting its id")
	}
	return created.SHA, nil
}

func (e *githubEndpoint) createTree(ctx context.Context, baseTree string, entries []remoteTreeEntry) (string, error) {
	var created remoteObject
	status, err := e.do(ctx, http.MethodPost, e.repoPath("/git/trees"), nil, remoteTreeRequest{BaseTree: baseTree, Tree: entries}, &created)
	if err != nil {
		return "", err
	}
	if status != http.StatusCreated {
		return "", fmt.Errorf("GitHub returned HTTP %d creating a tree", status)
	}
	if created.SHA == "" {
		return "", fmt.Errorf("GitHub created a tree without reporting its id")
	}
	return created.SHA, nil
}

func (e *githubEndpoint) createCommit(ctx context.Context, payload PayloadCommit, tree string, parents []string) (string, error) {
	identity := remoteIdentity{Name: payload.AuthorName, Email: payload.AuthorEmail, Date: payload.CommittedAt}
	var created remoteObject
	status, err := e.do(ctx, http.MethodPost, e.repoPath("/git/commits"), nil, remoteCommitRequest{
		Message: payload.Message, Tree: tree, Parents: parents, Author: identity, Committer: identity,
	}, &created)
	if err != nil {
		return "", err
	}
	if status != http.StatusCreated {
		return "", fmt.Errorf("GitHub returned HTTP %d creating a commit", status)
	}
	if created.SHA == "" {
		return "", fmt.Errorf("GitHub created a commit without reporting its id")
	}
	return created.SHA, nil
}

// createRef reports the status so a conflict can be reconciled by reading instead
// of being retried blindly.
func (e *githubEndpoint) createRef(ctx context.Context, branch, sha string) (int, error) {
	var created remoteRef
	return e.do(ctx, http.MethodPost, e.repoPath("/git/refs"), nil, remoteRefRequest{Ref: "refs/heads/" + branch, SHA: sha}, &created)
}

func (e *githubEndpoint) pullsForBranch(ctx context.Context, branch string) ([]remotePull, error) {
	var pulls []remotePull
	status, err := e.do(ctx, http.MethodGet, e.repoPath("/pulls"), url.Values{
		"head":  []string{e.owner + ":" + branch},
		"state": []string{"all"},
	}, nil, &pulls)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GitHub returned HTTP %d listing pull requests", status)
	}
	return pulls, nil
}

func (e *githubEndpoint) createPull(ctx context.Context, payload PayloadPullRequest, branch, baseBranch string) (remotePull, int, error) {
	var created remotePull
	status, err := e.do(ctx, http.MethodPost, e.repoPath("/pulls"), nil, remotePullRequest{
		Title: payload.Title, Head: branch, Base: baseBranch, Body: payload.Body, Draft: payload.Draft,
	}, &created)
	return created, status, err
}

func (e *githubEndpoint) pull(ctx context.Context, number int) (remotePull, error) {
	var pull remotePull
	status, err := e.do(ctx, http.MethodGet, e.repoPath(fmt.Sprintf("/pulls/%d", number)), nil, nil, &pull)
	if err != nil {
		return remotePull{}, err
	}
	if status != http.StatusOK {
		return remotePull{}, fmt.Errorf("GitHub returned HTTP %d reading pull request %d", status, number)
	}
	return pull, nil
}

// checkRuns reads the checks GitHub reports for one commit. They are the platform's
// observation, not zt's execution. found is false when GitHub holds no run for the
// commit, which is different from a failed read.
func (e *githubEndpoint) checkRuns(ctx context.Context, sha string) (remoteCheckRuns, bool, error) {
	var runs remoteCheckRuns
	status, err := e.do(ctx, http.MethodGet, e.repoPath("/commits/"+url.PathEscape(sha)+"/check-runs"), nil, nil, &runs)
	if err != nil {
		return remoteCheckRuns{}, false, err
	}
	switch status {
	case http.StatusOK:
		return runs, true, nil
	case http.StatusNotFound:
		return remoteCheckRuns{}, false, nil
	default:
		return remoteCheckRuns{}, false, fmt.Errorf("GitHub returned HTTP %d reading check runs", status)
	}
}
