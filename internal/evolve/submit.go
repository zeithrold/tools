package evolve

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/zeithrold/tools/internal/githubauth"
)

// SubmitOptions selects one candidate, the trusted policy, the local upstream that
// holds the pinned base and the authenticated endpoint. OpenAPI is the credential
// seam: it is only called after every local precondition has passed, and it never
// hands zt a token.
type SubmitOptions struct {
	Candidate   string
	Policy      string
	Upstream    string
	Root        string
	ShowPayload bool
	OpenAPI     func(ctx context.Context, target Target) (githubauth.API, error)
}

// SubmitPullRequest is the observed pull request.
type SubmitPullRequest struct {
	Number int    `json:"number"`
	URL    string `json:"url,omitempty"`
	State  string `json:"state,omitempty"`
	Draft  bool   `json:"draft"`
}

// SubmitResult reports what happened, including the case where nothing was written
// because an exact pull request already existed.
type SubmitResult struct {
	SchemaVersion       int                `json:"schemaVersion"`
	Action              string             `json:"action"` // payload, created, existing
	State               string             `json:"state"`  // planned, completed, uncertain
	CandidateID         string             `json:"candidateId"`
	Target              Target             `json:"target"`
	PublicPayloadDigest string             `json:"publicPayloadDigest"`
	Branch              string             `json:"branch"`
	BaseBranch          string             `json:"baseBranch,omitempty"`
	Commit              string             `json:"commit,omitempty"`
	Tree                string             `json:"tree,omitempty"`
	PullRequest         *SubmitPullRequest `json:"pullRequest,omitempty"`
	Writes              int                `json:"writes"`
	Publication         string             `json:"publication"`
	Payload             *PublicPayload     `json:"payload,omitempty"`
	JournalPath         string             `json:"journalPath,omitempty"`
	Notes               []string           `json:"notes,omitempty"`
}

