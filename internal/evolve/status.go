package evolve

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/zeithrold/tools/internal/githubauth"
)

// StatusOptions selects what status observes. Remote is the only opt-in that
// reaches the network and the credential store; without it status is a local,
// read-only report.
type StatusOptions struct {
	Candidate  string
	Policy     string
	Upstream   string
	Validation string
	Root       string
	Remote     bool
	OpenAPI    func(ctx context.Context, target Target) (githubauth.API, error)
}

// StatusCheck is the local verification record status found, if any.
type StatusCheck struct {
	ValidationDigest string `json:"validationDigest"`
	Status           string `json:"status"`
	StartedAt        string `json:"startedAt,omitempty"`
	WorkspaceChanged bool   `json:"workspaceChanged"`
}

// StatusJournal is the observed submission journal.
type StatusJournal struct {
	State       string              `json:"state"`
	UpdatedAt   string              `json:"updatedAt,omitempty"`
	BaseBranch  string              `json:"baseBranch,omitempty"`
	Commit      string              `json:"commit,omitempty"`
	PullRequest *JournalPullRequest `json:"pullRequest,omitempty"`
}

// StatusCheckRuns summarizes the checks GitHub reports for the head commit. These
// are platform observations, not zt execution.
type StatusCheckRuns struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Failure int `json:"failure"`
	Pending int `json:"pending"`
}

// StatusRemote is what an explicit remote read observed.
type StatusRemote struct {
	DefaultBranch string             `json:"defaultBranch,omitempty"`
	PullRequest   *SubmitPullRequest `json:"pullRequest,omitempty"`
	HeadSHA       string             `json:"headSha,omitempty"`
	BaseSHA       string             `json:"baseSha,omitempty"`
	CheckRuns     *StatusCheckRuns   `json:"checkRuns,omitempty"`
	Drift         []string           `json:"drift,omitempty"`
}

// StatusResult reports local observations and, only when asked, reconciled remote
// ones. Publication stays not_ready unless a submission journal records a completed
// draft pull request; even then nothing is merged, published or activated.
type StatusResult struct {
	SchemaVersion    int            `json:"schemaVersion"`
	CandidateID      string         `json:"candidateId"`
	Skill            string         `json:"skill"`
	Target           Target         `json:"target"`
	Branch           string         `json:"branch"`
	ValidationDigest string         `json:"validationDigest,omitempty"`
	Check            *StatusCheck   `json:"check,omitempty"`
	Journal          *StatusJournal `json:"journal,omitempty"`
	Publication      string         `json:"publication"`
	Remote           *StatusRemote  `json:"remote,omitempty"`
	Warnings         []string       `json:"warnings,omitempty"`
}

