package tests

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"goclaw/pkg/skills"
)

// skillsDir returns the absolute path to static/skills from any working directory.
func skillsDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// file is .../tests/file_skill_test.go; go up one level then into static/skills
	root := filepath.Dir(filepath.Dir(file))
	return filepath.Join(root, "static", "skills")
}

// TestLoadSkillsFromDir_LoadsDocx verifies that LoadSkillsFromDir successfully
// loads the docx skill bundled in static/skills.
func TestLoadSkillsFromDir_LoadsDocx(t *testing.T) {
	mgr := skills.NewFileSkillMgr()
	dir := skillsDir(t)

	if err := mgr.LoadSkillsFromDir(dir); err != nil {
		t.Fatalf("LoadSkillsFromDir(%q) error: %v", dir, err)
	}

	prompt := mgr.GetPrompt()
	if prompt == "" {
		t.Fatal("GetPrompt() returned empty string after loading skills")
	}

	fmt.Println(prompt)

	if !strings.Contains(prompt, "docx") {
		t.Errorf("GetPrompt() does not mention 'docx'; got:\n%s", prompt)
	}
}

// TestFileSkillMgr_GetPrompt_ContainsAbsPath verifies that the combined prompt
// contains the absolute path to each skill's directory.
func TestFileSkillMgr_GetPrompt_ContainsAbsPath(t *testing.T) {
	mgr := skills.NewFileSkillMgr()
	dir := skillsDir(t)

	if err := mgr.LoadSkillsFromDir(dir); err != nil {
		t.Fatalf("LoadSkillsFromDir: %v", err)
	}

	prompt := mgr.GetPrompt()

	// The prompt must contain the absolute path to static/skills/docx so that
	// the model knows exactly where to find files.
	docxAbs := filepath.Join(dir, "docx")
	if !strings.Contains(prompt, docxAbs) {
		t.Errorf("GetPrompt() does not contain absolute path %q\nprompt:\n%s", docxAbs, prompt)
	}
}

// TestFileSkillMgr_GetPrompt_NoPlaceholder verifies that the combined prompt
// does not contain the raw "{this_skill_dir}" placeholder.
func TestFileSkillMgr_GetPrompt_NoPlaceholder(t *testing.T) {
	mgr := skills.NewFileSkillMgr()
	dir := skillsDir(t)

	if err := mgr.LoadSkillsFromDir(dir); err != nil {
		t.Fatalf("LoadSkillsFromDir: %v", err)
	}

	prompt := mgr.GetPrompt()
	if strings.Contains(prompt, "{this_skill_dir}") {
		t.Errorf("GetPrompt() still contains literal '{this_skill_dir}' placeholder")
	}
}

// TestFileSkillMgr_GetPrompt_ContainsSkillContent verifies that body content
// from SKILL.md is present in the combined prompt (not just headers).
func TestFileSkillMgr_GetPrompt_ContainsSkillContent(t *testing.T) {
	mgr := skills.NewFileSkillMgr()
	dir := skillsDir(t)

	if err := mgr.LoadSkillsFromDir(dir); err != nil {
		t.Fatalf("LoadSkillsFromDir: %v", err)
	}

	prompt := mgr.GetPrompt()
	// A keyword that appears in the SKILL.md body of the docx skill
	if !strings.Contains(prompt, "DOCX") {
		t.Errorf("GetPrompt() does not contain SKILL.md body content; prompt:\n%s", prompt)
	}
}

// TestFileSkill_GetPrompt_ReplacesPlaceholder verifies that a single FileSkill's
// GetPrompt() replaces {this_skill_dir} with the absolute directory path.
func TestFileSkill_GetPrompt_ReplacesPlaceholder(t *testing.T) {
	dir := skillsDir(t)
	docxDir := filepath.Join(dir, "docx")

	skill, err := skills.NewFileSkill(docxDir)
	if err != nil {
		t.Fatalf("NewFileSkill(%q): %v", docxDir, err)
	}

	prompt := skill.GetPrompt()

	if strings.Contains(prompt, "{this_skill_dir}") {
		t.Error("FileSkill.GetPrompt() still contains literal '{this_skill_dir}' placeholder")
	}

	// The absolute dir must appear where the placeholder was
	if !strings.Contains(prompt, skill.GetDir()) {
		t.Errorf("FileSkill.GetPrompt() does not contain absolute dir %q\nprompt:\n%s", skill.GetDir(), prompt)
	}
}