// Submit turns one stored candidate into a deterministic dedicated branch and a
// draft pull request.
//
// It never merges, publishes, activates, force-updates a ref or reopens a closed
// pull request. Every local precondition — the candidate bytes and the policy
// binding, plus the journal of an earlier attempt at this payload — is checked
// against a read-only view of the workspace before the credential seam is used.
// Submission deliberately consults no local verification record: `evolve check` is
// an independent utility, and a submit needs neither that record nor the logs and
// workspace a check run retains.
//
// The candidate+payload transaction is then serialised by a cross-process lock.
// Every remaining step is decided by reading: an existing submission of this
// candidate is returned without any write, a verified branch missing only its pull
// request gets just the pull request, and any object that is not exactly this
// candidate's is refused. The pull request targets the repository's current default
// branch, so a default branch that has advanced does not stop a submission: the
// branch still points at one commit whose parent is the pinned base, and zt neither
// rebases it nor changes its content.
func Submit(ctx context.Context, options SubmitOptions) (SubmitResult, error) {
	if err := ctx.Err(); err != nil {
		return SubmitResult{}, err
	}
	policy, err := LoadPolicy(options.Policy)
	if err != nil {
		return SubmitResult{}, err
	}
	data, id, err := readCandidate(options.Candidate)
	if err != nil {
		return SubmitResult{}, err
	}
	var candidate Candidate
	if err := decodeDocument(data, &candidate, "candidate"); err != nil {
		return SubmitResult{}, err
	}
	canonical, err := candidate.canonicalBytes()
	if err != nil {
		return SubmitResult{}, fmt.Errorf("encode candidate: %w", err)
	}
	if !bytes.Equal(canonical, data) {
		return SubmitResult{}, fmt.Errorf("candidate: bytes are not canonical, so the file was modified after preparation")
	}
	repo, err := resolveUpstream(ctx, options.Upstream, options.Root, policy)
	if err != nil {
		return SubmitResult{}, err
	}
	if err := validateCandidate(ctx, repo, policy, candidate); err != nil {
		return SubmitResult{}, err
	}
	info, err := repo.CommitInfo(ctx, candidate.Target.BaseCommit)
	if err != nil {
		return SubmitResult{}, err
	}
	payload, err := buildPayload(candidate, id, info)
	if err != nil {
		return SubmitResult{}, err
	}
	payloadDigest, err := payload.digest()
	if err != nil {
		return SubmitResult{}, err
	}
	result := SubmitResult{
		SchemaVersion:       schemaVersion,
		State:               "planned",
		Action:              "payload",
		CandidateID:         id,
		Target:              candidate.Target,
		PublicPayloadDigest: payloadDigest,
		Branch:              payload.Branch,
		Publication:         publicationNotReady,
	}

	// The review preview stops here: it reads no credential and no journal, and it
	// writes nothing.
	if options.ShowPayload {
		result.Payload = &payload
		return result, nil
	}

	// Local preflight: a read-only view of the owned tree, so a malformed journal
	// fails before any credential is touched and before any directory is created.
	// A local verification is not consulted: `evolve check` is independent, and a
	// submission needs neither its record nor the logs and workspace it retained.
	readStore, err := openExistingStore(options.Root)
	if err != nil {
		return SubmitResult{}, err
	}
	defer readStore.Close()
	journalRelative := submitJournalRelative(id, payloadDigest)
	journal, found, err := loadJournal(readStore, journalRelative, id, payloadDigest)
	if err != nil {
		return SubmitResult{}, err
	}
	result.JournalPath = path.Join(artifactRoot, journalRelative)
	if !found {
		journal = Journal{
			SchemaVersion:       schemaVersion,
			CandidateID:         id,
			PublicPayloadDigest: payloadDigest,
		}
	}
	// This run records the candidate and payload it actually publishes, so a
	// replayed attempt describes the same transaction.
	journal.BaseCommit = candidate.Target.BaseCommit
	journal.Branch = payload.Branch
	baseEntries, err := repo.CompleteTree(ctx, candidate.Target.BaseCommit)
	if err != nil {
		return SubmitResult{}, err
	}
	expected, err := expectedTree(ctx, repo, baseEntries, candidate)
	if err != nil {
		return SubmitResult{}, err
	}

	// From here the transaction is serialised. The lock is only taken once every
	// local precondition above has passed; a preview, a local status, or a failing
	// preflight never reaches it.
	unlock, err := submissionLock(ctx, filepath.Join(readStore.dir, "locks"), id+"-"+payloadDigest)
	if err != nil {
		return SubmitResult{}, err
	}
	defer unlock()
	writeStore, err := openStore(options.Root)
	if err != nil {
		return SubmitResult{}, err
	}
	defer writeStore.Close()
	if current, currentFound, err := loadJournal(writeStore, journalRelative, id, payloadDigest); err != nil {
		return SubmitResult{}, err
	} else if currentFound {
		// Re-read under the lock: another process may have advanced this transaction
		// after our preflight read.
		journal = current
		journal.BaseCommit = candidate.Target.BaseCommit
		journal.Branch = payload.Branch
	}

	if options.OpenAPI == nil {
		return SubmitResult{}, fmt.Errorf("GitHub submission needs the githubauth credential seam, which this build did not provide")
	}
	api, err := options.OpenAPI(ctx, candidate.Target)
	if err != nil {
		return SubmitResult{}, err
	}
	endpoint, err := newGitHubEndpoint(api, candidate.Target)
	if err != nil {
		return SubmitResult{}, err
	}
	repository, err := endpoint.repository(ctx)
	if err != nil {
		return SubmitResult{}, err
	}
	baseBranch := repository.DefaultBranch
	result.BaseBranch = baseBranch
	journal.BaseBranch = baseBranch

	state, err := reconcile(ctx, repo, endpoint, payload, expected, baseBranch)
	if err != nil {
		return SubmitResult{}, err
	}
	if state.pull != nil {
		return finishExisting(ctx, writeStore, journalRelative, result, &journal, *state.pull)
	}

	commit := state.head
	if commit != "" {
		// The dedicated branch already holds exactly this candidate's commit and only
		// the pull request is missing. Never re-create the ref, never force it.
		journal.Commit = commit
		result.Commit = commit
		if err := advanceJournal(ctx, writeStore, journalRelative, &journal, journalRefCreated, "verified the existing candidate branch; the draft pull request is missing"); err != nil {
			return SubmitResult{}, err
		}
	} else {
		baseObject, err := endpoint.commitObject(ctx, candidate.Target.BaseCommit)
		if err != nil {
			return SubmitResult{}, err
		}
		if baseObject.SHA != candidate.Target.BaseCommit {
			return SubmitResult{}, fmt.Errorf("the remote repository reports a different object id for the pinned base commit, so its object format cannot be reconciled with the local upstream; refusing")
		}
		if err := assertRemoteTreeMatches(ctx, endpoint, baseObject.Tree.SHA, baseEntries); err != nil {
			return SubmitResult{}, err
		}

		// The journal exists before the first write, so an interrupted process leaves a
		// record of what may have been created. The write is atomic (temporary file
		// plus rename), which is a process-interruption guarantee, not a claim about
		// machine power loss.
		if err := advanceJournal(ctx, writeStore, journalRelative, &journal, journalStarted, "first remote write pending"); err != nil {
			return SubmitResult{}, err
		}

		fileEntries := make([]remoteTreeEntry, 0, len(candidate.Files))
		for _, file := range payload.Files {
			if err := ctx.Err(); err != nil {
				return uncertain(result, journalRelative, err)
			}
			expectedBlob, err := repo.BlobID(ctx, []byte(file.Content))
			if err != nil {
				return SubmitResult{}, err
			}
			blob, err := endpoint.createBlob(ctx, []byte(file.Content))
			result.Writes++
			if err != nil {
				return wrapUncertain(result, journalRelative, err)
			}
			if blob != expectedBlob {
				return SubmitResult{}, fmt.Errorf("GitHub stored %s with blob id %s, but those exact bytes hash to %s locally; refusing to continue", safeQuote(file.Path), blob, expectedBlob)
			}
			fileEntries = append(fileEntries, remoteTreeEntry{Path: file.Path, Mode: expected[file.Path].Mode, Type: "blob", SHA: blob})
		}
		tree, err := endpoint.createTree(ctx, baseObject.Tree.SHA, fileEntries)
		result.Writes++
		if err != nil {
			return wrapUncertain(result, journalRelative, err)
		}
		journal.Tree = tree
		result.Tree = tree
		if err := advanceJournal(ctx, writeStore, journalRelative, &journal, journalObjectsCreated, "tree created"); err != nil {
			return SubmitResult{}, err
		}
		commit, err = endpoint.createCommit(ctx, payload.Commit, tree, []string{payload.BaseCommit})
		result.Writes++
		if err != nil {
			return wrapUncertain(result, journalRelative, err)
		}
		journal.Commit = commit
		result.Commit = commit
		created, err := endpoint.commitObject(ctx, commit)
		if err != nil {
			return wrapUncertain(result, journalRelative, err)
		}
		if err := endpoint.verifyRealizedCommit(ctx, repo, created, payload, payload.BaseCommit, expected); err != nil {
			return SubmitResult{}, err
		}

		status, err := endpoint.createRef(ctx, payload.Branch, commit)
		result.Writes++
		if err != nil {
			return wrapUncertain(result, journalRelative, err)
		}
		switch status {
		case http.StatusCreated:
		case http.StatusUnprocessableEntity:
			// A ref already exists: adopt it only if it is exactly this commit.
			current, err := endpoint.ref(ctx, "refs/heads/"+refPath(payload.Branch))
			if err != nil {
				return wrapUncertain(result, journalRelative, err)
			}
			if current == nil || current.Object.SHA != commit {
				return SubmitResult{}, fmt.Errorf("branch %s already exists and does not point at this candidate's commit; zt never force-updates a ref", safeQuote(payload.Branch))
			}
		default:
			return SubmitResult{}, fmt.Errorf("GitHub returned HTTP %d creating branch %s", status, safeQuote(payload.Branch))
		}
		if err := advanceJournal(ctx, writeStore, journalRelative, &journal, journalRefCreated, "branch created"); err != nil {
			return SubmitResult{}, err
		}
	}

	pull, status, err := endpoint.createPull(ctx, payload.PullRequest, payload.Branch, baseBranch)
	result.Writes++
	if err != nil {
		return wrapUncertain(result, journalRelative, err)
	}
	switch status {
	case http.StatusCreated:
		if err := verifyPull(pull, payload, baseBranch, commit); err != nil {
			return SubmitResult{}, err
		}
	case http.StatusUnprocessableEntity:
		pull, err = adoptPull(ctx, repo, endpoint, payload, expected, baseBranch)
		if err != nil {
			return SubmitResult{}, err
		}
	default:
		return SubmitResult{}, fmt.Errorf("GitHub returned HTTP %d creating the draft pull request", status)
	}

	journal.PullRequest = &JournalPullRequest{Number: pull.Number, URL: pull.HTMLURL}
	if err := advanceJournal(ctx, writeStore, journalRelative, &journal, journalCompleted, "draft pull request created"); err != nil {
		return SubmitResult{}, err
	}
	result.Action = "created"
	result.State = "completed"
	result.Publication = publicationSubmitted
	result.PullRequest = &SubmitPullRequest{Number: pull.Number, URL: pull.HTMLURL, State: pull.State, Draft: pull.Draft}
	result.Notes = append(result.Notes, "draft pull request created; merging, publication and activation remain human decisions")
	return result, nil
}