// Status revalidates one candidate locally and reports what zt can see about it.
//
// The default path reads no credential, opens no connection and writes nothing: the
// owned artifact tree is opened only if it already exists. Remote reads require an
// explicit opt-in, use the same authenticated endpoint as submit, and never write,
// rewrite or "fix" anything they observe.
func Status(ctx context.Context, options StatusOptions) (StatusResult, error) {
	if err := ctx.Err(); err != nil {
		return StatusResult{}, err
	}
	policy, err := LoadPolicy(options.Policy)
	if err != nil {
		return StatusResult{}, err
	}
	data, id, err := readCandidate(options.Candidate)
	if err != nil {
		return StatusResult{}, err
	}
	var candidate Candidate
	if err := decodeDocument(data, &candidate, "candidate"); err != nil {
		return StatusResult{}, err
	}
	canonical, err := candidate.canonicalBytes()
	if err != nil {
		return StatusResult{}, fmt.Errorf("encode candidate: %w", err)
	}
	if !bytes.Equal(canonical, data) {
		return StatusResult{}, fmt.Errorf("candidate: bytes are not canonical, so the file was modified after preparation")
	}
	repo, err := OpenRepo(ctx, options.Upstream)
	if err != nil {
		return StatusResult{}, err
	}
	if err := validateCandidate(ctx, repo, policy, candidate); err != nil {
		return StatusResult{}, err
	}
	result := StatusResult{
		SchemaVersion: schemaVersion,
		CandidateID:   id,
		Skill:         candidate.Skill,
		Target:        candidate.Target,
		Branch:        branchFor(candidate.Skill, id),
		Publication:   publicationNotReady,
	}

	var payload *PublicPayload
	if strings.TrimSpace(options.Validation) != "" {
		validation, err := LoadValidation(options.Validation)
		if err != nil {
			return StatusResult{}, err
		}
		if err := validation.covers(policy); err != nil {
			return StatusResult{}, err
		}
		validationDigest, err := validation.digest()
		if err != nil {
			return StatusResult{}, err
		}
		result.ValidationDigest = validationDigest
	}

	store, err := openExistingStore(options.Root)
	if err != nil {
		return StatusResult{}, err
	}
	defer store.Close()
	if result.ValidationDigest != "" {
		if check, ok, err := observeCheck(store, id, result.ValidationDigest); err != nil {
			return StatusResult{}, err
		} else if ok {
			result.Check = check
		}
	}
	if journal, ok, err := observeJournal(store, id); err != nil {
		return StatusResult{}, err
	} else if ok {
		result.Journal = journal
		if journal.State == journalCompleted && journal.PullRequest != nil {
			result.Publication = publicationSubmitted
		}
	}

	if !options.Remote {
		return result, nil
	}

	info, err := repo.CommitInfo(ctx, candidate.Target.BaseCommit)
	if err != nil {
		return StatusResult{}, err
	}
	if strings.TrimSpace(options.Validation) != "" {
		validation, err := LoadValidation(options.Validation)
		if err != nil {
			return StatusResult{}, err
		}
		built, err := buildPayload(candidate, id, validation, result.ValidationDigest, info)
		if err != nil {
			return StatusResult{}, err
		}
		payload = &built
	}
	if options.OpenAPI == nil {
		return StatusResult{}, fmt.Errorf("remote status needs an explicit GitHub opt-in and the githubauth credential seam, which this build did not provide")
	}
	api, err := options.OpenAPI(ctx, candidate.Target)
	if err != nil {
		return StatusResult{}, err
	}
	endpoint, err := newGitHubEndpoint(api, candidate.Target)
	if err != nil {
		return StatusResult{}, err
	}
	remote, err := observeRemote(ctx, endpoint, candidate, result.Branch, payload, result.Journal)
	if err != nil {
		return StatusResult{}, err
	}
	result.Remote = remote
	return result, nil
}

// observeCheck reports the stored verification record without trusting it as more
// than a local file.
func observeCheck(store *Store, candidateID, validationDigest string) (*StatusCheck, bool, error) {
	data, found, err := store.readOwned(path.Join(checkRunRelative(candidateID, validationDigest), "result.json"), maxCandidateBytes)
	if err != nil || !found {
		return nil, false, err
	}
	var result CheckResult
	if err := decodeDocument(data, &result, "verification result"); err != nil {
		return nil, false, nil
	}
	if result.CandidateID != candidateID || result.ValidationDigest != validationDigest {
		return nil, false, nil
	}
	return &StatusCheck{
		ValidationDigest: result.ValidationDigest,
		Status:           result.Status,
		StartedAt:        result.StartedAt,
		WorkspaceChanged: result.WorkspaceChanged,
	}, true, nil
}

// observeJournal reports the newest journal recorded for this candidate, whatever
// payload it names: status describes the candidate, not one attempt.
func observeJournal(store *Store, candidateID string) (*StatusJournal, bool, error) {
	entries, err := store.listOwned(submitRecordRoot)
	if err != nil {
		return nil, false, err
	}
	var chosen *Journal
	for _, name := range entries {
		if !strings.HasPrefix(name, candidateID+"-") || !strings.HasSuffix(name, ".json") {
			continue
		}
		data, found, err := store.readOwned(path.Join(submitRecordRoot, name), maxJournalBytes)
		if err != nil || !found {
			continue
		}
		var journal Journal
		if err := decodeDocument(data, &journal, "submission journal"); err != nil {
			continue
		}
		if journal.CandidateID != candidateID {
			continue
		}
		if chosen == nil || journal.UpdatedAt > chosen.UpdatedAt {
			copy := journal
			chosen = &copy
		}
	}
	if chosen == nil {
		return nil, false, nil
	}
	return &StatusJournal{State: chosen.State, UpdatedAt: chosen.UpdatedAt, BaseBranch: chosen.BaseBranch, Commit: chosen.Commit, PullRequest: chosen.PullRequest}, true, nil
}

