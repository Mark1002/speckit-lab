package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// e builds n two-byte runes, so a byte-counting bug breaks the at-limit rows.
func e(n int) string { return strings.Repeat("é", n) }

const family = "👨‍👩‍👧" // zero-width joiners must stay allowed (research R7)

func TestCheckText(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		v         string
		required  bool
		max       int
		multiline bool
		want      string
	}{
		// Project name: Required; 1–100 chars; single line; no control chars
		{"name at limit", "name", e(100), true, 100, false, ""},
		{"name over limit", "name", e(101), true, 100, false, "name must be 1 to 100 characters"},
		{"name empty", "name", "", true, 100, false, "name must be 1 to 100 characters"},
		{"name spaces", "name", "   ", true, 100, false, "name must be 1 to 100 characters"},
		{"name only whitespace", "name", " \n ", true, 100, false, "name must be 1 to 100 characters"},
		{"name line break", "name", "a\nb", true, 100, false, "name must be a single line"},
		{"name CRLF", "name", "a\r\nb", true, 100, false, "name must not contain control characters"},
		{"name tab", "name", "a\tb", true, 100, false, "name must not contain control characters"},
		{"name NUL", "name", "a\x00b", true, 100, false, "name must not contain control characters"},
		{"name emoji", "name", family, true, 100, false, ""},

		// Project description: Optional (""); ≤ 1,000 chars; \n allowed; no other control chars
		{"project description empty", "description", "", false, 1000, true, ""},
		{"project description at limit", "description", e(1000), false, 1000, true, ""},
		{"project description over limit", "description", e(1001), false, 1000, true, "description must be at most 1000 characters"},
		{"project description spaces", "description", "   ", false, 1000, true, ""},
		{"project description only whitespace", "description", " \n ", false, 1000, true, ""},
		{"project description line break", "description", "a\nb", false, 1000, true, ""},
		{"project description CRLF", "description", "a\r\nb", false, 1000, true, "description must not contain control characters"},
		{"project description tab", "description", "a\tb", false, 1000, true, "description must not contain control characters"},
		{"project description NUL", "description", "a\x00b", false, 1000, true, "description must not contain control characters"},
		{"project description emoji", "description", family, false, 1000, true, ""},

		// Task title: Required; 1–200 chars; single line; no control chars
		{"title at limit", "title", e(200), true, 200, false, ""},
		{"title over limit", "title", e(201), true, 200, false, "title must be 1 to 200 characters"},
		{"title empty", "title", "", true, 200, false, "title must be 1 to 200 characters"},
		{"title spaces", "title", "   ", true, 200, false, "title must be 1 to 200 characters"},
		{"title only whitespace", "title", " \n ", true, 200, false, "title must be 1 to 200 characters"},
		{"title line break", "title", "a\nb", true, 200, false, "title must be a single line"},
		{"title CRLF", "title", "a\r\nb", true, 200, false, "title must not contain control characters"},
		{"title tab", "title", "a\tb", true, 200, false, "title must not contain control characters"},
		{"title NUL", "title", "a\x00b", true, 200, false, "title must not contain control characters"},
		{"title emoji", "title", family, true, 200, false, ""},

		// Task description: Optional (""); ≤ 5,000 chars; \n allowed; no other control chars
		{"task description empty", "description", "", false, 5000, true, ""},
		{"task description at limit", "description", e(5000), false, 5000, true, ""},
		{"task description over limit", "description", e(5001), false, 5000, true, "description must be at most 5000 characters"},
		{"task description spaces", "description", "   ", false, 5000, true, ""},
		{"task description only whitespace", "description", " \n ", false, 5000, true, ""},
		{"task description line break", "description", "a\nb", false, 5000, true, ""},
		{"task description CRLF", "description", "a\r\nb", false, 5000, true, "description must not contain control characters"},
		{"task description tab", "description", "a\tb", false, 5000, true, "description must not contain control characters"},
		{"task description NUL", "description", "a\x00b", false, 5000, true, "description must not contain control characters"},
		{"task description emoji", "description", family, false, 5000, true, ""},

		// Comment text: Required; 1–2,000 chars; \n allowed; no other control chars
		{"text at limit", "text", e(2000), true, 2000, true, ""},
		{"text over limit", "text", e(2001), true, 2000, true, "text must be 1 to 2000 characters"},
		{"text empty", "text", "", true, 2000, true, "text must be 1 to 2000 characters"},
		{"text spaces", "text", "   ", true, 2000, true, "text must be 1 to 2000 characters"},
		{"text only whitespace", "text", " \n ", true, 2000, true, "text must be 1 to 2000 characters"},
		{"text line break", "text", "a\nb", true, 2000, true, ""},
		{"text CRLF", "text", "a\r\nb", true, 2000, true, "text must not contain control characters"},
		{"text tab", "text", "a\tb", true, 2000, true, "text must not contain control characters"},
		{"text NUL", "text", "a\x00b", true, 2000, true, "text must not contain control characters"},
		{"text emoji", "text", family, true, 2000, true, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := checkText(tc.field, tc.v, tc.required, tc.max, tc.multiline)
			if tc.want == "" {
				if got != nil {
					t.Fatalf("want valid, got %q", got.Message)
				}
				return
			}
			if got == nil {
				t.Fatalf("want %q, got valid", tc.want)
			}
			if got.Field != tc.field || got.Message != tc.want {
				t.Fatalf("want {%s %q}, got {%s %q}", tc.field, tc.want, got.Field, got.Message)
			}
		})
	}
}