// finishExisting records and reports an exactly matching pull request that was
// already there. It writes only the local journal, never the remote side.
func finishExisting(ctx context.Context, store *Store, journalRelative string, result SubmitResult, journal *Journal, pull remotePull) (SubmitResult, error) {
	journal.PullRequest = &JournalPullRequest{Number: pull.Number, URL: pull.HTMLURL}
	journal.Commit = pull.Head.SHA
	if err := advanceJournal(ctx, store, journalRelative, journal, journalCompleted, "existing exact submission adopted without writes"); err != nil {
		return SubmitResult{}, err
	}
	result.Action = "existing"
	result.State = "completed"
	result.Publication = publicationSubmitted
	result.Commit = pull.Head.SHA
	result.PullRequest = &SubmitPullRequest{Number: pull.Number, URL: pull.HTMLURL, State: pull.State, Draft: pull.Draft}
	result.Notes = append(result.Notes, "the existing pull request was recognized; no remote writes were made")
	return result, nil
}

func uncertain(result SubmitResult, journalRelative string, err error) (SubmitResult, error) {
	result.State = "uncertain"
	result.Publication = publicationNotReady
	result.JournalPath = path.Join(artifactRoot, journalRelative)
	return result, err
}

// wrapUncertain turns a remote failure into an actionable instruction: reconcile by
// reading rather than retrying blindly.
func wrapUncertain(result SubmitResult, journalRelative string, err error) (SubmitResult, error) {
	if errors.Is(err, errRemoteUncertain) {
		return uncertain(result, journalRelative, fmt.Errorf("%w; the journal %s records how far this attempt got. Re-run the same command to reconcile by reading before any new write", err, path.Join(artifactRoot, journalRelative)))
	}
	return uncertain(result, journalRelative, err)
}

