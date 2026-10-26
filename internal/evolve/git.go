package evolve

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Repo is a read-only handle on one local upstream Git repository.
//
// zt runs only object-read plumbing (rev-parse, cat-file, ls-tree). That set
// cannot run a hook, invoke a shell alias, contact a transport or update an
// index or worktree, so repository content cannot cause an execution or a write.
// The environment strips variables that could redirect the repository, redirect a
// trace, or override the settings below, and every call passes
// --no-replace-objects so a replace ref cannot substitute the pinned tree. Lazy
// fetching, optional locks and terminal prompts are disabled, so a missing object
// is an error rather than a fetch.
//
// Git always reads repository configuration, and no flag disables that, so OpenRepo
// refuses a repository whose config could include another file. Together with the
// disabled system and global configuration this keeps every Git read inside the
// supplied repository.
type Repo struct {
	dir    string
	git    string
	format string // "sha1" or "sha256"
}

// treeMode is the object mode ls-tree reports for a directory.
const treeMode = "040000"

// Entry is one base tree entry. Trees carry treeMode so a caller can tell a
// directory from a file, symlink or submodule.
type Entry struct {
	Path string
	Mode string // 040000, 100644, 100755, 120000 (symlink), 160000 (submodule)
	SHA  string
}

// OpenRepo resolves dir to one local repository and pins its object format. The
// format matters because a policy's baseCommit must be a complete object id of
// the repository it names, not an abbreviated or differently-hashed one.
func OpenRepo(ctx context.Context, dir string) (*Repo, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, fmt.Errorf("--upstream requires a local Git repository directory")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("upstream: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("upstream: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("upstream is not a directory: %s", safeQuote(abs))
	}
	// This check has to run before the first Git process: a repository config is
	// read by every command, and an include directive can name any path.
	if err := rejectConfigIncludes(abs); err != nil {
		return nil, err
	}
	git, err := exec.LookPath("git")
	if err != nil {
		return nil, fmt.Errorf("git is required to read the pinned base commit: %w", err)
	}
	repo := &Repo{dir: abs, git: git}
	out, err := repo.output(ctx, 256, "rev-parse", "--show-object-format")
	if err != nil {
		return nil, err
	}
	format := strings.TrimSpace(string(out))
	if format != "sha1" && format != "sha256" {
		return nil, fmt.Errorf("local git reports unsupported object format %s", safeQuote(format))
	}
	repo.format = format
	return repo, nil
}

// rejectConfigIncludes refuses a repository whose local Git configuration names
// another file to include. include and includeIf can point at any path, including
// one under the user's home, and Git offers no switch that disables repository
// configuration, so zt refuses such a repository instead of claiming that it
// never reads outside the repository it was given.
func rejectConfigIncludes(dir string) error {
	files, err := repositoryConfigFiles(dir)
	if err != nil {
		return err
	}
	for _, path := range files {
		data, err := readOptionalFile(path)
		if err != nil {
			return err
		}
		if data == nil {
			continue
		}
		if section := includeSection(string(data)); section != "" {
			return fmt.Errorf("local git config %s has an [%s] section, so zt refuses this repository: an include can name files outside it", safeQuote(path), section)
		}
	}
	return nil
}

// repositoryConfigFiles returns every config file Git could read for the
// repository at dir. The layout is resolved here, without starting Git, because
// the first process would already follow an include. Layouts outside a plain
// worktree, a linked worktree or submodule and a bare repository are refused.
func repositoryConfigFiles(dir string) ([]string, error) {
	dotGit := filepath.Join(dir, ".git")
	info, err := os.Lstat(dotGit)
	switch {
	case errors.Is(err, os.ErrNotExist):
		// A bare repository, or a .git directory named directly, keeps its own
		// configuration beside HEAD and objects and may still point at a shared
		// common directory, so it uses the same resolver as any other git directory.
		if isDirectory(filepath.Join(dir, "objects")) && isRegularFile(filepath.Join(dir, "HEAD")) {
			return gitDirConfigFiles(dir)
		}
		return nil, fmt.Errorf("upstream is not a local git repository: %s", safeQuote(dir))
	case err != nil:
		return nil, fmt.Errorf("upstream: %w", err)
	case info.IsDir():
		return gitDirConfigFiles(dotGit)
	case info.Mode().IsRegular():
		// A linked worktree or submodule keeps a gitdir pointer in .git.
		data, err := readDocument(dotGit, "upstream .git file", maxGitConfigBytes)
		if err != nil {
			return nil, err
		}
		line, err := pointerLine(data, "upstream .git file")
		if err != nil {
			return nil, err
		}
		pointer, ok := strings.CutPrefix(line, "gitdir: ")
		if !ok {
			return nil, fmt.Errorf("upstream .git file is not a gitdir pointer")
		}
		gitDir, err := pointerPath(pointer, "upstream .git file")
		if err != nil {
			return nil, err
		}
		if !filepath.IsAbs(gitDir) {
			gitDir = filepath.Join(dir, gitDir)
		}
		return gitDirConfigFiles(gitDir)
	default:
		return nil, fmt.Errorf("upstream .git entry is neither a directory nor a gitdir file")
	}
}

