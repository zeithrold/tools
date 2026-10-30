package evolve

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"time"

	"github.com/zeithrold/tools/internal/check"
)

// Verification bounds. The workspace is a complete pinned base overlay, so it is
// bounded as a whole rather than per replacement.
const (
	executionTrustedLocal = "trusted_local_not_sandbox"
	maxWorkspaceBytes     = 32 << 20
	maxWorkspaceFileBytes = 4 << 20
	executionNotRun       = "not_run"
	executionPassed       = "passed"
	executionFailed       = "failed"
	executionBlocked      = "blocked"
)

// CheckOptions selects one candidate, the trusted policy and validation contract,
// and the local upstream holding the pinned base. TrustedLocalExecution is the
// operator's reviewed decision to execute the contract's native commands on this
// host: zt never executes them implicitly.
type CheckOptions struct {
	Candidate             string
	Policy                string
	Upstream              string
	Validation            string
	Root                  string
	TrustedLocalExecution bool
}

// CheckStepRecord is what zt observed for one validation command. LogBytes and
// LogSHA256 refer to the retained log so a reader can tie the record to the actual
// output instead of trusting a status word.
type CheckStepRecord struct {
	Name       string   `json:"name"`
	Argv       []string `json:"argv"`
	Status     string   `json:"status"`
	ExitCode   int      `json:"exitCode"`
	DurationMS int64    `json:"durationMs"`
	Log        string   `json:"log"`
	LogBytes   int      `json:"logBytes"`
	LogSHA256  string   `json:"logSha256"`
	Detail     string   `json:"detail,omitempty"`
}

// CheckResult is the stored verification record for one candidate under one
// validation contract. It is operator-local evidence: it lives in the operator's
// artifact tree, the same user can rewrite it, and it is never described as a
// sandbox, an isolation boundary or an independent attestation.
type CheckResult struct {
	SchemaVersion    int               `json:"schemaVersion"`
	CandidateID      string            `json:"candidateId"`
	ValidationDigest string            `json:"validationDigest"`
	Target           Target            `json:"target"`
	Skill            string            `json:"skill"`
	Files            []FileSummary     `json:"files"`
	Execution        string            `json:"execution"`
	WorkspaceDigest  string            `json:"workspaceDigest"`
	BaseTreeDigest   string            `json:"baseTreeDigest"`
	WorkspaceChanged bool              `json:"workspaceChanged"`
	StartedAt        string            `json:"startedAt"`
	DurationMS       int64             `json:"durationMs"`
	Status           string            `json:"status"`
	Checks           []CheckStepRecord `json:"checks"`
	Notes            []string          `json:"notes,omitempty"`
	ResultPath       string            `json:"resultPath,omitempty"`
	Reused           bool              `json:"reused"`
}

// expectedFile is one file of the workspace zt must materialise: the pinned base
// overlaid with the candidate. Content is held so the same value is written and
// then compared with what the filesystem actually stored.
type expectedFile struct {
	Mode    string
	Content []byte
	SHA256  string
}

// workspaceFile is one file as it exists on disk: the git-style mode (only the
// executable bit is meaningful) and the digest of the stored bytes.
type workspaceFile struct {
	Mode   string
	SHA256 string
}