// expectedTree is the complete tree the payload must produce: the pinned base with
// exactly the candidate's files replaced. Reconciliation compares this whole
// set, so extra, missing or rewritten unrelated files are all refused.
type treeFile struct {
	Mode string
	SHA  string
}

func expectedTree(ctx context.Context, repo *Repo, baseEntries []Entry, candidate Candidate) (map[string]treeFile, error) {
	replaced := make(map[string]bool, len(candidate.Files))
	for _, file := range candidate.Files {
		replaced[file.Path] = true
	}
	base := make(map[string]Entry, len(baseEntries))
	for _, entry := range baseEntries {
		// Every base entry is checked, including a replaced one: a replacement inherits
		// its base mode, so the mode must be one zt is willing to publish.
		if entry.Mode != "100644" && entry.Mode != "100755" {
			return nil, fmt.Errorf("pinned base entry %s has mode %s, which zt refuses to reason about when verifying a remote tree", safeQuote(entry.Path), entry.Mode)
		}
		base[entry.Path] = entry
	}
	expected := make(map[string]treeFile, len(baseEntries)+len(candidate.Files))
	for _, entry := range baseEntries {
		if replaced[entry.Path] {
			continue
		}
		expected[entry.Path] = treeFile{Mode: entry.Mode, SHA: entry.SHA}
	}
	for _, file := range candidate.Files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		blob, err := repo.BlobID(ctx, []byte(file.Content))
		if err != nil {
			return nil, err
		}
		mode := "100644"
		if existing, ok := base[file.Path]; ok {
			// A replacement carries content only, so it keeps the mode the pinned base
			// used; a new path keeps the existing 100644 default. The verification workspace
			// applies the same rule, so the tree zt submits and the tree it materialised
			// for the checks cannot disagree about a replaced file's mode.
			mode = existing.Mode
		}
		expected[file.Path] = treeFile{Mode: mode, SHA: blob}
	}
	return expected, nil
}

