// Package evolve prepares, verifies and submits immutable local candidates for
// evolving one Agent Skill.
//
// A candidate is a structured replacement proposal stored as one canonical JSON
// document: it names one Skill, declares provenance and license, states a reason,
// and lists explicit file replacements, each with a path, the digest of the bytes
// it replaces and its new content. It is never a unified diff, and zt never applies
// a patch stream.
//
// Two kinds of document are kept apart, and neither can supply the other:
//
//   - The trusted operator policy, supplied with --policy. It fixes the host,
//     repository, pinned base commit and allowed Skills, and may name required
//     checks. Only an operator writes it; a proposal never carries or influences it.
//   - The untrusted replacement proposal, supplied with --proposal. Everything it
//     asserts about itself is a declaration, never observed evidence.
//
// The commands are layered so that each stage adds evidence without gaining
// authority over the next:
//
//   - prepare validates policy and proposal against a local upstream commit and
//     writes only .zt/artifacts/evolve/<content-id>/candidate.json.
//   - plan re-reads a candidate and repeats that validation, writing nothing.
//   - Verify (`evolve check`) materialises the complete pinned base with the
//     candidate applied and runs a validation contract, but only when the operator
//     passed --trusted-local-execution: those commands are native local processes,
//     which is neither an OS sandbox nor credential isolation. Verification is a
//     standalone utility; publication neither requires it nor reads its results.
//   - Submit opens one deterministic branch pinned at the base commit and one draft
//     pull request against the repository's current default branch, returns the
//     submission that already exists, or completes a verified branch that is
//     missing only its pull request. It never merges, publishes, activates,
//     force-updates a ref or reopens a closed pull request.
//   - Status reports local observations, and with an explicit opt-in reconciles the
//     pull request and GitHub's checks without writing.
//
// Prepare, plan, local status and the submission preview read no credential and open
// no network connection. Only Submit and Status with --remote use the GitHub App
// credential, through a seam that never hands zt a token.
package evolve

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	// schemaVersion is the only supported document version for the operator
	// policy, the replacement proposal and the stored candidate manifest.
	schemaVersion = 1

	// Document bounds. Every untrusted document is read through its bound before
	// it is parsed. A candidate embeds its replacement content, so its bound is
	// the proposal bound plus canonical overhead.
	maxPolicyBytes    = 64 << 10
	maxProposalBytes  = 1 << 20
	maxCandidateBytes = 2 << 20
	maxGitOutputBytes = 1 << 20

	// Replacement bounds. plan enforces the same limits on a stored candidate,
	// so a hand-edited file cannot exceed what prepare accepted.
	maxFiles        = 8
	maxFileBytes    = 256 << 10
	maxTotalBytes   = 512 << 10
	maxPathBytes    = 200
	maxPathSegments = 8
	maxReasonBytes  = 2000
	maxEvidenceRefs = 16
	maxLabelBytes   = 100

	// maxBaseEntries bounds the pinned base listing read for one Skill directory,
	// maxGitConfigBytes bounds each repository config file zt inspects before
	// starting Git, and maxTreeEntryBytes bounds one single-entry tree lookup.
	maxBaseEntries    = 4096
	maxGitConfigBytes = 64 << 10
	maxTreeEntryBytes = 4 << 10
)

// readDocument reads one bounded regular file for parsing. A symlink, a
// non-regular file, or a file replaced between the check and the read is
// refused, and the returned bytes are known not to exceed limit.
func readDocument(path, what string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("%s: refusing symlinked file %s", what, safeQuote(path))
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s: not a regular file %s", what, safeQuote(path))
	}
	if info.Size() > limit {
		return nil, fmt.Errorf("%s: %s exceeds %d bytes", what, safeQuote(path), limit)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	if !os.SameFile(info, opened) {
		return nil, fmt.Errorf("%s: file changed while it was opened %s", what, safeQuote(path))
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%s: %s exceeds %d bytes", what, safeQuote(path), limit)
	}
	return data, nil
}