// Verify runs the trusted validation contract against a full pinned-base workspace
// with the candidate replacements applied. It implements `zt evolve check`.
//
// It refuses to execute anything unless the operator passed
// --trusted-local-execution, and it says plainly that the run is not an OS
// sandbox: the command executes with the operator's own ambient authority over
// files, network, OS credentials and this host. Bounds, argv handling and process
// timeouts reduce accidents; they are not an isolation or credential boundary.
func Verify(ctx context.Context, options CheckOptions) (CheckResult, error) {
	if err := ctx.Err(); err != nil {
		return CheckResult{}, err
	}
	if !options.TrustedLocalExecution {
		return CheckResult{}, fmt.Errorf("evolve check refuses to execute the validation contract without --trusted-local-execution: the contract's commands would run as native processes on this host with your own ambient authority, which is neither an OS sandbox nor credential isolation; review the candidate and the contract, then pass the flag deliberately")
	}
	policy, err := LoadPolicy(options.Policy)
	if err != nil {
		return CheckResult{}, err
	}
	data, id, err := readCandidate(options.Candidate)
	if err != nil {
		return CheckResult{}, err
	}
	var candidate Candidate
	if err := decodeDocument(data, &candidate, "candidate"); err != nil {
		return CheckResult{}, err
	}
	canonical, err := candidate.canonicalBytes()
	if err != nil {
		return CheckResult{}, fmt.Errorf("encode candidate: %w", err)
	}
	if string(canonical) != string(data) {
		return CheckResult{}, fmt.Errorf("candidate: bytes are not canonical, so the file was modified after preparation")
	}
	validation, err := LoadValidation(options.Validation)
	if err != nil {
		return CheckResult{}, err
	}
	if err := validation.covers(policy); err != nil {
		return CheckResult{}, err
	}
	validationDigest, err := validation.digest()
	if err != nil {
		return CheckResult{}, err
	}
	repo, err := resolveUpstream(ctx, options.Upstream, options.Root, policy)
	if err != nil {
		return CheckResult{}, err
	}
	if err := validateCandidate(ctx, repo, policy, candidate); err != nil {
		return CheckResult{}, err
	}
	// The exact file set is computed before anything is written, so a base that no
	// physical tree can hold is refused instead of being silently folded.
	entries, err := repo.CompleteTree(ctx, candidate.Target.BaseCommit)
	if err != nil {
		return CheckResult{}, err
	}
	expected, baseDigest, err := expectedWorkspace(ctx, repo, entries, candidate)
	if err != nil {
		return CheckResult{}, err
	}
	store, err := openStore(options.Root)
	if err != nil {
		return CheckResult{}, err
	}
	defer store.Close()

	summary := summarizeFiles(candidate.Files)
	runRelative := checkRunRelative(id, validationDigest)
	if reused, ok, err := store.reusableResult(ctx, runRelative, candidate, id, validationDigest, validation, summary, baseDigest); err != nil {
		return CheckResult{}, err
	} else if ok {
		reused.Reused = true
		return reused, nil
	}

	if err := materializeWorkspace(ctx, store, runRelative, expected); err != nil {
		// Nothing ran, so the incomplete workspace is removed again: a refused run must
		// not leave a tree that looks like a verification of this candidate.
		_ = store.evolve.RemoveAll(runRelative)
		return CheckResult{}, err
	}
	runDir := filepath.Join(store.dir, filepath.FromSlash(runRelative))
	treeDir := filepath.Join(runDir, "tree")
	outDir := filepath.Join(runDir, "out")
	// The digest covers the files the filesystem actually stores, and it was already
	// compared with the expected overlay, so a folded or aliased path cannot pass.
	stored, err := readPhysicalWorkspace(treeDir)
	if err != nil {
		return CheckResult{}, err
	}
	workDigest := digestWorkspaceSet(stored)

	result := CheckResult{
		SchemaVersion:    schemaVersion,
		CandidateID:      id,
		ValidationDigest: validationDigest,
		Target:           candidate.Target,
		Skill:            candidate.Skill,
		Files:            summary,
		Execution:        executionTrustedLocal,
		WorkspaceDigest:  workDigest,
		BaseTreeDigest:   baseDigest,
		StartedAt:        time.Now().UTC().Format(time.RFC3339Nano),
		Status:           executionPassed,
		Checks:           []CheckStepRecord{},
		ResultPath:       path.Join(artifactRoot, runRelative, "result.json"),
		Notes: []string{
			"trusted local execution requested by the operator; this is not an OS sandbox and not credential isolation",
			"verification evidence is operator-local and can be altered by the same user; it is not an independent attestation",
		},
	}
	started := time.Now()
	stopped := false
	for i, step := range validation.Checks {
		record := CheckStepRecord{Name: step.Name, Argv: append([]string(nil), step.Argv...), ExitCode: -1}
		switch {
		case stopped:
			record.Status = executionNotRun
			record.Detail = "an earlier validation command did not pass"
		default:
			relativeLog := validationLogName(i, step.Name)
			execution := check.RunProcess(ctx, check.ProcessRequest{
				Argv:    step.Argv,
				Dir:     treeDir,
				LogPath: filepath.Join(runDir, filepath.FromSlash(relativeLog)),
				Env:     validationEnv(candidate, id, validationDigest, treeDir, outDir),
				Timeout: time.Duration(step.TimeoutSeconds) * time.Second,
			})
			record.Status = execution.Status
			record.ExitCode = execution.ExitCode
			record.DurationMS = execution.DurationMS
			record.Detail = execution.Detail
			record.Log = relativeLog
			record.LogBytes, record.LogSHA256, err = digestLog(filepath.Join(runDir, filepath.FromSlash(relativeLog)))
			if err != nil {
				return CheckResult{}, err
			}
			if execution.Status != executionPassed {
				stopped = true
			}
		}
		result.Checks = append(result.Checks, record)
	}
	result.DurationMS = time.Since(started).Milliseconds()

	after, readErr := readPhysicalWorkspace(treeDir)
	switch {
	case readErr != nil:
		result.WorkspaceChanged = true
		result.Status = executionBlocked
		result.Notes = append(result.Notes, "the verification workspace could not be read back after the run ("+readErr.Error()+"); results are not reusable because zt cannot attribute them to the expected input")
	case digestWorkspaceSet(after) != workDigest:
		afterDigest := digestWorkspaceSet(after)
		result.WorkspaceChanged = true
		result.Status = executionBlocked
		result.Notes = append(result.Notes, "the validation command changed tracked files in the workspace (was "+workDigest+", now "+afterDigest+"); results are not reusable because zt cannot attribute them to the expected input. Write outputs to $ZT_EVOLVE_OUT_DIR instead")
	default:
		// Aggregate by outcome, not by position: a failed command is the strongest
		// truth, then anything that could not run, and only an all-passed run is passed.
		result.Status = executionPassed
		for _, step := range result.Checks {
			switch step.Status {
			case executionPassed:
			case executionFailed:
				result.Status = executionFailed
			default:
				if result.Status != executionFailed {
					result.Status = executionBlocked
				}
			}
		}
	}
	if result.Status != executionPassed && result.Status != executionFailed && result.Status != executionBlocked {
		result.Status = executionBlocked
	}

	encoded, err := canonicalJSON(result)
	if err != nil {
		return CheckResult{}, err
	}
	if err := store.writeOwned(ctx, path.Join(runRelative, "result.json"), encoded); err != nil {
		return CheckResult{}, err
	}
	return result, nil
}