// gitDirConfigFiles returns the config files beside one git directory, including
// the common directory a linked worktree shares with its main repository.
func gitDirConfigFiles(gitDir string) ([]string, error) {
	files := []string{filepath.Join(gitDir, "config"), filepath.Join(gitDir, "config.worktree")}
	common, err := readOptionalFile(filepath.Join(gitDir, "commondir"))
	if err != nil {
		return nil, err
	}
	if common != nil {
		line, err := pointerLine(common, "git directory commondir file")
		if err != nil {
			return nil, err
		}
		remote, err := pointerPath(line, "git directory commondir file")
		if err != nil {
			return nil, err
		}
		if !filepath.IsAbs(remote) {
			remote = filepath.Join(gitDir, remote)
		}
		files = append(files, filepath.Join(remote, "config"), filepath.Join(remote, "config.worktree"))
	}
	return files, nil
}

// pointerLine returns the first line of a pointer file without its terminator.
// Git reads exactly one value from .git and one from commondir, so content after
// that line means zt cannot tell which value Git would use.
func pointerLine(data []byte, what string) (string, error) {
	line, rest, _ := bytes.Cut(data, []byte{'\n'})
	if len(rest) != 0 {
		return "", fmt.Errorf("%s has more than one line, so zt refuses to guess which value Git would read", what)
	}
	return string(line), nil
}

// pointerPath validates one path a pointer file names, without normalizing it.
//
// Git drops the line terminator and keeps every other byte, so trimming a space
// here can make zt inspect one directory while the Git it then starts opens
// another. A directory name ending in a space is exactly that case: the include
// preflight would miss the configuration it promises to read before Git runs. A
// value that begins or ends with whitespace is therefore refused rather than
// trimmed. Refusing depends on no Git version's exact trimming rule and keeps the
// preflight and Git's own resolution describing one path.
func pointerPath(value, what string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("%s is empty", what)
	}
	if value != strings.TrimSpace(value) || strings.ContainsAny(value, "\r\n") {
		return "", fmt.Errorf("%s names a path that begins or ends with whitespace, so zt cannot prove which directory Git would read; refusing this repository", what)
	}
	return value, nil
}

// includeSection reports the config section that would make Git read another
// file. Every include is refused, including a conditional one zt could evaluate:
// the point is not to decide whether it matches but to keep Git from following
// any path the repository names.
func includeSection(text string) string {
	// Git skips a UTF-8 byte order mark at the very start of a config file, so zt
	// must skip it too. Otherwise the first section header stays hidden behind the
	// BOM and its include would be followed without being examined.
	text = strings.TrimLeft(text, "\ufeff")
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "[") {
			name := sectionName(lower)
			if name == "include" || name == "includeif" {
				return name
			}
			continue
		}
		if strings.HasPrefix(lower, "include.") {
			return "include"
		}
	}
	return ""
}

// sectionName extracts the section name from a config header line, for example
// `[includeIf "gitdir:~/x"]` becomes "includeif".
func sectionName(line string) string {
	line = strings.TrimPrefix(line, "[")
	if index := strings.IndexAny(line, " \t]\""); index >= 0 {
		line = line[:index]
	}
	return strings.TrimSpace(line)
}

