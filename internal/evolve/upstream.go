package evolve

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/zeithrold/tools/internal/githubauth"
)

const upstreamCacheRoot = "upstreams"

// FetchUpstreamOptions opts into acquiring a pinned baseline. The credential
// callback is reached only after local preflight and only with AllowNetwork.
type FetchUpstreamOptions struct {
	Policy       string
	Root         string
	AllowNetwork bool
	OpenGit      func(context.Context, Target) (githubauth.GitEndpoint, error)
}

// UpstreamResult separates complete local objects from authentication and actual
// validation. Ready means a bounded complete baseline, not passing Skill checks.
type UpstreamResult struct {
	SchemaVersion  int      `json:"schemaVersion"`
	Target         Target   `json:"target"`
	Status         string   `json:"status"`
	Upstream       string   `json:"upstream"`
	Created        bool     `json:"created"`
	Authentication string   `json:"authentication"`
	Checks         string   `json:"checks"`
	Problem        string   `json:"problem,omitempty"`
	NextCommand    []string `json:"nextCommand,omitempty"`
}

// upstreamRelative isolates host/repository/commit identities, never names a
// branch, and never puts operator-controlled path components in a cache path.
func upstreamRelative(target Target) string {
	identity := sha256Hex([]byte(target.Host + "\n" + target.Repository))
	return path.Join(upstreamCacheRoot, identity, target.BaseCommit)
}

// UpstreamStatus is a read-only local readiness report. A miss or damaged cache
// never opens credentials, creates a directory or fetches missing objects.
func UpstreamStatus(ctx context.Context, policyFile, root string) (UpstreamResult, error) {
	policy, err := LoadPolicy(policyFile)
	if err != nil {
		return UpstreamResult{}, err
	}
	store, err := openExistingStore(root)
	if err != nil {
		return UpstreamResult{}, err
	}
	defer store.Close()
	result := upstreamResult(policy, store)
	found, err := cachedUpstream(ctx, store, policy.target())
	if ctx.Err() != nil {
		return UpstreamResult{}, ctx.Err()
	}
	if err != nil {
		result.Status, result.Problem = "blocked", err.Error()
	} else if !found {
		result.Status, result.Problem = "missing", "the pinned upstream baseline is not cached"
	} else {
		result.Status = "ready"
	}
	if result.Status != "ready" {
		result.NextCommand = []string{"zt", "evolve", "upstream", "fetch", "--policy", policyFile, "--root", root, "--allow-network"}
	}
	return result, nil
}

func upstreamResult(policy Policy, store *Store) UpstreamResult {
	return UpstreamResult{
		SchemaVersion: schemaVersion, Target: policy.target(),
		Upstream:       filepath.Join(store.dir, filepath.FromSlash(upstreamRelative(policy.target())), "repo.git"),
		Authentication: "not_checked", Checks: checkNotRun,
	}
}

// resolveUpstream preserves the explicit local repository interface. Omitting it
// selects only this workspace's already-complete cache, never an online fallback.
func resolveUpstream(ctx context.Context, explicit, root string, policy Policy) (*Repo, error) {
	if explicit != "" {
		return OpenRepo(ctx, explicit)
	}
	if root == "" {
		root = "."
	}
	store, err := openExistingStore(root)
	if err != nil {
		return nil, err
	}
	defer store.Close()
	found, err := cachedUpstream(ctx, store, policy.target())
	if err != nil {
		return nil, fmt.Errorf("cached upstream is unusable: %w; use an explicit --upstream or inspect `zt evolve upstream status`", err)
	}
	if !found {
		return nil, fmt.Errorf("pinned upstream is not cached; run `zt evolve upstream fetch --policy POLICY --root WORKSPACE --allow-network`, or supply --upstream; no network or credential was used")
	}
	return OpenRepo(ctx, upstreamResult(policy, store).Upstream)
}

func cachedUpstream(ctx context.Context, store *Store, target Target) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if store.evolve == nil {
		return false, nil
	}
	relative := upstreamRelative(target)
	current := ""
	for _, component := range strings.Split(path.Join(relative, "repo.git"), "/") {
		current = path.Join(current, component)
		info, err := store.evolve.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			if current == path.Join(relative, "repo.git") {
				return false, fmt.Errorf("cached upstream has no bare repository")
			}
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return false, fmt.Errorf("cached upstream is not a real directory: %s", safeQuote(current))
		}
	}
	data, found, err := store.readOwned(path.Join(relative, "target.json"), maxPolicyBytes)
	if err != nil {
		return false, err
	}
	if !found {
		return false, fmt.Errorf("cached upstream has no target identity")
	}
	var recorded Target
	if err := decodeDocument(data, &recorded, "cached upstream target"); err != nil {
		return false, err
	}
	if recorded != target {
		return false, fmt.Errorf("cached upstream target does not match the policy")
	}
	repo, err := OpenRepo(ctx, filepath.Join(store.dir, filepath.FromSlash(relative), "repo.git"))
	if err != nil {
		return false, err
	}
	if err := verifyUpstream(ctx, repo, target.BaseCommit); err != nil {
		return false, err
	}
	return true, nil
}

