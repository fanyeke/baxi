package llm

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"fmt"
	"path"
	"strings"
	"sync"
	"text/template"
)

//go:embed prompts/*.md
var promptFiles embed.FS

// PromptTemplate holds a loaded prompt with both system and user components.
type PromptTemplate struct {
	ID           string
	Version      string
	SystemPrompt string
	UserTemplate string
	Hash         string
}

// UserPromptData is the data model for rendering user prompt templates.
type UserPromptData struct {
	ContextJSON      string
	AllowedActions   []string
	ForbiddenActions []string
	EnrichedObjects  []EnrichedObjectData
}

// PromptRegistry provides access to embedded prompt templates.
type PromptRegistry struct {
	mu      sync.RWMutex
	prompts map[string]*PromptTemplate
}

// NewPromptRegistry loads all prompts from embedded files.
func NewPromptRegistry() (*PromptRegistry, error) {
	reg := &PromptRegistry{
		prompts: make(map[string]*PromptTemplate),
	}

	entries, err := promptFiles.ReadDir("prompts")
	if err != nil {
		return nil, fmt.Errorf("read prompts directory: %w", err)
	}

	type pending struct {
		system string
		user   string
	}
	groups := make(map[string]*pending)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}

		content, err := promptFiles.ReadFile(path.Join("prompts", name))
		if err != nil {
			return nil, fmt.Errorf("read prompt file %s: %w", name, err)
		}

		promptID, promptType, version, err := parsePromptFilename(name)
		if err != nil {
			return nil, fmt.Errorf("parse filename %s: %w", name, err)
		}

		if _, ok := groups[promptID]; !ok {
			groups[promptID] = &pending{}
		}

		switch promptType {
		case "system":
			groups[promptID].system = string(content)
		case "user":
			groups[promptID].user = string(content)
		case "repair":
			// Repair prompts are loaded separately via repair_prompt.go; skip here.
			continue
		default:
			return nil, fmt.Errorf("unknown prompt type %q in %s", promptType, name)
		}

		if groups[promptID].system != "" && groups[promptID].user != "" {
			if _, err := template.New(promptID).Parse(groups[promptID].user); err != nil {
				return nil, fmt.Errorf("parse user template for %s: %w", promptID, err)
			}

			hash := fmt.Sprintf("%x", sha256.Sum256([]byte(groups[promptID].system)))

			reg.prompts[promptID] = &PromptTemplate{
				ID:           promptID,
				Version:      version,
				SystemPrompt: groups[promptID].system,
				UserTemplate: groups[promptID].user,
				Hash:         hash,
			}
		}
	}

	if len(reg.prompts) == 0 {
		return nil, fmt.Errorf("no prompts loaded")
	}

	return reg, nil
}

// parsePromptFilename extracts prompt ID, type, and version from a filename.
// Expected format: {domain}_{type}_v{version}.md
// Example: decision_system_v1.md -> ID="decision_support", type="system", version="v1"
func parsePromptFilename(filename string) (promptID string, promptType string, version string, err error) {
	base := strings.TrimSuffix(filename, ".md")
	parts := strings.Split(base, "_")
	if len(parts) < 3 {
		return "", "", "", fmt.Errorf("invalid filename format %q: expected {domain}_{type}_v{version}.md", filename)
	}

	promptType = parts[len(parts)-2]
	version = parts[len(parts)-1]

	if !strings.HasPrefix(version, "v") {
		return "", "", "", fmt.Errorf("invalid version %q in %s: must start with 'v'", version, filename)
	}

	domain := strings.Join(parts[:len(parts)-2], "_")
	promptID = domain + "_support"

	return promptID, promptType, version, nil
}

// Load returns a prompt template by ID.
func (r *PromptRegistry) Load(promptID string) (*PromptTemplate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tmpl, ok := r.prompts[promptID]
	if !ok {
		return nil, fmt.Errorf("prompt %q not found", promptID)
	}
	return tmpl, nil
}

// Hash returns the SHA-256 hex digest of the system prompt for the given ID.
func (r *PromptRegistry) Hash(promptID string) (string, error) {
	tmpl, err := r.Load(promptID)
	if err != nil {
		return "", err
	}
	return tmpl.Hash, nil
}

// List returns the IDs of all registered prompts.
func (r *PromptRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.prompts))
	for id := range r.prompts {
		ids = append(ids, id)
	}
	return ids
}

// RenderUserPrompt renders the user template for the given prompt ID with data.
func (r *PromptRegistry) RenderUserPrompt(promptID string, data UserPromptData) (string, error) {
	tmpl, err := r.Load(promptID)
	if err != nil {
		return "", err
	}

	parsed, err := template.New(promptID).Parse(tmpl.UserTemplate)
	if err != nil {
		return "", fmt.Errorf("parse user template for %s: %w", promptID, err)
	}

	var buf bytes.Buffer
	if err := parsed.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute user template for %s: %w", promptID, err)
	}

	return buf.String(), nil
}
