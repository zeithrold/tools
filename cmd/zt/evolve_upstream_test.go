package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/zeithrold/tools/internal/evolve"
	"github.com/zeithrold/tools/internal/githubauth"
)

func runOfflineUpstreamCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	stdout := &bytes.Buffer{}
	a := newApp(stdout, &bytes.Buffer{})
	a.homeDir = func() (string, error) {
		t.Fatal("cache command resolved home")
		return "", nil
	}
	a.openGit = func(context.Context, string, string, string, string) (githubauth.GitEndpoint, error) {
		t.Fatal("offline/cache reuse opened Git credential")
		return githubauth.GitEndpoint{}, nil
	}
	a.openAPI = func(context.Context, string, string, string, string) (githubauth.API, error) {
		t.Fatal("offline/cache reuse opened API credential")
		return githubauth.API{}, nil
	}
	a.newAuth = func(string) (authService, error) {
		t.Fatal("offline/cache reuse opened auth service")
		return nil, nil
	}
	err := executeApp(a, context.Background(), args)
	return stdout.String(), err
}

func TestUpstreamCLIReadinessAndCachedPrepare(t *testing.T) {
	f := newEvolveFixture(t)
	args := []string{"evolve", "upstream", "status", "--policy", f.policy, "--root", f.root, "--json"}
	before := snapshotFiles(t, f.root)
	text, err := runOfflineUpstreamCLI(t, args...)
	if err != nil {
		t.Fatal(err)
	}
	var readiness evolve.UpstreamResult
	if err := json.Unmarshal([]byte(text), &readiness); err != nil {
		t.Fatal(err)
	}
	if readiness.Status != "missing" || readiness.Authentication != "not_checked" || readiness.Checks != "not_run" ||
		!reflect.DeepEqual(before, snapshotFiles(t, f.root)) {
		t.Fatalf("readiness had effects or false claims: %+v", readiness)
	}
	// Seed the expected cache independently using native Git and the reported
	// target location. No credential or production endpoint is used.
	if err := os.MkdirAll(filepath.Dir(readiness.Upstream), 0700); err != nil {
		t.Fatal(err)
	}
	fixtureGit(t, f.upstream, "clone", "--quiet", "--bare", "--no-hardlinks", f.upstream, readiness.Upstream)
	target, err := json.Marshal(readiness.Target)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(filepath.Dir(readiness.Upstream), "target.json"), string(target))
	before = snapshotFiles(t, f.root)
	_, err = runOfflineUpstreamCLI(t, "evolve", "upstream", "fetch", "--policy", f.policy, "--root", f.root, "--allow-network", "--json")
	if err != nil || !reflect.DeepEqual(before, snapshotFiles(t, f.root)) {
		t.Fatalf("cached fetch changed files: %v", err)
	}
	text, err = runOfflineUpstreamCLI(t, "evolve", "prepare", "--proposal", f.proposal, "--policy", f.policy, "--root", f.root, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var result evolve.PrepareResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		t.Fatal(err)
	}
	candidate := filepath.Join(f.root, filepath.FromSlash(result.CandidatePath))
	before = snapshotFiles(t, f.root)
	if _, err := runOfflineUpstreamCLI(t, "evolve", "plan", candidate, "--policy", f.policy, "--root", f.root, "--json"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, snapshotFiles(t, f.root)) {
		t.Fatal("cached plan wrote files")
	}
}

func TestUpstreamCLINetworkConsentAndMiss(t *testing.T) {
	f := newEvolveFixture(t)
	before := snapshotFiles(t, f.root)
	_, err := runOfflineUpstreamCLI(t, "evolve", "upstream", "fetch", "--policy", f.policy, "--root", f.root)
	if err == nil || !strings.Contains(err.Error(), "--allow-network") {
		t.Fatalf("missing consent: %v", err)
	}
	_, err = runOfflineUpstreamCLI(t, "evolve", "prepare", "--proposal", f.proposal, "--policy", f.policy, "--root", f.root)
	if err == nil || !strings.Contains(err.Error(), "not cached") {
		t.Fatalf("cache miss: %v", err)
	}
	if !reflect.DeepEqual(before, snapshotFiles(t, f.root)) {
		t.Fatal("consent failure/cache miss wrote files")
	}
}