// verifyUpstream verifies object identities/connectivity, not just the existence
// of a commit name, then reads every baseline blob under the native workspace
// limits. A shallow history is allowed; missing baseline trees/blobs are not.
func verifyUpstream(ctx context.Context, repo *Repo, commit string) error {
	if err := repo.VerifyBaseCommit(ctx, commit); err != nil {
		return err
	}
	if _, err := repo.output(ctx, maxGitOutputBytes, "fsck", "--strict", "--no-reflogs", "--no-dangling", commit); err != nil {
		return fmt.Errorf("cached Git object integrity: %w", err)
	}
	if _, err := repo.CommitInfo(ctx, commit); err != nil {
		return err
	}
	entries, err := repo.CompleteTree(ctx, commit)
	if err != nil {
		return err
	}
	total := 0
	for _, entry := range entries {
		if entry.Mode != "100644" && entry.Mode != "100755" {
			return fmt.Errorf("cached baseline entry %s has unsupported mode %s", safeQuote(entry.Path), entry.Mode)
		}
		data, err := repo.ReadEntry(ctx, entry, maxWorkspaceFileBytes, "cached baseline bound")
		if err != nil {
			return err
		}
		total += len(data)
		if total > maxWorkspaceBytes {
			return fmt.Errorf("cached baseline exceeds %d bytes", maxWorkspaceBytes)
		}
	}
	return nil
}

// FetchUpstream publishes a complete immutable bare baseline in the workspace's
// owned artifact tree. It never checks out files, executes validation or writes
// to GitHub. A complete existing entry is reused without opening credentials.
func FetchUpstream(ctx context.Context, options FetchUpstreamOptions) (UpstreamResult, error) {
	if err := ctx.Err(); err != nil {
		return UpstreamResult{}, err
	}
	if !options.AllowNetwork {
		return UpstreamResult{}, fmt.Errorf("upstream fetch requires --allow-network; use upstream status for a credential-free local inspection")
	}
	policy, err := LoadPolicy(options.Policy)
	if err != nil {
		return UpstreamResult{}, err
	}
	// GitHub's Git transport currently uses SHA-1 repositories. Local explicit
	// upstreams retain their existing SHA-256 support.
	if len(policy.BaseCommit) != 40 {
		return UpstreamResult{}, fmt.Errorf("GitHub upstream fetch requires a full SHA-1 commit; use --upstream for a SHA-256 local repository")
	}
	existing, err := openExistingStore(options.Root)
	if err != nil {
		return UpstreamResult{}, err
	}
	found, err := cachedUpstream(ctx, existing, policy.target())
	result := upstreamResult(policy, existing)
	existing.Close()
	if err != nil {
		return UpstreamResult{}, fmt.Errorf("refusing to overwrite unusable cached upstream: %w; supply --upstream or inspect the owned cache before removing it", err)
	}
	if found {
		result.Status = "ready"
		return result, nil
	}
	if options.OpenGit == nil {
		return UpstreamResult{}, fmt.Errorf("GitHub App Git authentication is unavailable")
	}
	endpoint, err := options.OpenGit(ctx, policy.target())
	if err != nil {
		return UpstreamResult{}, err
	}
	result.Authentication = "app_used"
	store, err := openStore(options.Root)
	if err != nil {
		return UpstreamResult{}, err
	}
	defer store.Close()
	parent := path.Dir(upstreamRelative(policy.target()))
	if err := store.evolve.MkdirAll(parent, 0700); err != nil {
		return UpstreamResult{}, err
	}
	stage := path.Join(parent, ".fetch-"+rand.Text())
	if err := store.evolve.Mkdir(stage, 0700); err != nil {
		return UpstreamResult{}, err
	}
	defer store.evolve.RemoveAll(stage)
	stageDir := filepath.Join(store.dir, filepath.FromSlash(stage))
	if err := fetchGitBaseline(ctx, filepath.Join(stageDir, "repo.git"), policy.target(), endpoint); err != nil {
		return UpstreamResult{}, err
	}
	repo, err := OpenRepo(ctx, filepath.Join(stageDir, "repo.git"))
	if err != nil {
		return UpstreamResult{}, err
	}
	if err := verifyUpstream(ctx, repo, policy.BaseCommit); err != nil {
		return UpstreamResult{}, err
	}
	data, err := canonicalJSON(policy.target())
	if err != nil {
		return UpstreamResult{}, err
	}
	if err := store.evolve.WriteFile(path.Join(stage, "target.json"), data, 0600); err != nil {
		return UpstreamResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return UpstreamResult{}, err
	}
	if err := store.evolve.Rename(stage, upstreamRelative(policy.target())); err != nil {
		// Concurrent fetches stage independently. A winner is adopted only after
		// independently verifying its whole baseline; no cache is overwritten.
		ready, checkErr := cachedUpstream(ctx, store, policy.target())
		if checkErr != nil {
			return UpstreamResult{}, fmt.Errorf("verify concurrently published upstream: %w", checkErr)
		}
		if ready {
			result.Status = "ready"
			return result, nil
		}
		return UpstreamResult{}, fmt.Errorf("install cached upstream: %w", err)
	}
	result.Status, result.Created, result.Authentication = "ready", true, "app_used"
	return result, nil
}
