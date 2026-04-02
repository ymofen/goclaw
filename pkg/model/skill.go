package model

// Skill represents a named skill with a descriptive prompt.
type Skill interface {
	// GetName returns the skill's unique name.
	GetName() string
	// GetPrompt returns the prompt text that describes this skill.
	GetPrompt() string
}

// SkillMgr manages a collection of skills and their associated tools.
type SkillMgr interface {
	// RegisterSkill adds or replaces a skill registration.
	RegisterSkill(skill Skill) error
	// UnregisterSkill removes a skill by its name.
	UnregisterSkill(skill Skill)
	// GetPrompt returns the combined prompt text of all registered skills.
	GetPrompt() string
	// RegisterToolFunction registers skill-related tool functions into the toolkit.
	RegisterToolFunction(toolkit Toolkit)
}
