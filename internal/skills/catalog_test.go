package skills

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path"
	"regexp"
	"strings"
	"testing"

	tools "github.com/zeithrold/tools"
)

func TestCanonicalSkillEntrypoints(t *testing.T) {
	for _, name := range []string{
		"verification", "ui-design", "frontend-engineering", "backend-engineering",
		"task-planning", "code-review", "systematic-debugging", "agent-team",
		"performance-engineering", "skill-authoring", "evolution-proposals",
	} {
		if _, err := tools.SkillFiles.ReadFile("skills/" + name + "/SKILL.md"); err != nil {
			t.Errorf("canonical Skill %s: %v", name, err)
		}
	}
	for _, name := range []string{
		"go-testing", "js-ts-testing", "frontend-verification",
		"ui-foundation", "ui-web", "ui-flutter",
	} {
		if _, err := fs.Stat(tools.SkillFiles, "skills/"+name); err == nil {
			t.Errorf("retired Skill %s still creates an indexed entry point", name)
		} else if !os.IsNotExist(err) {
			t.Errorf("inspect retired Skill %s: %v", name, err)
		}
	}
}

// Distributed frontmatter deliberately uses only these portable, single-line
// fields. This is a check of our authoring subset, not a general YAML parser.
func TestSkillFrontmatterSubset(t *testing.T) {
	entries, err := fs.ReadDir(tools.SkillFiles, "skills")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			data, err := tools.SkillFiles.ReadFile("skills/" + entry.Name() + "/SKILL.md")
			if err != nil {
				t.Fatal(err)
			}
			rest, ok := strings.CutPrefix(string(data), "---\n")
			if !ok {
				t.Fatal("missing YAML frontmatter")
			}
			header, _, ok := strings.Cut(rest, "\n---\n")
			if !ok {
				t.Fatal("unclosed YAML frontmatter")
			}
			fields := map[string]string{}
			for _, line := range strings.Split(header, "\n") {
				key, value, ok := strings.Cut(line, ": ")
				if !ok || fields[key] != "" {
					t.Fatalf("invalid or duplicate frontmatter field: %q", line)
				}
				if key != "name" && key != "description" && key != "license" {
					t.Fatalf("field %q needs an explicit portable-format decision", key)
				}
				if strings.Contains(value, ": ") || strings.Contains(value, " #") ||
					strings.ContainsAny(value, "\r\t") || value == "" ||
					strings.ContainsRune("\"'[{&*!|>@`", rune(value[0])) {
					t.Fatalf("%s is not a safe plain single-line YAML scalar", key)
				}
				fields[key] = value
			}
			if fields["name"] != entry.Name() || fields["license"] != "MIT" ||
				fields["description"] == "" || len(fields["description"]) > 1024 {
				t.Fatal("name, MIT license or description contract violated")
			}
		})
	}
}

func TestSkillReferencesAreIndependentlyInstallable(t *testing.T) {
	link := regexp.MustCompile(`\]\(([^)\s]+)\)`)
	err := fs.WalkDir(tools.SkillFiles, "skills", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(name, ".md") {
			return err
		}
		data, err := tools.SkillFiles.ReadFile(name)
		if err != nil {
			return err
		}
		parts := strings.Split(name, "/")
		owner := "skills/" + parts[1] + "/"
		for _, match := range link.FindAllStringSubmatch(string(data), -1) {
			target := strings.SplitN(match[1], "#", 2)[0]
			if target == "" || strings.Contains(target, "://") {
				continue
			}
			target = path.Clean(path.Join(path.Dir(name), target))
			if !strings.HasPrefix(target, owner) {
				t.Errorf("%s requires a resource outside its installation unit: %s", name, match[1])
				continue
			}
			if _, err := fs.Stat(tools.SkillFiles, target); err != nil {
				t.Errorf("%s has missing resource %s: %v", name, match[1], err)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMigratedResourcesRetainReviewedBytes(t *testing.T) {
	var fixture struct {
		Resources []struct {
			From   string `json:"from"`
			To     string `json:"to"`
			SHA256 string `json:"sha256"`
		} `json:"resources"`
	}
	data, err := os.ReadFile("testdata/migration-content.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Resources) == 0 {
		t.Fatal("no migrated resources checked")
	}
	for _, resource := range fixture.Resources {
		data, err := tools.SkillFiles.ReadFile(resource.To)
		if err != nil {
			t.Errorf("%s -> %s: %v", resource.From, resource.To, err)
			continue
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != resource.SHA256 {
			t.Errorf("%s differs from its reviewed migration source %s; review intentional revisions separately", resource.To, resource.From)
		}
	}
}