func validationLogName(index int, name string) string {
	return path.Join("logs", fmt.Sprintf("%02d-%s.log", index, name))
}

// expectedWorkspace computes the exact file set a verification run must hold: the
// pinned base overlaid with the candidate replacements. It is computed before
// anything is written, so the run can be compared with it afterwards.
//
// It deliberately does not try to model a host's case folding or Unicode
// normalisation: hosts differ, and a legitimate base may hold paths that are
// distinct on a case-sensitive filesystem. Exclusive creation and the physical
// set comparison below decide what this host can actually store.
func expectedWorkspace(ctx context.Context, repo *Repo, entries []Entry, candidate Candidate) (map[string]expectedFile, string, error) {
	replaced := make(map[string]bool, len(candidate.Files))
	for _, file := range candidate.Files {
		replaced[file.Path] = true
	}
	base := make(map[string]Entry, len(entries))
	for _, entry := range entries {
		base[entry.Path] = entry
	}
	expected := make(map[string]expectedFile, len(base)+len(candidate.Files))
	total := 0
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, "", err
		}
		if replaced[entry.Path] {
			continue
		}
		if entry.Mode != "100644" && entry.Mode != "100755" {
			return nil, "", fmt.Errorf("pinned base entry %s has mode %s, which zt refuses to materialise: a verification workspace must contain exactly the tracked files of the pinned commit", safeQuote(entry.Path), entry.Mode)
		}
		content, err := repo.ReadEntry(ctx, entry, maxWorkspaceFileBytes, "verification workspace bound")
		if err != nil {
			return nil, "", err
		}
		total += len(content)
		if total > maxWorkspaceBytes {
			return nil, "", fmt.Errorf("pinned base tree exceeds the %d byte verification workspace bound", maxWorkspaceBytes)
		}
		expected[entry.Path] = expectedFile{Mode: entry.Mode, Content: content, SHA256: sha256Hex(content)}
	}
	for _, file := range candidate.Files {
		mode := "100644"
		if existing, ok := base[file.Path]; ok {
			// A replacement carries content only, so it keeps the mode the pinned base
			// used. That keeps the materialised workspace and the submitted tree the same.
			mode = existing.Mode
		}
		expected[file.Path] = expectedFile{Mode: mode, Content: []byte(file.Content), SHA256: sha256Hex([]byte(file.Content))}
	}
	return expected, digestEntrySet(base), nil
}

