package evolve

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/zeithrold/tools/internal/githubauth"
)

// seedUpstreamCache uses native Git, not the fetch implementation, to establish
// the offline consumer's expected complete bare-object baseline.
func seedUpstreamCache(t *testing.T, h *harness) string {
	t.Helper()
	h.writePolicy(policyDoc(h.commit))
	store, err := openStore(h.workspace)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	policy, err := LoadPolicy(h.policy)
	if err != nil {
		t.Fatal(err)
	}
	relative := upstreamRelative(policy.target())
	if err := store.evolve.MkdirAll(relative, 0700); err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(store.dir, filepath.FromSlash(relative), "repo.git")
	runGit(t, h.upstream, "clone", "--quiet", "--bare", "--no-hardlinks", h.upstream, cache)
	data, err := canonicalJSON(policy.target())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.evolve.WriteFile(path.Join(relative, "target.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	return cache
}

func TestUpstreamReadinessIsReadOnly(t *testing.T) {
	h := newHarness(t)
	h.writePolicy(policyDoc(h.commit))
	before := snapshotTree(t, h.workspace)
	result, err := UpstreamStatus(context.Background(), h.policy, h.workspace)
	if err != nil || result.Status != "missing" || result.Authentication != "not_checked" || result.Checks != "not_run" || len(result.NextCommand) == 0 {
		t.Fatalf("missing readiness: %+v %v", result, err)
	}
	if !reflect.DeepEqual(before, snapshotTree(t, h.workspace)) {
		t.Fatal("readiness created cache directories")
	}
	seedUpstreamCache(t, h)
	before = snapshotTree(t, h.workspace)
	result, err = UpstreamStatus(context.Background(), h.policy, h.workspace)
	if err != nil || result.Status != "ready" || result.Created || len(result.NextCommand) != 0 {
		t.Fatalf("ready: %+v %v", result, err)
	}
	// Even fetch must reuse a complete cache before constructing credentials.
	fetched, err := FetchUpstream(context.Background(), FetchUpstreamOptions{
		Policy: h.policy, Root: h.workspace, AllowNetwork: true,
		OpenGit: func(context.Context, Target) (githubauth.GitEndpoint, error) {
			t.Fatal("cache reuse opened credentials")
			return githubauth.GitEndpoint{}, nil
		},
	})
	if err != nil || fetched.Created || fetched.Status != "ready" {
		t.Fatalf("reuse: %+v %v", fetched, err)
	}
	if !reflect.DeepEqual(before, snapshotTree(t, h.workspace)) {
		t.Fatal("cache inspection/reuse wrote files")
	}
}

func TestCachedBaselineFeedsOfflineLifecycle(t *testing.T) {
	h := newHarness(t)
	options := installedOptions(t, h)
	cache := seedUpstreamCache(t, h)
	writeFile(t, filepath.Join(options.Installed.SkillsDir, fixtureSkill, "SKILL.md"), "# Revised\n")
	beforeUpstream := snapshotTree(t, h.upstream)
	beforeInstall := snapshotTree(t, options.Installed.Root)
	options.Upstream = ""
	result, err := FromInstalled(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	candidate := h.candidatePath(result)
	plan, err := Plan(context.Background(), PlanOptions{Candidate: candidate, Policy: h.policy, Root: h.workspace})
	if err != nil || plan.CandidateID != result.CandidateID {
		t.Fatalf("cached plan: %+v %v", plan, err)
	}
	validation := filepath.Join(t.TempDir(), "validation.json")
	writeJSONFile(t, validation, validationDoc(
		validationCheck("go-test", 60, helperArgv("TestEvolveHelperPass")...),
		validationCheck("repo-lint", 60, helperArgv("TestEvolveHelperPass")...),
	))
	check, err := Verify(context.Background(), CheckOptions{
		Candidate: candidate, Policy: h.policy, Root: h.workspace, Validation: validation, TrustedLocalExecution: true,
	})
	if err != nil || check.Status != "passed" {
		t.Fatalf("cached check: %+v %v", check, err)
	}
	// Validation must have received the full repository, not only replacements.
	tree := filepath.Join(h.workspace, filepath.Dir(filepath.FromSlash(check.ResultPath)), "tree")
	if data, err := os.ReadFile(filepath.Join(tree, "README.md")); err != nil || string(data) != baseFileSet()["README.md"] {
		t.Fatalf("missing unrelated baseline file in %s: %q %v", tree, data, err)
	}
	status, err := Status(context.Background(), StatusOptions{Candidate: candidate, Policy: h.policy, Root: h.workspace, Validation: validation})
	if err != nil || status.Skill != fixtureSkill {
		t.Fatalf("cached status: %+v %v", status, err)
	}
	preview, err := Submit(context.Background(), SubmitOptions{Candidate: candidate, Policy: h.policy, Root: h.workspace, ShowPayload: true})
	if err != nil || preview.Action != "payload" {
		t.Fatalf("cached payload preview: %+v %v", preview, err)
	}
	if !reflect.DeepEqual(beforeUpstream, snapshotTree(t, h.upstream)) || !reflect.DeepEqual(beforeInstall, snapshotTree(t, options.Installed.Root)) {
		t.Fatal("cached workflow modified source/installation")
	}
	// The explicit upstream remains independent of the managed cache.
	if err := os.RemoveAll(cache); err != nil {
		t.Fatal(err)
	}
	if _, err := Plan(context.Background(), PlanOptions{Candidate: candidate, Policy: h.policy, Upstream: h.upstream}); err != nil {
		t.Fatalf("explicit upstream compatibility: %v", err)
	}
	before := snapshotTree(t, h.workspace)
	if _, err := Plan(context.Background(), PlanOptions{Candidate: candidate, Policy: h.policy, Root: h.workspace}); err == nil {
		t.Fatal("plan repaired or accepted a damaged cache")
	}
	if !reflect.DeepEqual(before, snapshotTree(t, h.workspace)) {
		t.Fatal("failed plan wrote cache state")
	}
}

func TestUpstreamRejectsCorruptionWithoutRepair(t *testing.T) {
	for _, problem := range []string{"identity", "blob"} {
		t.Run(problem, func(t *testing.T) {
			h := newHarness(t)
			cache := seedUpstreamCache(t, h)
			if problem == "identity" {
				writeFile(t, filepath.Join(filepath.Dir(cache), "target.json"), "{}\n")
			} else {
				blob := runGit(t, cache, "rev-parse", h.commit+":README.md")
				if err := os.Remove(filepath.Join(cache, "objects", blob[:2], blob[2:])); err != nil {
					t.Fatal(err)
				}
			}
			before := snapshotTree(t, h.workspace)
			result, err := UpstreamStatus(context.Background(), h.policy, h.workspace)
			if err != nil || result.Status != "blocked" || result.Problem == "" {
				t.Fatalf("corruption: %+v %v", result, err)
			}
			_, err = FetchUpstream(context.Background(), FetchUpstreamOptions{Policy: h.policy, Root: h.workspace, AllowNetwork: true})
			if err == nil || !strings.Contains(err.Error(), "overwrite") {
				t.Fatalf("corrupt cache fetch: %v", err)
			}
			if !reflect.DeepEqual(before, snapshotTree(t, h.workspace)) {
				t.Fatal("damaged cache was changed")
			}
		})
	}
}

func TestUpstreamRequiresExplicitNetworkConsent(t *testing.T) {
	h := newHarness(t)
	h.writePolicy(policyDoc(h.commit))
	before := snapshotTree(t, h.workspace)
	_, err := FetchUpstream(context.Background(), FetchUpstreamOptions{Policy: h.policy, Root: h.workspace})
	if err == nil || !strings.Contains(err.Error(), "--allow-network") {
		t.Fatalf("missing consent: %v", err)
	}
	if !reflect.DeepEqual(before, snapshotTree(t, h.workspace)) {
		t.Fatal("missing consent wrote cache")
	}
}

func TestUpstreamRejectsSymlinkedCachePaths(t *testing.T) {
	for _, component := range []string{".zt", "artifacts", "upstreams", "repo.git"} {
		t.Run(component, func(t *testing.T) {
			h := newHarness(t)
			cache := seedUpstreamCache(t, h)
			var selected string
			switch component {
			case ".zt":
				selected = filepath.Join(h.workspace, ".zt")
			case "artifacts":
				selected = filepath.Join(h.workspace, ".zt", "artifacts")
			case "upstreams":
				selected = filepath.Join(h.workspace, filepath.FromSlash(artifactRoot), "upstreams")
			default:
				selected = cache
			}
			outside := filepath.Join(t.TempDir(), "moved")
			if err := os.Rename(selected, outside); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, selected); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			before := snapshotTree(t, outside)
			result, err := UpstreamStatus(context.Background(), h.policy, h.workspace)
			if err == nil && result.Status != "blocked" {
				t.Fatalf("symlink accepted: %+v %v", result, err)
			}
			_, err = FetchUpstream(context.Background(), FetchUpstreamOptions{
				Policy: h.policy, Root: h.workspace, AllowNetwork: true,
				OpenGit: func(context.Context, Target) (githubauth.GitEndpoint, error) {
					t.Fatal("symlink reached credentials")
					return githubauth.GitEndpoint{}, nil
				},
			})
			if err == nil {
				t.Fatal("fetch accepted symlink")
			}
			if !reflect.DeepEqual(before, snapshotTree(t, outside)) {
				t.Fatal("readiness/fetch wrote through symlink")
			}
		})
	}
}