func TestCheckStatus(t *testing.T) {
	for _, v := range []string{"todo", "in_progress", "in_review", "done"} {
		if got := checkStatus(v); got != nil {
			t.Errorf("%q: want valid, got %q", v, got.Message)
		}
	}
	for _, v := range []string{"Done", "todo ", ""} {
		got := checkStatus(v)
		if got == nil || got.Field != "status" || got.Message != "status must be one of todo, in_progress, in_review, done" {
			t.Errorf("%q: want status error, got %+v", v, got)
		}
	}
}

func TestCheckAssignee(t *testing.T) {
	for id := 0; id <= 5; id++ {
		if got := checkAssignee(id); got != nil {
			t.Errorf("%d: want valid, got %q", id, got.Message)
		}
	}
	for _, id := range []int{-1, 6} {
		got := checkAssignee(id)
		if got == nil || got.Field != "assigneeId" || got.Message != "assigneeId must be 0 or a user ID from 1 to 5" {
			t.Errorf("%d: want assignee error, got %+v", id, got)
		}
	}
}

func TestParseConfigDefaults(t *testing.T) {
	port, path, err := parseConfig(nil)
	if err != nil || port != 8080 || path != "taskify.json" {
		t.Fatalf("want 8080 taskify.json nil, got %d %q %v", port, path, err)
	}
}

func TestParseConfigAcceptsPortLimits(t *testing.T) {
	for _, p := range []struct {
		arg  string
		want int
	}{{"1", 1}, {"65535", 65535}} {
		port, _, err := parseConfig([]string{"-port", p.arg})
		if err != nil || port != p.want {
			t.Errorf("-port %s: want %d nil, got %d %v", p.arg, p.want, port, err)
		}
	}
}

func TestParseConfigRefusesBadInput(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		args    []string
		mention string
		secret  string
	}{
		{[]string{"-port", "0"}, "-port", ""},
		{[]string{"-port", "65536"}, "-port", "65536"},
		{[]string{"-port", "abc"}, "-port", "abc"},
		{[]string{"-data", ""}, "-data", ""},
		{[]string{"-data", dir}, "-data", dir},
		{[]string{"-data", filepath.Join(dir, "missing", "x.json")}, "-data", "missing"},
		{[]string{"extra"}, "not allowed", "extra"},
		{[]string{"-addr", "x"}, "not allowed", "addr"},
	}
	for _, tc := range tests {
		_, _, err := parseConfig(tc.args)
		if err == nil {
			t.Errorf("%q: want error, got nil", tc.args)
			continue
		}
		if !strings.Contains(err.Error(), tc.mention) {
			t.Errorf("%q: error %q does not mention %q", tc.args, err, tc.mention)
		}
		if tc.secret != "" && strings.Contains(err.Error(), tc.secret) {
			t.Errorf("%q: error %q repeats the input", tc.args, err)
		}
	}
}