// materializeWorkspace writes the expected file set with exclusive creation at each
// exact spelling, then verifies the physical result. A create that finds an existing
// path, or a physical set that differs in name, mode or content, refuses the run.
func materializeWorkspace(ctx context.Context, store *Store, runRelative string, expected map[string]expectedFile) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := store.evolve.RemoveAll(runRelative); err != nil {
		return fmt.Errorf("clear %s: %w", runRelative, err)
	}
	for _, name := range []string{"tree", "logs", "out"} {
		if err := store.evolve.MkdirAll(path.Join(runRelative, name), 0755); err != nil {
			return fmt.Errorf("create %s: %w", runRelative, err)
		}
	}
	paths := make([]string, 0, len(expected))
	for name := range expected {
		paths = append(paths, name)
	}
	sort.Strings(paths)
	for _, name := range paths {
		if err := ctx.Err(); err != nil {
			return err
		}
		file := expected[name]
		target := path.Join(runRelative, "tree", name)
		if parent := path.Dir(target); parent != "." {
			if err := store.evolve.MkdirAll(parent, 0755); err != nil {
				return fmt.Errorf("create %s: %w", parent, err)
			}
		}
		perm := os.FileMode(0644)
		if file.Mode == "100755" {
			perm = 0755
		}
		// O_EXCL: if this host stores an already-created path as the same file (case
		// folding, Unicode normalisation), the second create fails here rather than
		// letting one path overwrite another and leaving the run describing a tree the
		// candidate never named. This is a support limit of the host filesystem, not a
		// claim about which paths a Git tree may contain.
		handle, err := store.evolve.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
		if err != nil {
			if errors.Is(err, os.ErrExist) {
				return fmt.Errorf("this host's filesystem stores %s as an existing workspace path, so the expected overlay cannot be materialised distinctly; zt refuses to run against a tree it cannot reproduce", safeQuote(name))
			}
			return fmt.Errorf("write %s: %w", target, err)
		}
		if _, err := handle.Write(file.Content); err != nil {
			handle.Close()
			return fmt.Errorf("write %s: %w", target, err)
		}
		if err := handle.Close(); err != nil {
			return fmt.Errorf("write %s: %w", target, err)
		}
	}
	stored, err := readPhysicalWorkspace(filepath.Join(store.dir, filepath.FromSlash(runRelative), "tree"))
	if err != nil {
		return err
	}
	return compareWorkspace(stored, expected)
}

// readPhysicalWorkspace reads the files a directory actually holds: their stored
// names, modes (only the executable bit, as Git records it) and content digests.
func readPhysicalWorkspace(treeDir string) (map[string]workspaceFile, error) {
	stored := make(map[string]workspaceFile)
	err := filepath.WalkDir(treeDir, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if len(stored) >= 2*maxBaseEntries {
			return fmt.Errorf("the verification workspace holds more than %d files", 2*maxBaseEntries)
		}
		relative, err := filepath.Rel(treeDir, current)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(relative)
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("the verification workspace holds %s, which is not a regular file; zt refuses to run against anything else", safeQuote(name))
		}
		sha, err := fileSHA256(current, maxWorkspaceFileBytes)
		if err != nil {
			return err
		}
		stored[name] = workspaceFile{Mode: gitMode(info.Mode()), SHA256: sha}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read the verification workspace: %w", err)
	}
	return stored, nil
}

// gitMode renders the only modes Git distinguishes for regular files.
func gitMode(mode os.FileMode) string {
	if mode&0o111 != 0 {
		return "100755"
	}
	return "100644"
}

