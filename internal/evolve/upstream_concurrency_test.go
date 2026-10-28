package evolve

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/zeithrold/tools/internal/githubauth"
)

func TestFetchUpstreamConcurrentPublishers(t *testing.T) {
	for _, initialized := range []bool{false, true} {
		name := "fresh-store"
		if initialized {
			name = "existing-store"
		}
		t.Run(name, func(t *testing.T) {
			work, commit := newUpstream(t, baseFileSet())
			upstream := newServedFetchUpstream(t, work)
			_, endpoint := upstreamFetchEndpoint(t, upstream.server.URL, commit)
			root := t.TempDir()
			policy := filepath.Join(t.TempDir(), "policy.json")
			doc := policyDoc(commit)
			doc["host"], doc["repository"] = "127.0.0.1", "owner/name"
			writeJSONFile(t, policy, doc)
			if initialized {
				store, err := openStore(root)
				if err != nil {
					t.Fatal(err)
				}
				store.Close()
			}
			const callers = 16
			var ready sync.WaitGroup
			ready.Add(callers)
			type outcome struct {
				result UpstreamResult
				err    error
			}
			results := make(chan outcome, callers)
			for i := 0; i < callers; i++ {
				go func() {
					result, err := FetchUpstream(context.Background(), FetchUpstreamOptions{
						Policy: policy, Root: root, AllowNetwork: true,
						OpenGit: func(context.Context, Target) (githubauth.GitEndpoint, error) {
							// All callers finish local preflight before the first
							// can initialize the artifact store or publish.
							ready.Done()
							ready.Wait()
							return endpoint, nil
						},
					})
					results <- outcome{result, err}
				}()
			}
			created := 0
			var cache string
			for i := 0; i < callers; i++ {
				result := <-results
				if result.err != nil {
					t.Errorf("concurrent fetch failed: %v", result.err)
					continue
				}
				if result.result.Status != "ready" {
					t.Errorf("fetch not ready: %+v", result.result)
				}
				if result.result.Created {
					created++
				}
				cache = result.result.Upstream
			}
			if created != 1 {
				t.Fatalf("created=%d, want one published baseline", created)
			}
			status, err := UpstreamStatus(context.Background(), policy, root)
			if err != nil || status.Status != "ready" {
				t.Fatalf("winner cache: %+v %v", status, err)
			}
			entries, err := os.ReadDir(filepath.Dir(filepath.Dir(cache)))
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), ".fetch-") {
					t.Errorf("fetch left a staging directory: %s", entry.Name())
				}
			}
		})
	}
}