// assertRemoteTreeMatches proves the remote base commit holds exactly the pinned
// local content, so a submission cannot silently overlay files onto a tree zt never
// inspected.
func assertRemoteTreeMatches(ctx context.Context, endpoint *githubEndpoint, treeSHA string, baseEntries []Entry) error {
	remote, err := endpoint.treeEntries(ctx, treeSHA)
	if err != nil {
		return err
	}
	if len(remote) != len(baseEntries) {
		return fmt.Errorf("the remote base tree holds %d files but the pinned local base holds %d; refusing to submit against a different tree", len(remote), len(baseEntries))
	}
	for _, entry := range baseEntries {
		value, ok := remote[entry.Path]
		if !ok {
			return fmt.Errorf("the remote base tree is missing %s, which the pinned local base holds; refusing to submit", safeQuote(entry.Path))
		}
		if value.SHA != entry.SHA || value.Mode != entry.Mode {
			return fmt.Errorf("the remote base tree holds a different %s than the pinned local base; refusing to submit", safeQuote(entry.Path))
		}
	}
	return nil
}

// verifyRealizedCommit checks one realized commit against the payload by
// rebuilding the exact Git commit object the payload names and comparing object ids.
//
// The object id decides, not the REST representation: GitHub returns `message` with
// one trailing newline removed, while the stored object keeps the exact payload
// bytes, so a presentation difference is acceptable only when the reconstructed
// object hashes to the observed commit. Parents, tree identity and the author and
// committer identity and date are checked separately, and any additional header in
// the remote object would change its id and fail closed. The full tree is verified
// afterwards.
func (e *githubEndpoint) verifyRealizedCommit(ctx context.Context, repo *Repo, commit remoteCommit, payload PublicPayload, baseCommit string, expected map[string]treeFile) error {
	if len(commit.Parents) != 1 || commit.Parents[0].SHA != baseCommit {
		return fmt.Errorf("commit %s does not have exactly the pinned base commit as its parent; refusing", safeQuote(commit.SHA))
	}
	if commit.Tree.SHA == "" {
		return fmt.Errorf("commit %s does not report a tree", safeQuote(commit.SHA))
	}
	if err := verifyIdentity(commit.Author, payload.Commit); err != nil {
		return err
	}
	if err := verifyIdentity(commit.Committer, payload.Commit); err != nil {
		return err
	}
	object, err := rawCommitObject(commit.Tree.SHA, baseCommit, payload.Commit)
	if err != nil {
		return err
	}
	expectedID, err := repo.CommitID(ctx, object)
	if err != nil {
		return err
	}
	if expectedID != commit.SHA {
		return fmt.Errorf("the remote commit %s is not exactly the commit the payload describes (its raw object hashes to %s); zt compares the Git object, not the API's message view", safeQuote(commit.SHA), expectedID)
	}
	return e.verifyRealizedTree(ctx, commit.Tree.SHA, expected)
}

