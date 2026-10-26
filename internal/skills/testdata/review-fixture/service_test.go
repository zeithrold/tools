package reviewfixture

import "testing"

type memoryStore struct {
	writes []string
}

func (s *memoryStore) Save(account, value string) error {
	s.writes = append(s.writes, account+":"+value)
	return nil
}

func TestSuccessfulRequestCanBeRetried(t *testing.T) {
	store := &memoryStore{}
	service := New(store)
	principal := Principal{Account: "account-a", Role: "writer"}
	request := Request{Account: "account-a", Role: "writer", Key: "key-1", Value: "value"}
	for i := 0; i < 2; i++ {
		if err := service.Submit(principal, request); err != nil {
			t.Fatal(err)
		}
	}
	if len(store.writes) != 1 {
		t.Fatalf("writes = %v; want one", store.writes)
	}
}