// readOptionalFile reads one bounded regular file and reports a missing file as
// nil rather than as an error.
func readOptionalFile(path string) ([]byte, error) {
	data, err := readDocument(path, "local git config", maxGitConfigBytes)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return data, nil
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func isRegularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// VerifyBaseCommit checks that the pinned commit is a complete object id in this
// repository's own hash format and that the object is present locally.
func (r *Repo) VerifyBaseCommit(ctx context.Context, commit string) error {
	length, ok := fullCommitID(commit)
	if !ok {
		return fmt.Errorf("pinned base commit must be a complete lowercase git object id")
	}
	expected := 40
	if r.format == "sha256" {
		expected = 64
	}
	if length != expected {
		return fmt.Errorf("pinned base commit has %d hex digits but the local repository uses %s", length, r.format)
	}
	out, err := r.output(ctx, 256, "cat-file", "-t", commit)
	if err != nil {
		return fmt.Errorf("pinned base commit %s is not readable in the local upstream repository, and zt never fetches missing objects: %w", commit, err)
	}
	if strings.TrimSpace(string(out)) != "commit" {
		return fmt.Errorf("pinned base %s is not a commit", commit)
	}
	return nil
}

// TreeEntries lists every non-tree entry below prefix at the pinned commit.
// Trees are not listed; a file, symlink or submodule at a contemplated ancestor
// therefore appears as an entry in the way of a replacement.
func (r *Repo) TreeEntries(ctx context.Context, commit, prefix string) (map[string]Entry, error) {
	out, err := r.output(ctx, maxGitOutputBytes, "ls-tree", "-r", "-z", "--full-tree", commit, "--", prefix)
	if err != nil {
		return nil, err
	}
	entries := make(map[string]Entry)
	for _, record := range bytes.Split(out, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		meta, value, ok := strings.Cut(string(record), "\t")
		if !ok {
			return nil, fmt.Errorf("local git returned an unexpected tree record")
		}
		fields := strings.Fields(meta)
		if len(fields) != 3 {
			return nil, fmt.Errorf("local git returned an unexpected tree record")
		}
		if value != prefix && !strings.HasPrefix(value, prefix+"/") {
			return nil, fmt.Errorf("local git returned a path outside %s", safeQuote(prefix))
		}
		if len(entries) >= maxBaseEntries {
			return nil, fmt.Errorf("base directory %s has more than %d entries", safeQuote(prefix), maxBaseEntries)
		}
		entries[value] = Entry{Path: value, Mode: fields[0], SHA: fields[2]}
	}
	return entries, nil
}

// Entry returns the exact tree entry at path below commit. A directory is
// reported with treeMode, so a caller can distinguish it from a file, symlink or
// submodule that would redirect every path below it.
func (r *Repo) Entry(ctx context.Context, commit, path string) (Entry, bool, error) {
	out, err := r.output(ctx, maxTreeEntryBytes, "ls-tree", "-z", "--full-tree", commit, "--", path)
	if err != nil {
		return Entry{}, false, err
	}
	for _, record := range bytes.Split(out, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		meta, value, ok := strings.Cut(string(record), "\t")
		if !ok {
			return Entry{}, false, fmt.Errorf("local git returned an unexpected tree record")
		}
		if value != path {
			continue
		}
		fields := strings.Fields(meta)
		if len(fields) != 3 {
			return Entry{}, false, fmt.Errorf("local git returned an unexpected tree record")
		}
		return Entry{Path: value, Mode: fields[0], SHA: fields[2]}, true, nil
	}
	return Entry{}, false, nil
}

// DirectoryNames lists the direct child names of one directory at commit. The
// empty directory names the repository root. Names are returned as the tree
// spells them, so a caller can compare spellings that differ only in case.
func (r *Repo) DirectoryNames(ctx context.Context, commit, dir string) ([]string, error) {
	tree := commit
	if dir != "" {
		tree = commit + ":" + dir
	}
	out, err := r.output(ctx, maxGitOutputBytes, "ls-tree", "-z", tree)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, 16)
	for _, record := range bytes.Split(out, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		meta, value, ok := strings.Cut(string(record), "\t")
		if !ok {
			return nil, fmt.Errorf("local git returned an unexpected tree record")
		}
		if len(strings.Fields(meta)) != 3 {
			return nil, fmt.Errorf("local git returned an unexpected tree record")
		}
		if strings.Contains(value, "/") {
			continue // ls-tree without -r lists direct children only
		}
		if len(names) >= maxBaseEntries {
			return nil, fmt.Errorf("base directory %s has more than %d entries", safeQuote(dir), maxBaseEntries)
		}
		names = append(names, value)
	}
	return names, nil
}