// decodeDocument decodes one JSON document strictly.
//
// The document must be valid UTF-8 with no unpaired surrogate escape: a lossy
// decode would replace those bytes with U+FFFD and then pass text validation.
// Repeated keys, null values, unknown fields, trailing data and field names that
// are not spelled exactly as the schema declares are all refused. The last rule
// matters because encoding/json matches struct field names case-insensitively,
// so "Skill" would otherwise stand in for "skill" and two keys differing only in
// case could assign one field twice.
func decodeDocument(data []byte, target any, what string) error {
	if !utf8.Valid(data) {
		return fmt.Errorf("%s: is not valid UTF-8", what)
	}
	if err := rejectSurrogateEscapes(data, what); err != nil {
		return err
	}
	paths, err := scanJSON(data, what, false)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("%s: %s", what, describeJSONError(err))
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%s: trailing JSON data", what)
	}
	canonical, err := canonicalJSON(target)
	if err != nil {
		return err
	}
	canonicalPaths, err := scanJSON(canonical, what, true)
	if err != nil {
		return err
	}
	return compareKeyShapes(paths, canonicalPaths, what)
}

// canonicalJSON renders one document deterministically: two-space indentation,
// one trailing newline and no HTML escaping. Disabling HTML escaping keeps a
// document's bytes close to its content size, so a producer bound and a reader
// bound cannot disagree by a factor of six on `<`, `>` or `&`.
func canonicalJSON(value any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// scanJSON walks one document and returns the multiset of object key paths,
// refusing a repeated key inside an object and, unless allowNull is set, any null
// value. Array elements fold into one "*" segment so the same shape can be
// compared with a canonical re-encoding of the same value.
func scanJSON(data []byte, what string, allowNull bool) (map[string]int, error) {
	type frame struct {
		object  bool
		keys    map[string]bool // non-nil for objects
		keyNext bool
		keyPath string // prefix for an object's keys
		element string // prefix for a container inside an array
	}
	paths := make(map[string]int)
	decoder := json.NewDecoder(bytes.NewReader(data))
	stack := make([]*frame, 0, 8)
	lastKey := ""
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return paths, nil
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %s", what, describeJSONError(err))
		}
		var top *frame
		if len(stack) > 0 {
			top = stack[len(stack)-1]
		}
		valuePrefix := ""
		switch {
		case top == nil:
		case top.object && !top.keyNext:
			valuePrefix = lastKey
		case !top.object:
			valuePrefix = top.element
		}
		switch value := token.(type) {
		case json.Delim:
			switch value {
			case '{':
				stack = append(stack, &frame{object: true, keys: map[string]bool{}, keyNext: true, keyPath: valuePrefix})
			case '[':
				element := valuePrefix
				if element == "" {
					element = "*"
				} else {
					element += "/*"
				}
				stack = append(stack, &frame{element: element})
			default: // '}' or ']'
				if len(stack) == 0 {
					return nil, fmt.Errorf("%s: unbalanced JSON", what)
				}
				stack = stack[:len(stack)-1]
				// A composite value just ended, so its parent expects a key or the
				// next element again.
				if len(stack) > 0 && stack[len(stack)-1].object {
					stack[len(stack)-1].keyNext = true
				}
			}
		case string:
			if top != nil && top.object && top.keyNext {
				path := joinPath(top.keyPath, value)
				if top.keys[value] {
					// The repeated key is withheld: a malformed document must not echo
					// its own payload into logs.
					return nil, fmt.Errorf("%s: duplicate JSON object key", what)
				}
				top.keys[value] = true
				top.keyNext = false
				lastKey = path
				paths[path]++
				continue
			}
			if top != nil && top.object {
				top.keyNext = true // a string value completed
			}
		case nil:
			if !allowNull {
				return nil, fmt.Errorf("%s: null values are not supported; omit an optional field instead", what)
			}
			if top != nil && top.object {
				top.keyNext = true
			}
		default: // number or bool: a completed value
			if top != nil && top.object {
				top.keyNext = true
			}
		}
	}
}

// compareKeyShapes requires a document's key paths to match its canonical
// re-encoding, so every field uses its documented spelling and every required
// field is present. Exactly one field is optional: the evidence list, which is
// omitted when empty.
func compareKeyShapes(raw, canonical map[string]int, what string) error {
	for _, path := range sortedPaths(raw) {
		if optionalKey(path) {
			continue
		}
		if canonical[path] == raw[path] {
			continue
		}
		if canonical[path] > raw[path] {
			// The same key is required more often than the document spells it.
			return fmt.Errorf("%s: required field %s is missing", what, safeQuote(path))
		}
		return fmt.Errorf("%s: field %s is not the documented spelling", what, safeQuote(path))
	}
	for _, path := range sortedPaths(canonical) {
		if optionalKey(path) {
			continue
		}
		if raw[path] != canonical[path] {
			return fmt.Errorf("%s: required field %s is missing", what, safeQuote(path))
		}
	}
	return nil
}