func fileSHA256(path string, limit int64) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	written, err := io.Copy(hash, io.LimitReader(file, limit+1))
	if err != nil {
		return "", err
	}
	if written > limit {
		return "", fmt.Errorf("a verification workspace file exceeds %d bytes", limit)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// compareWorkspace requires the stored set to be exactly the expected overlay.
func compareWorkspace(stored map[string]workspaceFile, expected map[string]expectedFile) error {
	if len(stored) != len(expected) {
		return fmt.Errorf("the verification workspace holds %d files but the expected overlay has %d; zt refuses to verify a tree it did not materialise exactly", len(stored), len(expected))
	}
	names := make([]string, 0, len(expected))
	for name := range expected {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		want := expected[name]
		got, ok := stored[name]
		if !ok {
			return fmt.Errorf("the verification workspace is missing %s from the expected overlay", safeQuote(name))
		}
		if got.Mode != want.Mode {
			return fmt.Errorf("the verification workspace holds %s with mode %s instead of %s", safeQuote(name), got.Mode, want.Mode)
		}
		if got.SHA256 != want.SHA256 {
			return fmt.Errorf("the verification workspace holds different bytes for %s than the expected overlay", safeQuote(name))
		}
	}
	return nil
}

// digestWorkspaceSet hashes one physical file set, so a digest always describes
// bytes that exist rather than a set zt merely intended.
func digestWorkspaceSet(set map[string]workspaceFile) string {
	names := make([]string, 0, len(set))
	for name := range set {
		names = append(names, name)
	}
	sort.Strings(names)
	hash := sha256.New()
	for _, name := range names {
		fmt.Fprintf(hash, "%s\x00%s\x00%s\n", name, set[name].Mode, set[name].SHA256)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// digestEntrySet hashes the exact Git entry set zt materialised, so a check result
// names the base it ran against rather than an implied one.
func digestEntrySet(entries map[string]Entry) string {
	paths := make([]string, 0, len(entries))
	for name := range entries {
		paths = append(paths, name)
	}
	sort.Strings(paths)
	hash := sha256.New()
	for _, name := range paths {
		fmt.Fprintf(hash, "%s\x00%s\x00%s\n", name, entries[name].Mode, entries[name].SHA)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// reusableResult returns an earlier record for exactly this candidate and contract
// only when it still validates against the retained artifacts. A failed, blocked,
// mutated, hand-edited or partially deleted run is repeated instead of trusted.
func (s *Store) reusableResult(ctx context.Context, runRelative string, candidate Candidate, candidateID, validationDigest string, validation Validation, summary []FileSummary, baseDigest string) (CheckResult, bool, error) {
	if err := ctx.Err(); err != nil {
		return CheckResult{}, false, err
	}
	result, err := validateVerificationRecord(s, runRelative, candidateID, validationDigest, candidate, validation, summary, baseDigest)
	if err != nil {
		// Not reusable. The run is simply repeated below; only an unreadable store is an
		// error worth reporting here.
		if _, found, readErr := s.readOwned(path.Join(runRelative, "result.json"), maxCandidateBytes); readErr != nil || !found {
			return CheckResult{}, false, readErr
		}
		return CheckResult{}, false, nil
	}
	return result, true, nil
}

// validateVerificationRecord is the single rule both reuse and submission apply: the
// stored record must be canonical, bound to exactly this candidate, contract and
// pinned base, and still true of the artifacts that remain on disk. It never claims
// to detect a same user forging a complete record; it only refuses records that
// contradict themselves or the retained evidence.
//
// baseDigest is computed by the caller from the current pinned base, never read from
// the record, so a record cannot describe a base zt did not verify against.
func validateVerificationRecord(store *Store, runRelative, candidateID, validationDigest string, candidate Candidate, validation Validation, summary []FileSummary, baseDigest string) (CheckResult, error) {
	data, found, err := store.readOwned(path.Join(runRelative, "result.json"), maxCandidateBytes)
	if err != nil {
		return CheckResult{}, err
	}
	if !found {
		return CheckResult{}, fmt.Errorf("no trusted local verification is recorded for this candidate and validation contract; run `zt evolve check --trusted-local-execution` first (expected %s/result.json)", path.Join(artifactRoot, runRelative))
	}
	var result CheckResult
	if err := decodeDocument(data, &result, "verification result"); err != nil {
		return CheckResult{}, err
	}
	canonical, err := canonicalJSON(result)
	if err != nil {
		return CheckResult{}, err
	}
	if !bytes.Equal(canonical, data) {
		return CheckResult{}, fmt.Errorf("the verification result is not in canonical form, so it was modified after it was written; re-run `zt evolve check`")
	}
	if result.SchemaVersion != schemaVersion || result.CandidateID != candidateID || result.ValidationDigest != validationDigest {
		return CheckResult{}, fmt.Errorf("the stored verification result is for a different candidate or validation contract; re-run `zt evolve check`")
	}
	if result.Target != candidate.Target || result.Skill != candidate.Skill {
		return CheckResult{}, fmt.Errorf("the stored verification result names a different target or Skill; re-run `zt evolve check`")
	}
	if result.Execution != executionTrustedLocal {
		return CheckResult{}, fmt.Errorf("the stored verification result does not record trusted local execution; re-run `zt evolve check`")
	}
	if result.WorkspaceChanged {
		return CheckResult{}, fmt.Errorf("the recorded verification run modified its own tracked inputs, so zt cannot attribute it to the candidate; re-run `zt evolve check` with outputs under $ZT_EVOLVE_OUT_DIR")
	}
	if result.Status != executionPassed {
		return CheckResult{}, fmt.Errorf("the recorded verification status is %s, not passed; only passed runs for this exact candidate and contract are reusable", safeQuote(result.Status))
	}
	if !equalEvidenceFiles(result.Files, summary) {
		return CheckResult{}, fmt.Errorf("the stored verification result covers different file content; re-run `zt evolve check`")
	}
	if len(result.Checks) != len(validation.Checks) {
		return CheckResult{}, fmt.Errorf("the stored verification result does not cover the current contract; re-run `zt evolve check`")
	}
	runDir := filepath.Join(store.dir, filepath.FromSlash(runRelative))
	for i, step := range result.Checks {
		expected := validation.Checks[i]
		if step.Name != expected.Name || !equalStrings(step.Argv, expected.Argv) {
			return CheckResult{}, fmt.Errorf("the stored verification result describes a different command for %s; re-run `zt evolve check`", safeQuote(expected.Name))
		}
		if step.Status != executionPassed || step.ExitCode != 0 {
			return CheckResult{}, fmt.Errorf("the recorded verification of %s is %s with exit %d, not a passed run for the current contract; re-run `zt evolve check`", safeQuote(step.Name), safeQuote(step.Status), step.ExitCode)
		}
		if step.Log != validationLogName(i, expected.Name) {
			return CheckResult{}, fmt.Errorf("the stored verification result names an unexpected log for %s; re-run `zt evolve check`", safeQuote(step.Name))
		}
		bytesOnDisk, sha, err := digestLog(filepath.Join(runDir, filepath.FromSlash(step.Log)))
		if err != nil {
			return CheckResult{}, fmt.Errorf("the recorded log for %s is no longer readable: %w; re-run `zt evolve check`", safeQuote(step.Name), err)
		}
		if bytesOnDisk != step.LogBytes || sha != step.LogSHA256 {
			return CheckResult{}, fmt.Errorf("the recorded log for %s changed after the run; re-run `zt evolve check`", safeQuote(step.Name))
		}
	}
	if !isSHA256Hex(result.WorkspaceDigest) || !isSHA256Hex(result.BaseTreeDigest) {
		return CheckResult{}, fmt.Errorf("the stored verification result does not name the materialised workspace and base set; re-run `zt evolve check`")
	}
	if result.BaseTreeDigest != baseDigest {
		return CheckResult{}, fmt.Errorf("the stored verification result names a different pinned base than the current one; re-run `zt evolve check` against this base")
	}
	stored, err := readPhysicalWorkspace(filepath.Join(runDir, "tree"))
	if err != nil {
		return CheckResult{}, fmt.Errorf("the verification workspace is no longer readable: %w; re-run `zt evolve check`", err)
	}
	if digestWorkspaceSet(stored) != result.WorkspaceDigest {
		return CheckResult{}, fmt.Errorf("the verification workspace changed after the run, so its result no longer describes the expected input; re-run `zt evolve check`")
	}
	return result, nil
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func equalEvidenceFiles(left, right []FileSummary) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

// digestLog records what the retained log actually holds, bounded, so a result
// references the output it observed.
func digestLog(path string) (int, string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return 0, "", fmt.Errorf("validation log %s: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return 0, "", fmt.Errorf("validation log %s is not a regular file", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return 0, "", err
	}
	defer file.Close()
	digest := sha256.New()
	written, err := io.Copy(digest, io.LimitReader(file, int64(check.MaxProcessLogBytes)+4096))
	if err != nil {
		return 0, "", err
	}
	return int(written), hex.EncodeToString(digest.Sum(nil)), nil
}

// validationEnv is the child's complete environment. The parent environment is
// not inherited: zt copies a few innocuous variables and its own metadata, which
// keeps an incidentally exported credential out of the child. This is accident
// reduction, not isolation: no boundary stops the command from reading files,
// using the network or reaching the user's OS keychain.
func validationEnv(candidate Candidate, candidateID, validationDigest, treeDir, outDir string) []string {
	env := make([]string, 0, 12)
	for _, name := range []string{"PATH", "HOME", "TMPDIR", "TMP", "TEMP", "LANG", "LC_ALL", "LC_CTYPE"} {
		if value, ok := os.LookupEnv(name); ok && value != "" {
			env = append(env, name+"="+value)
		}
	}
	return append(env,
		"ZT_EVOLVE_CANDIDATE_ID="+candidateID,
		"ZT_EVOLVE_SKILL="+candidate.Skill,
		"ZT_EVOLVE_VALIDATION_DIGEST="+validationDigest,
		"ZT_EVOLVE_TREE="+treeDir,
		"ZT_EVOLVE_OUT_DIR="+outDir,
	)
}
