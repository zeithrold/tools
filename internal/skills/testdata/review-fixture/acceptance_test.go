//go:build acceptance

package reviewfixture

import (
	"errors"
	"testing"
)

func TestRejectsUntrustedRoleAndForeignAccount(t *testing.T) {
	store := &memoryStore{}
	service := New(store)
	err := service.Submit(
		Principal{Account: "account-a", Role: "reader"},
		Request{Account: "account-b", Role: "writer", Key: "key-1", Value: "value"},
	)
	if !errors.Is(err, ErrForbidden) || len(store.writes) != 0 {
		t.Fatalf("unauthorized write: err=%v writes=%v", err, store.writes)
	}
}

func TestIdempotencyKeyIsAccountScoped(t *testing.T) {
	store := &memoryStore{}
	service := New(store)
	for _, account := range []string{"account-a", "account-b"} {
		if err := service.Submit(
			Principal{Account: account, Role: "writer"},
			Request{Account: account, Role: "writer", Key: "key-1", Value: "value"},
		); err != nil {
			t.Fatal(err)
		}
	}
	if len(store.writes) != 2 {
		t.Fatalf("independent account write suppressed: %v", store.writes)
	}
}

type failingStore struct {
	calls int
}

func (s *failingStore) Save(_, _ string) error {
	s.calls++
	return errors.New("definite write failure")
}

func TestDefiniteWriteFailureRemainsRetryable(t *testing.T) {
	store := &failingStore{}
	service := New(store)
	principal := Principal{Account: "account-a", Role: "writer"}
	request := Request{Account: "account-a", Role: "writer", Key: "key-2", Value: "value"}
	first := service.Submit(principal, request)
	second := service.Submit(principal, request)
	if first == nil || second == nil || store.calls != 2 {
		t.Fatalf("failed write falsely complete: first=%v second=%v calls=%d", first, second, store.calls)
	}
}
