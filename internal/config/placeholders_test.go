package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestExtractPlaceholdersOrderAndDedupe(t *testing.T) {
	tmpl := TaskTemplate{Name: "Evidence Item", Tasks: []TemplateTask{
		{Title: "[{{HOST}}] Acquire {{ IMAGE }}", Description: "Ticket {{TICKET}} for {{HOST}}"},
		{Title: "{{IMAGE}} hash", Description: "{{OWNER}}"},
		{Title: "{{HOST}} done"},
	}}
	got, err := ExtractPlaceholders(tmpl)
	if err != nil {
		t.Fatalf("ExtractPlaceholders: %v", err)
	}
	want := []string{"HOST", "IMAGE", "TICKET", "OWNER"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExtractPlaceholdersTitleBeforeDescription(t *testing.T) {
	tmpl := TaskTemplate{Tasks: []TemplateTask{{Title: "{{B}}", Description: "{{A}}"}}}
	got, err := ExtractPlaceholders(tmpl)
	if err != nil {
		t.Fatalf("ExtractPlaceholders: %v", err)
	}
	if want := []string{"B", "A"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExtractPlaceholdersTrimsWhitespace(t *testing.T) {
	tmpl := TaskTemplate{Tasks: []TemplateTask{{Title: "{{ HOST }} and {{HOST}} and {{\tHOST  }}"}}}
	got, err := ExtractPlaceholders(tmpl)
	if err != nil {
		t.Fatalf("ExtractPlaceholders: %v", err)
	}
	if want := []string{"HOST"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExtractPlaceholdersCaseSensitive(t *testing.T) {
	tmpl := TaskTemplate{Tasks: []TemplateTask{{Title: "{{host}} {{HOST}} {{Host}}"}}}
	got, err := ExtractPlaceholders(tmpl)
	if err != nil {
		t.Fatalf("ExtractPlaceholders: %v", err)
	}
	if want := []string{"host", "HOST", "Host"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExtractPlaceholdersNoneReturnsEmptySlice(t *testing.T) {
	tmpl := TaskTemplate{Tasks: []TemplateTask{{Title: "Plain {X} [Y] {{HOST"}}}
	got, err := ExtractPlaceholders(tmpl)
	if err != nil {
		t.Fatalf("ExtractPlaceholders: %v", err)
	}
	// Non-nil so the binding serializes to [] rather than null.
	if got == nil || len(got) != 0 {
		t.Errorf("got %#v, want empty non-nil slice", got)
	}
}

func TestExtractPlaceholdersEmptyNameError(t *testing.T) {
	for _, s := range []string{"{{}}", "{{   }}", "ok {{HOST}} then {{ }}"} {
		tmpl := TaskTemplate{Name: "T", Tasks: []TemplateTask{{Title: "fine"}, {Title: "x", Description: s}}}
		_, err := ExtractPlaceholders(tmpl)
		if err == nil {
			t.Errorf("%q: expected error", s)
			continue
		}
		if !strings.Contains(err.Error(), `"T"`) || !strings.Contains(err.Error(), "task 2 description") {
			t.Errorf("%q: error %q should name template and task", s, err)
		}
	}
}

func TestSubstitutePlaceholders(t *testing.T) {
	values := map[string]string{"HOST": "WS-042", "OTHER": "nope"}
	cases := []struct {
		name, in, want string
		values         map[string]string
	}{
		{"basic", "[{{HOST}}] Acquire", "[WS-042] Acquire", values},
		{"whitespace in token", "{{ HOST }}", "WS-042", values},
		{"multiple occurrences", "{{HOST}}/{{HOST}}/{{ HOST }}", "WS-042/WS-042/WS-042", values},
		{"value containing token stays literal", "[{{HOST}}] {{OTHER}}", "[{{OTHER}}] nope",
			map[string]string{"HOST": "{{OTHER}}", "OTHER": "nope"}},
		{"unbalanced open", "{{HOST and more", "{{HOST and more", values},
		{"unbalanced close", "HOST}} and more", "HOST}} and more", values},
		{"single braces", "{HOST} {X}", "{HOST} {X}", values},
		{"square brackets", "[HOST] [[E001]]", "[HOST] [[E001]]", values},
		{"triple braces", "{{{HOST}}}", "{WS-042}", values},
		{"case sensitive miss left alone", "{{host}}", "{{host}}", values},
		{"no placeholders", "plain text", "plain text", values},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := SubstitutePlaceholders(c.in, c.values); got != c.want {
				t.Errorf("SubstitutePlaceholders(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestValidatePlaceholderValues(t *testing.T) {
	names := []string{"HOST", "TICKET"}

	got, err := ValidatePlaceholderValues(names, map[string]string{"HOST": "  WS-042 ", "TICKET": "INC1"})
	if err != nil {
		t.Fatalf("valid values: %v", err)
	}
	if got["HOST"] != "WS-042" || got["TICKET"] != "INC1" {
		t.Errorf("values not trimmed: %v", got)
	}

	bad := []struct {
		name   string
		values map[string]string
	}{
		{"missing", map[string]string{"HOST": "a"}},
		{"empty", map[string]string{"HOST": "a", "TICKET": ""}},
		{"whitespace only", map[string]string{"HOST": "a", "TICKET": "   "}},
		{"newline", map[string]string{"HOST": "a\nb", "TICKET": "c"}},
		{"carriage return", map[string]string{"HOST": "a\rb", "TICKET": "c"}},
		{"extra key", map[string]string{"HOST": "a", "TICKET": "b", "EXTRA": "c"}},
		{"key differs by case", map[string]string{"host": "a", "TICKET": "b"}},
	}
	for _, b := range bad {
		if _, err := ValidatePlaceholderValues(names, b.values); err == nil {
			t.Errorf("%s: expected error", b.name)
		}
	}

	if got, err := ValidatePlaceholderValues(nil, map[string]string{}); err != nil || len(got) != 0 {
		t.Errorf("no placeholders, no values: got %v, %v", got, err)
	}
}

func TestValidateTemplatesValid(t *testing.T) {
	tf := &TemplatesFile{Templates: []TaskTemplate{
		{Name: "Evidence Item", Tasks: []TemplateTask{{Title: "[{{HOST}}] Acquire", Description: "{X} [Y]"}}},
		{Name: "Triage", Tasks: []TemplateTask{{Title: "Interview"}}},
		{Name: "Empty", Tasks: nil},
	}}
	if err := ValidateTemplates(tf); err != nil {
		t.Errorf("valid file rejected: %v", err)
	}
	if err := ValidateTemplates(&TemplatesFile{}); err != nil {
		t.Errorf("empty file rejected: %v", err)
	}
}

func TestValidateTemplatesDuplicateNames(t *testing.T) {
	for _, dup := range []string{"Triage", "triage", "TRIAGE", "  Triage  ", "\tTriAge"} {
		tf := &TemplatesFile{Templates: []TaskTemplate{
			{Name: "Triage", Tasks: []TemplateTask{{Title: "a"}}},
			{Name: dup, Tasks: []TemplateTask{{Title: "b"}}},
		}}
		err := ValidateTemplates(tf)
		if err == nil {
			t.Errorf("duplicate %q not rejected", dup)
			continue
		}
		if !strings.Contains(err.Error(), "duplicate template name") {
			t.Errorf("duplicate %q: unexpected error %q", dup, err)
		}
	}
}

func TestValidateTemplatesBlankTitle(t *testing.T) {
	for _, title := range []string{"", "   ", "\t\n"} {
		tf := &TemplatesFile{Templates: []TaskTemplate{
			{Name: "Triage", Tasks: []TemplateTask{{Title: "ok"}, {Title: title}}},
		}}
		err := ValidateTemplates(tf)
		if err == nil {
			t.Errorf("blank title %q not rejected", title)
			continue
		}
		if !strings.Contains(err.Error(), `"Triage"`) || !strings.Contains(err.Error(), "task 2") {
			t.Errorf("blank title %q: error %q should name template and task", title, err)
		}
	}
}

func TestValidateTemplatesEmptyPlaceholder(t *testing.T) {
	for _, task := range []TemplateTask{{Title: "a {{}}"}, {Title: "a", Description: "{{  }}"}} {
		tf := &TemplatesFile{Templates: []TaskTemplate{{Name: "T", Tasks: []TemplateTask{task}}}}
		if err := ValidateTemplates(tf); err == nil {
			t.Errorf("empty placeholder in %+v not rejected", task)
		}
	}
}

func TestValidateTemplatesBlankName(t *testing.T) {
	tf := &TemplatesFile{Templates: []TaskTemplate{{Name: "   ", Tasks: []TemplateTask{{Title: "a"}}}}}
	if err := ValidateTemplates(tf); err == nil {
		t.Error("blank template name not rejected")
	}
}

func TestSaveTemplatesRejectsInvalidAndLoadToleratesIt(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	invalid := &TemplatesFile{Templates: []TaskTemplate{
		{Name: "Dup", Tasks: []TemplateTask{{Title: "a"}}},
		{Name: "dup", Tasks: []TemplateTask{{Title: ""}}},
	}}
	if err := SaveTemplates(invalid); err == nil {
		t.Fatal("SaveTemplates accepted an invalid file")
	}
	if tf, err := LoadTemplates(); err != nil || len(tf.Templates) != 0 {
		t.Fatalf("rejected save should not write: %v, %v", tf, err)
	}

	// An existing file with old problems must still load.
	path, err := templatesPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := writeRawForTest(path, `{"templates":[{"name":"Dup","tasks":[{"title":""}]},{"name":"dup","tasks":[{"title":"{{}}"}]}]}`); err != nil {
		t.Fatal(err)
	}
	tf, err := LoadTemplates()
	if err != nil {
		t.Fatalf("LoadTemplates on legacy file: %v", err)
	}
	if len(tf.Templates) != 2 {
		t.Errorf("got %d templates, want 2", len(tf.Templates))
	}
}

func writeRawForTest(path, data string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(data), 0o600)
}
