package mcpskills

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Limits every conforming host supports (Skills extension, "Limits").
const (
	MaxResources = 512
	MaxTotalSize = 16 << 20 // 16 MiB
)

// namePattern: lowercase letters, digits and single hyphens, not at the ends
// (Agent Skills specification, "name" field).
var namePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// ValidateName returns why name violates the Agent Skills naming rules, or "".
func ValidateName(name string) string {
	switch {
	case name == "":
		return "name is empty"
	case len(name) > 64:
		return fmt.Sprintf("name has %d characters, at most 64 allowed", len(name))
	case !namePattern.MatchString(name):
		return fmt.Sprintf("name %q: only a-z, 0-9 and single hyphens, not at the start or end", name)
	}
	return ""
}

// ValidateFrontmatter checks the fields the Agent Skills specification
// requires or constrains; it returns the problems found.
func ValidateFrontmatter(fm map[string]any) []string {
	var problems []string
	name, _ := fm["name"].(string)
	if msg := ValidateName(name); msg != "" {
		problems = append(problems, msg)
	}
	desc, _ := fm["description"].(string)
	switch {
	case strings.TrimSpace(desc) == "":
		problems = append(problems, "description is missing or empty")
	case len([]rune(desc)) > 1024:
		problems = append(problems, fmt.Sprintf("description has %d characters, at most 1024 allowed", len([]rune(desc))))
	}
	if c, ok := fm["compatibility"]; ok {
		s, _ := c.(string)
		if n := len([]rune(s)); n < 1 || n > 500 {
			problems = append(problems, "compatibility must be a string of 1-500 characters")
		}
	}
	if m, ok := fm["metadata"]; ok {
		mm, isMap := m.(map[string]any)
		if !isMap {
			problems = append(problems, "metadata must be a mapping")
		}
		for k, v := range mm {
			if _, isString := v.(string); !isString {
				problems = append(problems, fmt.Sprintf("metadata.%s must be a string", k))
			}
		}
	}
	return problems
}

// ParseFrontmatter returns the YAML frontmatter of a SKILL.md as a JSON-like
// map (the form the extension puts into a skill entry).
func ParseFrontmatter(skillMD []byte) (map[string]any, error) {
	text := bytes.TrimPrefix(skillMD, []byte("\xef\xbb\xbf"))
	if !bytes.HasPrefix(text, []byte("---\n")) && !bytes.HasPrefix(text, []byte("---\r\n")) {
		return nil, errors.New("SKILL.md does not start with YAML frontmatter (---)")
	}
	rest := text[bytes.IndexByte(text, '\n')+1:]
	end := bytes.Index(rest, []byte("\n---"))
	if end < 0 {
		return nil, errors.New("SKILL.md frontmatter is not closed (---)")
	}
	var fm map[string]any
	if err := yaml.Unmarshal(rest[:end], &fm); err != nil {
		return nil, fmt.Errorf("SKILL.md frontmatter: %w", err)
	}
	if fm == nil {
		fm = map[string]any{}
	}
	return fm, nil
}

// Digest formats the SHA-256 of data as the extension requires: sha256:{hex}.
func Digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

var digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// ValidDigest reports whether d has the form sha256:{64 lowercase hex}.
func ValidDigest(d string) bool { return digestPattern.MatchString(d) }
