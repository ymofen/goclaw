package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// FileSkill implements model.Skill by loading a SKILL.md file from a directory.
//
// The skill name is resolved in order:
//  1. The `name` field from a YAML front matter block at the top of SKILL.md.
//  2. The base name of the directory if no YAML or no `name` field is present.
type FileSkill struct {
	name    string
	rawMeta string                 // raw YAML text between the --- delimiters
	meta    map[string]interface{} // parsed YAML front matter fields (supports nested structures)
	body    string                 // content after the front matter block
	dir     string
}

// NewFileSkill loads a skill from dir. dir must contain a SKILL.md file.
// dir is resolved to an absolute path so that GetDir() and GetPrompt() always
// return stable, absolute references regardless of the caller's working directory.
func NewFileSkill(dir string) (*FileSkill, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("skills: cannot resolve absolute path for %s: %w", dir, err)
	}
	skillFile := filepath.Join(absDir, "SKILL.md")
	data, err := os.ReadFile(skillFile)
	if err != nil {
		return nil, fmt.Errorf("skills: cannot read %s: %w", skillFile, err)
	}

	meta, rawMeta, body := parseFrontMatter(string(data))

	name, _ := meta["name"].(string)
	if name == "" {
		name = filepath.Base(dir)
	}

	return &FileSkill{
		name:    name,
		rawMeta: rawMeta,
		meta:    meta,
		body:    body,
		dir:     absDir,
	}, nil
}

// GetName returns the skill's unique name (from front matter `name` field or directory base name).
func (s *FileSkill) GetName() string { return s.name }

// GetDir returns the absolute path to the skill's directory.
func (s *FileSkill) GetDir() string { return s.dir }

// GetDescription returns the `description` field from the YAML front matter, or "" if absent.
func (s *FileSkill) GetDescription() string {
	v, _ := s.meta["description"].(string)
	return v
}

// GetMetaString returns the string value of any top-level YAML front matter field by key.
// Returns "" if the key is absent or its value is not a plain string.
func (s *FileSkill) GetMetaString(key string) string {
	v, _ := s.meta[key].(string)
	return v
}

func (s *FileSkill) GetMetaStrings(sep string, ignoreKeys ...string) (lst []string) {
	for k, v := range s.meta {
		if contains(ignoreKeys, k) {
			continue
		}
		str := fmt.Sprintf("%s%s%v", k, sep, v)
		lst = append(lst, str)
	}
	return
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// GetMeta returns all parsed YAML front matter fields.
// Supports nested structures, arrays, and other complex YAML types.
func (s *FileSkill) GetMeta() map[string]interface{} { return s.meta }

// GetRawMeta returns the raw YAML text from the front matter block (without --- delimiters).
func (s *FileSkill) GetRawMeta() string { return s.rawMeta }

// GetPrompt returns the description body of SKILL.md (content after the front matter block).
// All occurrences of the placeholder "{this_skill_dir}" are replaced with the
// skill's absolute directory path so the model can directly reference files.
func (s *FileSkill) GetPrompt() string {
	return strings.ReplaceAll(s.body, "{this_skill_dir}", s.dir)
}

// frontMatterRe matches a YAML front matter block at the start of a file.
// Group 1 = raw YAML content, Group 2 = body after the closing delimiter.
var frontMatterRe = regexp.MustCompile(`(?s)^---\n(.*?)\n---\n?(.*)`)

// nameLineRe matches a `name:` line (case-insensitive key) in YAML front matter.
var nameLineRe = regexp.MustCompile(`(?im)^name:.*\n?`)

// parseFrontMatter splits content into a key-value map of YAML front matter
// fields, the raw YAML text, and the remaining body text.
// Supports complex YAML structures (nested objects, arrays, etc.) via gopkg.in/yaml.v3.
// If no front matter is present, the map is empty, rawMeta is "", and body equals the full content.
// Line endings are normalized to LF before parsing so CRLF files on Windows work correctly.
func parseFrontMatter(content string) (meta map[string]interface{}, rawMeta, body string) {
	// Normalize Windows CRLF → LF so the regex and YAML parser are not affected by line endings.
	content = strings.ReplaceAll(content, "\r\n", "\n")
	meta = make(map[string]interface{})
	m := frontMatterRe.FindStringSubmatch(content)
	if m == nil {
		return meta, "", content
	}
	rawMeta = m[1]
	// rawMeta = nameLineRe.ReplaceAllString(rawMeta, "")
	// rawMeta = strings.TrimSpace(rawMeta)

	// Use YAML library to parse the front matter
	err := yaml.Unmarshal([]byte(rawMeta), &meta)
	if err != nil {
		// If YAML parsing fails, return empty meta and treat entire content as body
		// This provides graceful degradation for malformed front matter
		return make(map[string]interface{}), "", content
	}

	return meta, rawMeta, m[2]
}