func sortedPaths(paths map[string]int) []string {
	names := make([]string, 0, len(paths))
	for path := range paths {
		names = append(names, path)
	}
	sort.Strings(names)
	return names
}

func joinPath(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "/" + key
}

func lastSegment(value string) string {
	if index := strings.LastIndexByte(value, '/'); index >= 0 {
		return value[index+1:]
	}
	return value
}

func optionalKey(path string) bool {
	return lastSegment(path) == "evidence"
}

// describeJSONError reports what kind of decode failure happened without echoing
// the document's text. encoding/json's own message for an unknown field quotes
// the field name in full, which can be arbitrary payload.
func describeJSONError(err error) string {
	var syntax *json.SyntaxError
	if errors.As(err, &syntax) {
		return fmt.Sprintf("is not valid JSON (problem at byte %d)", syntax.Offset)
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return "is incomplete JSON"
	}
	var typeError *json.UnmarshalTypeError
	if errors.As(err, &typeError) {
		if typeError.Field != "" {
			return fmt.Sprintf("field %s has the wrong JSON type", safeQuote(typeError.Field))
		}
		return "has a value of the wrong JSON type"
	}
	if errors.Is(err, io.EOF) {
		return "is empty"
	}
	return "does not match the documented schema; unknown, misspelled or undocumented field names are rejected"
}

// rejectSurrogateEscapes refuses a JSON string escape that decodes to an unpaired
// UTF-16 surrogate. encoding/json would silently replace it with U+FFFD, so the
// document would not round-trip to the bytes it names.
func rejectSurrogateEscapes(data []byte, what string) error {
	inString := false
	for i := 0; i < len(data); i++ {
		switch data[i] {
		case '"':
			inString = !inString
		case '\\':
			if !inString {
				continue
			}
			if i+1 >= len(data) {
				return nil // the decoder reports the truncated escape
			}
			if data[i+1] != 'u' {
				i++ // skip the escaped character
				continue
			}
			code, ok := hexQuad(data, i+2)
			if !ok {
				return nil // the decoder reports the malformed escape
			}
			i += 5 // now points at the last hex digit
			switch {
			case code >= 0xDC00 && code <= 0xDFFF:
				return fmt.Errorf("%s: contains an unpaired surrogate escape", what)
			case code >= 0xD800 && code <= 0xDBFF:
				// A high surrogate is only valid immediately followed by a low one.
				if i+2 >= len(data) || data[i+1] != '\\' || data[i+2] != 'u' {
					return fmt.Errorf("%s: contains an unpaired surrogate escape", what)
				}
				low, ok := hexQuad(data, i+3)
				if !ok || low < 0xDC00 || low > 0xDFFF {
					return fmt.Errorf("%s: contains an unpaired surrogate escape", what)
				}
				i += 6
			}
		}
	}
	return nil
}

func hexQuad(data []byte, start int) (int, bool) {
	if start+4 > len(data) {
		return 0, false
	}
	value := 0
	for _, b := range data[start : start+4] {
		var digit int
		switch {
		case '0' <= b && b <= '9':
			digit = int(b - '0')
		case 'a' <= b && b <= 'f':
			digit = int(b-'a') + 10
		case 'A' <= b && b <= 'F':
			digit = int(b-'A') + 10
		default:
			return 0, false
		}
		value = value<<4 | digit
	}
	return value, true
}

// sha256Hex returns the lowercase hexadecimal SHA-256 of data.
func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// isSHA256Hex reports whether value is a complete lowercase SHA-256 digest.
func isSHA256Hex(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if !('0' <= r && r <= '9' || 'a' <= r && r <= 'f') {
			return false
		}
	}
	return true
}

// safeQuote renders an untrusted value for an error message. Long or
// unprintable values are withheld instead of echoed, so a malformed document or
// path cannot leak its payload into logs.
func safeQuote(value string) string {
	if len(value) > 200 || !utf8.ValidString(value) {
		return "<withheld " + strconv.Itoa(len(value)) + " byte value>"
	}
	for _, r := range value {
		if !unicode.IsPrint(r) {
			return "<withheld value with unprintable characters>"
		}
	}
	return strconv.Quote(value)
}

// truncate keeps the first max bytes of a printable message without splitting a
// UTF-8 sequence.
func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return strings.ToValidUTF8(value[:max], "") + "..."
}