// Blob reads one base file, refusing anything above the per-file bound before it
// is read.
func (r *Repo) Blob(ctx context.Context, entry Entry) ([]byte, error) {
	out, err := r.output(ctx, 64, "cat-file", "-s", entry.SHA)
	if err != nil {
		return nil, err
	}
	size, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil || size < 0 {
		return nil, fmt.Errorf("local git reported an invalid size for %s", safeQuote(entry.Path))
	}
	if size > maxFileBytes {
		return nil, fmt.Errorf("base file %s is %d bytes, above the %d byte replacement bound", safeQuote(entry.Path), size, maxFileBytes)
	}
	data, err := r.output(ctx, maxFileBytes, "cat-file", "blob", entry.SHA)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) != size {
		return nil, fmt.Errorf("base file %s changed size while it was read", safeQuote(entry.Path))
	}
	return data, nil
}

// gitEnv returns the environment for one plumbing call: the caller's environment
// without variables that could redirect the repository, inject config or override
// the read-only settings below, plus those settings themselves. Every name zt
// sets is removed first, because a duplicate entry would let the inherited value
// win depending on the platform's lookup order.
func gitEnv() []string {
	blocked := map[string]bool{
		"GIT_DIR":                          true,
		"GIT_WORK_TREE":                    true,
		"GIT_INDEX_FILE":                   true,
		"GIT_OBJECT_DIRECTORY":             true,
		"GIT_ALTERNATE_OBJECT_DIRECTORIES": true,
		"GIT_COMMON_DIR":                   true,
		"GIT_NAMESPACE":                    true,
		"GIT_CONFIG":                       true,
		"GIT_CONFIG_GLOBAL":                true,
		"GIT_CONFIG_SYSTEM":                true,
		"GIT_CONFIG_PARAMETERS":            true,
		"GIT_CONFIG_COUNT":                 true,
		"GIT_CONFIG_NOSYSTEM":              true,
		"GIT_NO_LAZY_FETCH":                true,
		"GIT_OPTIONAL_LOCKS":               true,
		"GIT_TERMINAL_PROMPT":              true,
		"GIT_PAGER":                        true,
		"GIT_NO_REPLACE_OBJECTS":           true,
		"GIT_REPLACE_REF_BASE":             true,
	}
	environ := os.Environ()
	env := make([]string, 0, len(environ)+8)
	for _, entry := range environ {
		name, _, _ := strings.Cut(entry, "=")
		if blocked[name] || strings.HasPrefix(name, "GIT_TRACE") {
			// Every GIT_TRACE* variable names a destination Git writes to, which
			// would let an otherwise read-only command write outside the workspace.
			continue
		}
		env = append(env, entry)
	}
	return append(env,
		"GIT_NO_LAZY_FETCH=1",
		"GIT_OPTIONAL_LOCKS=0",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_NO_REPLACE_OBJECTS=1",
		"GIT_PAGER=cat",
	)
}

// output runs one read-only plumbing command and returns its bounded stdout.
// --no-replace-objects is a main option, so a replacement ref cannot substitute
// the pinned commit's tree for the object the policy names.
func (r *Repo) output(ctx context.Context, limit int64, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, r.git, append([]string{"--no-replace-objects", "--no-pager", "-C", r.dir}, args...)...)
	command.Env = gitEnv()
	stdout := &boundedBuffer{limit: limit}
	stderr := &boundedBuffer{limit: 8 << 10}
	command.Stdout, command.Stderr = stdout, stderr
	err := command.Run()
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, fmt.Errorf("local git %s: %w", args[0], ctxErr)
	}
	if err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			return nil, fmt.Errorf("local git %s: %w", args[0], err)
		}
		return nil, fmt.Errorf("local git %s failed: %s", args[0], safeQuote(truncate(detail, 160)))
	}
	if stdout.overflow {
		return nil, fmt.Errorf("local git %s produced more than %d bytes", args[0], limit)
	}
	return stdout.buf.Bytes(), nil
}

// boundedBuffer keeps at most limit bytes and records an overflow, so an
// unexpected large object or listing cannot exhaust memory.
type boundedBuffer struct {
	limit    int64
	buf      bytes.Buffer
	overflow bool
}

func (b *boundedBuffer) Write(data []byte) (int, error) {
	if int64(b.buf.Len())+int64(len(data)) > b.limit {
		b.overflow = true
		// Consume and discard so the child never blocks on a full pipe; the
		// overflow is reported after it exits.
		return len(data), nil
	}
	return b.buf.Write(data)
}

func (b *boundedBuffer) String() string {
	if b.overflow {
		return b.buf.String() + "..."
	}
	return b.buf.String()
}
