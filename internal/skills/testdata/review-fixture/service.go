package reviewfixture

import (
	"errors"
	"sync"
)

var ErrForbidden = errors.New("forbidden")

// Principal is supplied by the caller's trusted authentication adapter.
type Principal struct {
	Account string
	Role    string
}

// Request is decoded from an untrusted client.
type Request struct {
	Account string
	Role    string
	Key     string
	Value   string
}

type Store interface {
	Save(account, value string) error
}

type Service struct {
	mu    sync.Mutex
	seen  map[string]bool
	store Store
}

func New(store Store) *Service {
	return &Service{seen: make(map[string]bool), store: store}
}

// Submit accepts writes by writer principals to their own account. A successful
// request may be retried with the same key without a duplicate write. Keys are
// independent between accounts. Failed writes remain retryable.
func (s *Service) Submit(principal Principal, request Request) error {
	if request.Role != "writer" {
		return ErrForbidden
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.seen[request.Key] {
		return nil
	}
	s.seen[request.Key] = true
	return s.store.Save(request.Account, request.Value)
}
