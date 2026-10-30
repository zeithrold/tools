package evolve

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// Bounds for the operator-supplied validation contract. It is read through the
// same strict decoder as every other document: exact field spelling, no
// duplicate or unknown keys, no null, bounded size.
const (
	maxValidationBytes    = 64 << 10
	maxValidationChecks   = 16
	maxArgvEntries        = 32
	maxArgvBytes          = 4096
	maxArgvTotalBytes     = 16 << 10
	maxCheckTimeoutSecond = 3600
)

// ValidationCheck is one reviewed native command that verifies the candidate's
// replaced files. It is never executed by prepare, plan or status: only `evolve
// check` runs it, and only when the operator passes --trusted-local-execution.
type ValidationCheck struct {
	Name           string   `json:"name"`
	Argv           []string `json:"argv"`
	TimeoutSeconds int      `json:"timeoutSeconds"`
}

// Validation is the trusted operator's validation contract for one candidate.
//
// It lives outside the candidate, the proposal and the policy: a candidate that
// is valid under one contract stays valid under another. It belongs to `evolve
// check`, which is independent of submission. The contract must run a command for
// every name the policy requires, and may add further checks of its own; nothing
// in the policy can prevent the operator checking more than it asked for, and the
// contract cannot leave a required check unverified.
type Validation struct {
	SchemaVersion int               `json:"schemaVersion"`
	Checks        []ValidationCheck `json:"checks"`
}

// LoadValidation reads and validates the operator's validation contract.
func LoadValidation(path string) (Validation, error) {
	if strings.TrimSpace(path) == "" {
		return Validation{}, fmt.Errorf("--validation requires an operator-supplied validation file")
	}
	data, err := readDocument(path, "validation file", maxValidationBytes)
	if err != nil {
		return Validation{}, err
	}
	var validation Validation
	if err := decodeDocument(data, &validation, "validation file"); err != nil {
		return Validation{}, err
	}
	return validation.normalized()
}

// normalized validates the contract and orders its checks by name, so the
// digest describes the contract's meaning rather than its formatting.
func (v Validation) normalized() (Validation, error) {
	if v.SchemaVersion != schemaVersion {
		return Validation{}, fmt.Errorf("validation: unsupported schemaVersion %d", v.SchemaVersion)
	}
	if len(v.Checks) == 0 || len(v.Checks) > maxValidationChecks {
		return Validation{}, fmt.Errorf("validation: checks must list 1-%d commands", maxValidationChecks)
	}
	checks := make([]ValidationCheck, len(v.Checks))
	seen := make(map[string]bool, len(v.Checks))
	for i, check := range v.Checks {
		what := fmt.Sprintf("validation: checks[%d]", i)
		if !skillID.MatchString(check.Name) {
			return Validation{}, fmt.Errorf("%s: name must be a lowercase identifier, because it names a policy requiredChecks entry", what)
		}
		if seen[check.Name] {
			return Validation{}, fmt.Errorf("%s: name %s repeats", what, safeQuote(check.Name))
		}
		seen[check.Name] = true
		if err := validateArgv(check.Argv, what); err != nil {
			return Validation{}, err
		}
		if check.TimeoutSeconds < 1 || check.TimeoutSeconds > maxCheckTimeoutSecond {
			return Validation{}, fmt.Errorf("%s: timeoutSeconds must be between 1 and %d", what, maxCheckTimeoutSecond)
		}
		checks[i] = ValidationCheck{Name: check.Name, Argv: append([]string(nil), check.Argv...), TimeoutSeconds: check.TimeoutSeconds}
	}
	sort.Slice(checks, func(i, j int) bool { return checks[i].Name < checks[j].Name })
	v.Checks = checks
	return v, nil
}

// validateArgv checks one command line. zt never invokes a shell, so a shell
// metacharacter is inert data rather than a command injection; what is refused
// is content that cannot be passed to a process faithfully: empty entries,
// embedded NUL, and line breaks or control characters that would tear the argv
// apart in logs.
func validateArgv(argv []string, what string) error {
	if len(argv) == 0 || len(argv) > maxArgvEntries {
		return fmt.Errorf("%s: argv must list 1-%d entries", what, maxArgvEntries)
	}
	total := 0
	for i, entry := range argv {
		if entry == "" {
			return fmt.Errorf("%s: argv[%d] is empty", what, i)
		}
		if len(entry) > maxArgvBytes {
			return fmt.Errorf("%s: argv[%d] exceeds %d bytes", what, i, maxArgvBytes)
		}
		total += len(entry)
		if total > maxArgvTotalBytes {
			return fmt.Errorf("%s: argv exceeds %d bytes in total", what, maxArgvTotalBytes)
		}
		for _, r := range entry {
			if r == 0 || !unicode.IsPrint(r) {
				return fmt.Errorf("%s: argv[%d] contains a control character", what, i)
			}
		}
	}
	return nil
}

// covers requires the contract to run a command for every name the policy
// requires: a missing command leaves a required check unverified. Additional
// checks are the operator's own business and are accepted, so a standalone
// validation contract can verify more than the policy asks for.
func (v Validation) covers(policy Policy) error {
	provided := make(map[string]bool, len(v.Checks))
	for _, check := range v.Checks {
		provided[check.Name] = true
	}
	for _, name := range policy.RequiredChecks {
		if !provided[name] {
			return fmt.Errorf("validation: checks (%s) must include the policy's requiredChecks (%s)",
				strings.Join(v.names(), ", "), strings.Join(policy.RequiredChecks, ", "))
		}
	}
	return nil
}

func (v Validation) names() []string {
	names := make([]string, len(v.Checks))
	for i, check := range v.Checks {
		names[i] = check.Name
	}
	return names
}

// canonicalBytes renders the contract the way every stored document
// is rendered, so the digest and the bytes a reviewer reads are one value.
func (v Validation) canonicalBytes() ([]byte, error) {
	return canonicalJSON(v)
}

// digest identifies the exact contract. A changed command, argument or timeout
// changes it, so a recorded local result for another contract is not reused.
func (v Validation) digest() (string, error) {
	data, err := v.canonicalBytes()
	if err != nil {
		return "", err
	}
	return sha256Hex(data), nil
}
