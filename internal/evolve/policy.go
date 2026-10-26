package evolve

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Policy is the trusted operator policy for one upstream target. The operator
// supplies it out of band; no field is derived from a proposal or from a
// consumer project configuration, and zt has no built-in host, repository or
// base commit.
//
// requiredChecks carries descriptive names only. A policy that tried to declare a
// command would not decode (unknown field), and nothing here is ever executed.
type Policy struct {
	SchemaVersion  int      `json:"schemaVersion"`
	Host           string   `json:"host"`
	Repository     string   `json:"repository"`
	BaseCommit     string   `json:"baseCommit"`
	AllowedSkills  []string `json:"allowedSkills"`
	RequiredChecks []string `json:"requiredChecks"`
}

var (
	// skillID mirrors the shared Skill id convention: lowercase letters, digits
	// and hyphens, starting with a letter.
	skillID = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	// hostname is one or more lowercase DNS labels, without a scheme, port or
	// path. zt only records it: no command here resolves or contacts it.
	hostname = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)*$`)
	// repository is exactly owner/name, the shape GitHub uses for a repository.
	repositoryName = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
)

// LoadPolicy reads and validates the operator policy at path.
func LoadPolicy(path string) (Policy, error) {
	data, err := readDocument(path, "policy file", maxPolicyBytes)
	if err != nil {
		return Policy{}, err
	}
	var policy Policy
	if err := decodeDocument(data, &policy, "policy file"); err != nil {
		return Policy{}, err
	}
	return policy.normalized()
}

// normalized trims the declared strings, orders both lists and validates the
// result, so the policy digest depends on meaning rather than formatting.
func (p Policy) normalized() (Policy, error) {
	if p.SchemaVersion != schemaVersion {
		return Policy{}, fmt.Errorf("policy: unsupported schemaVersion %d", p.SchemaVersion)
	}
	p.Host = strings.ToLower(strings.TrimSpace(p.Host))
	p.Repository = strings.TrimSpace(p.Repository)
	p.BaseCommit = strings.TrimSpace(p.BaseCommit)
	if !hostname.MatchString(p.Host) {
		return Policy{}, fmt.Errorf("policy: host must be one lowercase hostname, for example github.com")
	}
	if !repositoryName.MatchString(p.Repository) {
		return Policy{}, fmt.Errorf("policy: repository must be owner/name, got %s", safeQuote(p.Repository))
	}
	if _, ok := fullCommitID(p.BaseCommit); !ok {
		return Policy{}, fmt.Errorf("policy: baseCommit must be a complete lowercase Git object id (40 or 64 hex digits)")
	}
	skills, err := identifierList(p.AllowedSkills, "allowedSkills")
	if err != nil {
		return Policy{}, err
	}
	if len(skills) == 0 {
		return Policy{}, fmt.Errorf("policy: allowedSkills must name at least one Skill")
	}
	checks, err := identifierList(p.RequiredChecks, "requiredChecks")
	if err != nil {
		return Policy{}, err
	}
	if len(checks) == 0 {
		return Policy{}, fmt.Errorf("policy: requiredChecks must name at least one check; zt reports it as not_run and never runs it")
	}
	p.AllowedSkills, p.RequiredChecks = skills, checks
	return p, nil
}

// identifierList trims, validates, orders and de-duplicates a list of ids.
func identifierList(values []string, field string) ([]string, error) {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if !skillID.MatchString(value) {
			return nil, fmt.Errorf("policy: %s contains an invalid id %s; use lowercase letters, digits and hyphens", field, safeQuote(value))
		}
		if seen[value] {
			return nil, fmt.Errorf("policy: %s repeats %s", field, value)
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result, nil
}

// digest identifies the normalized policy. Any semantic change to the host,
// repository, base commit, allowed Skills or required checks changes it, and a
// stored candidate that names another digest is stale.
func (p Policy) digest() (string, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return sha256Hex(data), nil
}

// target is the pinned upstream identity every candidate is bound to.
func (p Policy) target() Target {
	return Target{Host: p.Host, Repository: p.Repository, BaseCommit: p.BaseCommit}
}

func (p Policy) allows(skill string) bool {
	for _, allowed := range p.AllowedSkills {
		if skill == allowed {
			return true
		}
	}
	return false
}

// fullCommitID accepts a complete lowercase object id: 40 hex digits for a SHA-1
// repository or 64 for a SHA-256 one. An abbreviated id is refused because it
// does not identify one immutable tree.
func fullCommitID(value string) (int, bool) {
	if len(value) != 40 && len(value) != 64 {
		return 0, false
	}
	for _, r := range value {
		if !('0' <= r && r <= '9' || 'a' <= r && r <= 'f') {
			return 0, false
		}
	}
	return len(value), true
}
