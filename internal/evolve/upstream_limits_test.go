package evolve

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type upstreamResponseFixture func(*http.Request) (*http.Response, error)

func (f upstreamResponseFixture) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestUpstreamBoundsDiscoveryBeforeParsing(t *testing.T) {
	body := &upstreamCloseFixture{Reader: strings.NewReader(strings.Repeat("x", maxGitOutputBytes+1))}
	transport := redirectRefusingTransport{base: upstreamResponseFixture(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
	})}
	request, err := http.NewRequest(http.MethodGet, "https://github.example/example/repo.git/info/refs?service=git-upload-pack", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := transport.RoundTrip(request)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(response.Body)
	if !errors.Is(err, errUpstreamPackBound) || len(data) > maxGitOutputBytes {
		t.Fatalf("unbounded discovery: bytes=%d err=%v", len(data), err)
	}
	response.Body.Close()
	if !body.closed {
		t.Fatal("bounded body did not close underlying response")
	}
}

type upstreamCloseFixture struct {
	io.Reader
	closed bool
}

func (f *upstreamCloseFixture) Close() error { f.closed = true; return nil }
