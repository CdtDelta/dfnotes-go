package config

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Keep in sync with frontend/src/utils/placeholders.ts. The backend is
// authoritative; the frontend copy drives the preview and editor checks.
//
// placeholderRe matches {{...}} with no braces inside. Excluding braces makes
// "{{{HOST}}}" resolve to the inner {{HOST}} instead of a name of "{HOST".
var placeholderRe = regexp.MustCompile(`\{\{([^{}]*)\}\}`)

// placeholderName returns the trimmed name inside a full {{...}} match.
func placeholderName(match string) string {
	return strings.TrimSpace(match[2 : len(match)-2])
}

// ExtractPlaceholders returns the unique placeholder names in t, in order of
// first appearance (task order, title before description). Names are
// case-sensitive. An empty name such as {{}} or {{  }} is an error.
func ExtractPlaceholders(t TaskTemplate) ([]string, error) {
	names := []string{}
	seen := make(map[string]bool)
	for i, task := range t.Tasks {
		fields := []struct{ label, text string }{
			{"title", task.Title},
			{"description", task.Description},
		}
		for _, f := range fields {
			for _, m := range placeholderRe.FindAllString(f.text, -1) {
				name := placeholderName(m)
				if name == "" {
					return nil, fmt.Errorf("template %q: task %d %s contains an empty placeholder", t.Name, i+1, f.label)
				}
				if !seen[name] {
					seen[name] = true
					names = append(names, name)
				}
			}
		}
	}
	return names, nil
}

// SubstitutePlaceholders replaces every {{NAME}} in s with values[NAME] in a
// single pass over the original string, so a value that itself looks like a
// placeholder is inserted as literal text and never expanded. Placeholders
// with no matching value are left unchanged.
func SubstitutePlaceholders(s string, values map[string]string) string {
	return placeholderRe.ReplaceAllStringFunc(s, func(m string) string {
		if v, ok := values[placeholderName(m)]; ok {
			return v
		}
		return m
	})
}

// ValidatePlaceholderValues checks values against the placeholder names a
// template uses and returns the trimmed values. Every name needs a non-empty,
// single-line value, and keys that are not placeholders in the template are
// rejected so a frontend/backend mismatch fails loudly.
func ValidatePlaceholderValues(names []string, values map[string]string) (map[string]string, error) {
	known := make(map[string]bool, len(names))
	for _, n := range names {
		known[n] = true
	}

	var extra []string
	for k := range values {
		if !known[k] {
			extra = append(extra, k)
		}
	}
	if len(extra) > 0 {
		sort.Strings(extra)
		return nil, fmt.Errorf("unknown placeholder value(s): %s", strings.Join(quoteAll(extra), ", "))
	}

	trimmed := make(map[string]string, len(names))
	for _, n := range names {
		v, ok := values[n]
		if !ok {
			return nil, fmt.Errorf("missing value for placeholder %q", n)
		}
		v = strings.TrimSpace(v)
		if v == "" {
			return nil, fmt.Errorf("value for placeholder %q is required", n)
		}
		if strings.ContainsAny(v, "\r\n") {
			return nil, fmt.Errorf("value for placeholder %q must be a single line", n)
		}
		trimmed[n] = v
	}
	return trimmed, nil
}

// ValidateTemplates enforces the rules templates.json must satisfy on save:
// non-blank, unique template names (trimmed, case-insensitive), non-blank task
// titles, and no empty placeholders. Messages name the template and the
// 1-based task index so the frontend can show them as-is.
func ValidateTemplates(tf *TemplatesFile) error {
	if tf == nil {
		return errors.New("templates file is nil")
	}
	seen := make(map[string]string, len(tf.Templates))
	for _, t := range tf.Templates {
		name := strings.TrimSpace(t.Name)
		if name == "" {
			return errors.New("template name is required")
		}
		key := strings.ToLower(name)
		if prev, ok := seen[key]; ok {
			return fmt.Errorf("duplicate template name %q (conflicts with %q)", t.Name, prev)
		}
		seen[key] = t.Name

		for i, task := range t.Tasks {
			if strings.TrimSpace(task.Title) == "" {
				return fmt.Errorf("template %q: task %d has a blank title", t.Name, i+1)
			}
		}
		if _, err := ExtractPlaceholders(t); err != nil {
			return err
		}
	}
	return nil
}

// Find returns the template whose name matches exactly.
func (tf *TemplatesFile) Find(name string) (*TaskTemplate, error) {
	for i := range tf.Templates {
		if tf.Templates[i].Name == name {
			return &tf.Templates[i], nil
		}
	}
	return nil, fmt.Errorf("template not found: %s", name)
}

func quoteAll(ss []string) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = fmt.Sprintf("%q", s)
	}
	return out
}
