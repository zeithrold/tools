package evolve

import (
	"context"
	"fmt"
	"time"

	"github.com/zeithrold/tools/internal/keystore"
)

const maxJournalBytes = 64 << 10

// Journal states. They describe what zt has observed, in order; a run that lost
// the connection after a write leaves the state at the last observed step and the
// next run reconciles through reads instead of guessing.
const (
	journalStarted        = "started"
	journalObjectsCreated = "objects_created"
	journalRefCreated     = "ref_created"
	journalCompleted      = "completed"
)

// JournalPullRequest is the pull request zt observed, by number and public URL.
// No token, log or private path is ever recorded here.
type JournalPullRequest struct {
	Number int    `json:"number"`
	URL    string `json:"url,omitempty"`
}

// Journal is the durable-under-process-interruption record of one submission
// attempt for one candidate and one public payload. A rename makes an update
// visible atomically, so a crash mid-write leaves the previous record; it is not a
// claim of durability across machine power loss.
//
// It lives beside the candidates in the operator's owned artifact tree, never
// inside a candidate directory: candidate bytes are their own identity, while a
// journal is mutable state about a remote operation. It is written before the
// first remote write, so a crashed or interrupted run leaves evidence of what may
// have been created.
type Journal struct {
	SchemaVersion       int                 `json:"schemaVersion"`
	CandidateID         string              `json:"candidateId"`
	PublicPayloadDigest string              `json:"publicPayloadDigest"`
	Branch              string              `json:"branch"`
	BaseBranch          string              `json:"baseBranch"`
	BaseCommit          string              `json:"baseCommit"`
	Tree                string              `json:"tree,omitempty"`
	Commit              string              `json:"commit,omitempty"`
	State               string              `json:"state"`
	PullRequest         *JournalPullRequest `json:"pullRequest,omitempty"`
	UpdatedAt           string              `json:"updatedAt"`
	Note                string              `json:"note,omitempty"`
}

// loadJournal reads the journal for exactly this candidate and payload. A journal
// naming anything else is not adopted, because a stale record of another payload
// must not describe this attempt.
func loadJournal(store *Store, relative, candidateID, payloadDigest string) (Journal, bool, error) {
	data, found, err := store.readOwned(relative, maxJournalBytes)
	if err != nil || !found {
		return Journal{}, false, err
	}
	var journal Journal
	if err := decodeDocument(data, &journal, "submission journal"); err != nil {
		return Journal{}, false, err
	}
	if journal.CandidateID != candidateID || journal.PublicPayloadDigest != payloadDigest {
		return Journal{}, false, fmt.Errorf("submission journal %s describes a different candidate or payload; refusing to treat it as this attempt", relative)
	}
	if journal.SchemaVersion != schemaVersion {
		return Journal{}, false, fmt.Errorf("submission journal: unsupported schemaVersion %d", journal.SchemaVersion)
	}
	return journal, true, nil
}

// record advances the journal and writes it atomically before the caller acts on
// the transition.
//
// "Atomically" is scoped: the write replaces a temporary sibling by rename, so a
// reader never sees a half-written journal and a process interrupted mid-write
// leaves the previous record intact. It does not claim durability across machine
// power loss.
func record(ctx context.Context, store *Store, relative string, journal *Journal, state, note string) error {
	journal.State = state
	journal.Note = note
	journal.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	data, err := canonicalJSON(journal)
	if err != nil {
		return err
	}
	return store.writeOwned(ctx, relative, data)
}

// journalRank orders the observable states, so a stale attempt can be recognised
// instead of overwriting newer progress.
var journalRank = map[string]int{
	journalStarted:        1,
	journalObjectsCreated: 2,
	journalRefCreated:     3,
	journalCompleted:      4,
}

// advanceJournal records the next observed state, after re-reading what is stored.
//
// A completed submission is terminal: no later attempt may overwrite it, because
// that would claim a finished publication had regressed. A lower state is not
// written when a higher one is already recorded; recovery re-reads the remote side
// instead of rewinding the local record. The caller holds the submission lock, so
// this read-modify-write is serialised against other zt processes.
func advanceJournal(ctx context.Context, store *Store, relative string, journal *Journal, state, note string) error {
	current, found, err := loadJournal(store, relative, journal.CandidateID, journal.PublicPayloadDigest)
	if err != nil {
		return err
	}
	if found {
		if current.State == journalCompleted && state != journalCompleted {
			return fmt.Errorf("the submission journal already records a completed submission; refusing to overwrite it with %s. Re-run to reconcile with the remote state", state)
		}
		// Carry forward what an earlier attempt already observed.
		if current.Tree != "" {
			journal.Tree = current.Tree
		}
		if current.Commit != "" && journal.Commit == "" {
			journal.Commit = current.Commit
		}
		if current.PullRequest != nil && journal.PullRequest == nil {
			journal.PullRequest = current.PullRequest
		}
		if journalRank[current.State] > journalRank[state] {
			// The stored record already describes further progress; keep it rather than
			// rewinding to a state this attempt re-entered.
			return nil
		}
	}
	return record(ctx, store, relative, journal, state, note)
}

// submissionLockService scopes the submission lock. The lock file contains no
// credential; it only serialises one candidate+payload transaction across zt
// processes, reusing the native lock the credential store already relies on.
const submissionLockService = "dev.zeithrold.zt.evolve"

// submissionLockWait bounds how long a second attempt waits for the first. A
// crashed process leaves its lock behind by design: zt never steals a lock, and the
// error names the path so an operator can confirm it is stale and remove it.
const submissionLockWait = 2 * time.Minute

// submissionLock serialises one candidate+payload submission across processes.
// Callers acquire it after local preflight and before any journal write or remote
// write; previews, local status and failing preflights never take it.
func submissionLock(ctx context.Context, lockDir, key string) (func(), error) {
	locker, err := keystore.NewLocker(keystore.LockerOptions{Service: submissionLockService, Dir: lockDir})
	if err != nil {
		return nil, fmt.Errorf("create the submission lock: %w", err)
	}
	lockCtx, cancel := context.WithTimeout(ctx, submissionLockWait)
	defer cancel()
	unlocker, err := locker.Lock(lockCtx, key)
	if err != nil {
		return nil, fmt.Errorf("another zt evolve submit holds this candidate's lock; wait for it to finish. If no zt process is running, confirm the lock is stale and remove it by hand: %w", err)
	}
	return func() { _ = unlocker.Unlock() }, nil
}
