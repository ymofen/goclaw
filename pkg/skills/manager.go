package skills

import (
	"encoding/json"
	"fmt"
	"sync"

	"goclaw/pkg/model"
)

// Handler executes one skill with parsed JSON arguments.
// args is the raw JSON object passed by the model/skill caller.
type Handler = model.ToolHandler

// Manager stores skill definitions and their execution handlers.
type Manager struct {
	mu       sync.RWMutex
	defs     map[string]SkillDef
	handlers map[string]Handler
	order    []string
}

// NewManager creates an empty skill manager.
func NewManager() *Manager {
	return &Manager{
		defs:     map[string]SkillDef{},
		handlers: map[string]Handler{},
		order:    make([]string, 0),
	}
}

// Register adds a skill definition and its executor.
func (m *Manager) Register(def SkillDef, handler Handler) error {
	if m == nil {
		return fmt.Errorf("skills: manager is nil")
	}
	if handler == nil {
		return fmt.Errorf("skills: handler is nil")
	}
	if def.Function.Name == "" {
		return fmt.Errorf("skills: function.name is required")
	}
	if def.Type == "" {
		def.Type = "function"
	}
	if def.Type != "function" {
		return fmt.Errorf("skills: unsupported type %q", def.Type)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.handlers[def.Function.Name]; exists {
		return fmt.Errorf("skills: %s already registered", def.Function.Name)
	}

	m.defs[def.Function.Name] = def
	m.handlers[def.Function.Name] = handler
	m.order = append(m.order, def.Function.Name)
	return nil
}

// Unregister removes one registered skill by function name.
func (m *Manager) Unregister(name string) error {
	if m == nil {
		return fmt.Errorf("skills: manager is nil")
	}
	if name == "" {
		return fmt.Errorf("skills: function.name is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.handlers[name]; !exists {
		return fmt.Errorf("unknown skill: %s", name)
	}

	delete(m.handlers, name)
	delete(m.defs, name)

	for i, item := range m.order {
		if item == name {
			m.order = append(m.order[:i], m.order[i+1:]...)
			break
		}
	}

	return nil
}

// Definitions returns a snapshot of all registered skill definitions.
func (m *Manager) Definitions() []SkillDef {
	if m == nil {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]SkillDef, 0, len(m.order))
	for _, name := range m.order {
		out = append(out, m.defs[name])
	}
	return out
}

// Execute dispatches to a registered skill handler.
func (m *Manager) Execute(name, argsJSON string) (string, error) {
	if m == nil {
		return "", fmt.Errorf("skills: manager is nil")
	}

	m.mu.RLock()
	h, ok := m.handlers[name]
	m.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("unknown skill: %s", name)
	}

	return h(json.RawMessage(argsJSON))
}