// observeRemote reads the deterministic branch, its pull request and the checks
// GitHub reports for the head. Differences are collected as drift and reported;
// nothing is rewritten and no exact match is claimed without reading the objects.
func observeRemote(ctx context.Context, endpoint *githubEndpoint, candidate Candidate, branch string, payload *PublicPayload, journal *StatusJournal) (*StatusRemote, error) {
	repository, err := endpoint.repository(ctx)
	if err != nil {
		return nil, err
	}
	remote := &StatusRemote{DefaultBranch: repository.DefaultBranch}

	ref, err := endpoint.ref(ctx, "refs/heads/"+refPath(branch))
	if err != nil {
		return nil, err
	}
	if ref != nil {
		remote.HeadSHA = ref.Object.SHA
	}
	pulls, err := endpoint.pullsForBranch(ctx, branch)
	if err != nil {
		return nil, err
	}
	var chosen *remotePull
	for i := range pulls {
		if pulls[i].State == "open" {
			chosen = &pulls[i]
			break
		}
	}
	if chosen == nil && len(pulls) > 0 {
		chosen = &pulls[0]
		remote.Drift = append(remote.Drift, "pull-request-closed-or-merged")
	}
	if chosen != nil {
		remote.PullRequest = &SubmitPullRequest{Number: chosen.Number, URL: chosen.HTMLURL, State: chosen.State, Draft: chosen.Draft}
		remote.HeadSHA = chosen.Head.SHA
		remote.BaseSHA = chosen.Base.SHA
		switch {
		case chosen.State != "open":
		case !chosen.Draft:
			remote.Drift = append(remote.Drift, "pull-request-not-draft")
		}
		if chosen.Base.Ref != repository.DefaultBranch {
			remote.Drift = append(remote.Drift, "base-branch-changed")
		}
		if chosen.Head.Repo == nil || !equalRepository(chosen.Head.Repo.FullName, candidate.Target.Repository) {
			remote.Drift = append(remote.Drift, "head-repository-changed")
		}
		if payload != nil {
			// Exact bytes: any difference in what a reader sees is drift, and status never
			// rewrites it.
			if chosen.Title != payload.PullRequest.Title || chosen.Body != payload.PullRequest.Body {
				remote.Drift = append(remote.Drift, "public-text-changed")
			}
		}
	}
	if journal != nil && journal.Commit != "" && remote.HeadSHA != "" && journal.Commit != remote.HeadSHA {
		remote.Drift = append(remote.Drift, "head-commit-changed")
	}
	head, err := endpoint.ref(ctx, "refs/heads/"+refPath(repository.DefaultBranch))
	if err != nil {
		return nil, err
	}
	if head != nil && head.Object.SHA != candidate.Target.BaseCommit {
		remote.Drift = append(remote.Drift, "base-branch-moved-past-pinned-base")
	}
	if remote.HeadSHA != "" {
		runs, found, err := endpoint.checkRuns(ctx, remote.HeadSHA)
		if err != nil {
			return nil, err
		}
		if !found {
			remote.Drift = append(remote.Drift, "check-runs-unavailable")
		} else {
			summary := &StatusCheckRuns{Total: runs.TotalCount}
			for _, run := range runs.CheckRuns {
				switch {
				case run.Status != "completed":
					summary.Pending++
				case run.Conclusion == "success":
					summary.Success++
				case run.Conclusion == "":
					summary.Pending++
				default:
					summary.Failure++
				}
			}
			remote.CheckRuns = summary
		}
	}
	return remote, nil
}