// rawCommitObject renders the exact Git commit object the payload names:
// one tree, one parent, the fixed author and committer identity and date, and the
// message bytes verbatim. No newline is added, trimmed or normalised.
func rawCommitObject(tree, parent string, commit PayloadCommit) ([]byte, error) {
	instant, err := time.Parse(time.RFC3339, commit.CommittedAt)
	if err != nil {
		return nil, fmt.Errorf("payload commit date is not an RFC 3339 instant: %w", err)
	}
	seconds := instant.UTC().Unix()
	var object bytes.Buffer
	fmt.Fprintf(&object, "tree %s\nparent %s\n", tree, parent)
	fmt.Fprintf(&object, "author %s <%s> %d +0000\n", commit.AuthorName, commit.AuthorEmail, seconds)
	fmt.Fprintf(&object, "committer %s <%s> %d +0000\n", commit.AuthorName, commit.AuthorEmail, seconds)
	object.WriteByte('\n')
	object.WriteString(commit.Message)
	return object.Bytes(), nil
}

func verifyIdentity(identity remoteIdentity, commit PayloadCommit) error {
	if identity.Name != commit.AuthorName || identity.Email != commit.AuthorEmail {
		return fmt.Errorf("commit identity %s <%s> is not the zt-evolve identity the payload names", safeQuote(identity.Name), safeQuote(identity.Email))
	}
	date, err := time.Parse(time.RFC3339, identity.Date)
	if err != nil {
		return fmt.Errorf("commit date %s is not an RFC 3339 instant", safeQuote(identity.Date))
	}
	expected, err := time.Parse(time.RFC3339, commit.CommittedAt)
	if err != nil || !date.Equal(expected) {
		return fmt.Errorf("commit date does not equal the pinned base committer instant in the payload")
	}
	return nil
}

// verifyRealizedTree compares the full remote tree with the expected set.
func (e *githubEndpoint) verifyRealizedTree(ctx context.Context, treeSHA string, expected map[string]treeFile) error {
	remote, err := e.treeEntries(ctx, treeSHA)
	if err != nil {
		return err
	}
	if len(remote) != len(expected) {
		return fmt.Errorf("the remote tree holds %d files but the publication has %d; refusing to call this an exact match", len(remote), len(expected))
	}
	paths := make([]string, 0, len(expected))
	for name := range expected {
		paths = append(paths, name)
	}
	sort.Strings(paths)
	for _, name := range paths {
		value, ok := remote[name]
		if !ok {
			return fmt.Errorf("the remote tree is missing %s from the publication", safeQuote(name))
		}
		if value.SHA != expected[name].SHA || value.Mode != expected[name].Mode {
			return fmt.Errorf("the remote tree holds a different %s than the publication", safeQuote(name))
		}
	}
	return nil
}

// verifyPull requires every field that identifies this submission: this branch of
// this repository as the head, the default branch of this repository as the base,
// an open pull request and the exact head commit. A title or body the operator has
// since edited, and a draft that was marked ready for review, are review decisions
// rather than identity: zt recognises such a pull request as its own and never
// rewrites it.
func verifyPull(pull remotePull, payload PublicPayload, baseBranch, headSHA string) error {
	if pull.Head.Ref != payload.Branch {
		return fmt.Errorf("pull request %d has head branch %s instead of %s; refusing to treat it as this submission", pull.Number, safeQuote(pull.Head.Ref), safeQuote(payload.Branch))
	}
	if pull.Head.Repo == nil || !equalRepository(pull.Head.Repo.FullName, payload.Repository) {
		return fmt.Errorf("pull request %d comes from a different head repository; refusing to treat it as this submission", pull.Number)
	}
	if pull.Base.Ref != baseBranch {
		return fmt.Errorf("pull request %d targets %s instead of the default branch %s", pull.Number, safeQuote(pull.Base.Ref), safeQuote(baseBranch))
	}
	if pull.Base.Repo != nil && !equalRepository(pull.Base.Repo.FullName, payload.Repository) {
		return fmt.Errorf("pull request %d targets a different repository", pull.Number)
	}
	if pull.State != "open" {
		return fmt.Errorf("pull request %d is %s, not open; zt never reopens or reuses a closed pull request", pull.Number, safeQuote(pull.State))
	}
	if headSHA != "" && pull.Head.SHA != headSHA {
		return fmt.Errorf("pull request %d points at commit %s instead of the expected %s; that is not this candidate", pull.Number, pull.Head.SHA, headSHA)
	}
	return nil
}

func equalRepository(left, right string) bool {
	return strings.EqualFold(left, right)
}

// reconcileOutcome is what a read-only reconciliation found: an exact existing pull
// request, or a branch that already holds exactly this candidate's commit with only the
// pull request missing.
type reconcileOutcome struct {
	pull *remotePull
	head string
}

// reconcile inspects the deterministic branch and returns an existing open pull
// request that carries this candidate's exact commit, or a verified head that is
// missing only its pull request. Any existing but non-matching object is refused
// instead of adopted, so an unknown object can never be reported as this submission.
func reconcile(ctx context.Context, repo *Repo, endpoint *githubEndpoint, payload PublicPayload, expected map[string]treeFile, baseBranch string) (reconcileOutcome, error) {
	pulls, err := endpoint.pullsForBranch(ctx, payload.Branch)
	if err != nil {
		return reconcileOutcome{}, err
	}
	var open, closed []remotePull
	for _, pull := range pulls {
		if pull.State == "open" {
			open = append(open, pull)
			continue
		}
		closed = append(closed, pull)
	}
	if len(open) == 0 && len(closed) > 0 {
		return reconcileOutcome{}, fmt.Errorf("a closed pull request already exists for branch %s; zt never reopens a closed proposal. Prepare a new candidate instead", safeQuote(payload.Branch))
	}
	if len(open) > 1 {
		return reconcileOutcome{}, fmt.Errorf("more than one open pull request uses branch %s; zt cannot tell which one is this submission", safeQuote(payload.Branch))
	}
	if len(open) == 1 {
		pull := open[0]
		commit, err := endpoint.commitObject(ctx, pull.Head.SHA)
		if err != nil {
			return reconcileOutcome{}, err
		}
		if err := endpoint.verifyRealizedCommit(ctx, repo, commit, payload, payload.BaseCommit, expected); err != nil {
			return reconcileOutcome{}, err
		}
		if err := verifyPull(pull, payload, baseBranch, pull.Head.SHA); err != nil {
			return reconcileOutcome{}, err
		}
		return reconcileOutcome{pull: &pull}, nil
	}

	// No pull request exists. The branch may still do: a lost response after the ref
	// was created leaves exactly that state, and the required recovery is to verify
	// the real object and create only the missing pull request.
	ref, err := endpoint.ref(ctx, "refs/heads/"+refPath(payload.Branch))
	if err != nil {
		return reconcileOutcome{}, err
	}
	if ref == nil {
		return reconcileOutcome{}, nil
	}
	commit, err := endpoint.commitObject(ctx, ref.Object.SHA)
	if err != nil {
		return reconcileOutcome{}, err
	}
	if err := endpoint.verifyRealizedCommit(ctx, repo, commit, payload, payload.BaseCommit, expected); err != nil {
		return reconcileOutcome{}, fmt.Errorf("branch %s exists but does not hold exactly this candidate's commit: %w. zt never reuses or force-updates it", safeQuote(payload.Branch), err)
	}
	return reconcileOutcome{head: ref.Object.SHA}, nil
}

// adoptPull re-reads the pull request after a create conflict and accepts it only
// when it verifies exactly.
func adoptPull(ctx context.Context, repo *Repo, endpoint *githubEndpoint, payload PublicPayload, expected map[string]treeFile, baseBranch string) (remotePull, error) {
	pulls, err := endpoint.pullsForBranch(ctx, payload.Branch)
	if err != nil {
		return remotePull{}, err
	}
	for _, pull := range pulls {
		if pull.State != "open" {
			continue
		}
		commit, err := endpoint.commitObject(ctx, pull.Head.SHA)
		if err != nil {
			return remotePull{}, err
		}
		if err := endpoint.verifyRealizedCommit(ctx, repo, commit, payload, payload.BaseCommit, expected); err != nil {
			return remotePull{}, err
		}
		if err := verifyPull(pull, payload, baseBranch, pull.Head.SHA); err != nil {
			return remotePull{}, err
		}
		return pull, nil
	}
	return remotePull{}, fmt.Errorf("GitHub refused to create the pull request and no exactly matching open pull request exists; inspect branch %s before retrying", safeQuote(payload.Branch))
}
