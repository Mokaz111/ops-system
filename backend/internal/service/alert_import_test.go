package service

import "testing"

func TestSeverityToLevel(t *testing.T) {
	cases := map[string]string{
		"critical":  "critical",
		"Page":      "critical",
		"error":     "critical",
		"warning":   "warning",
		"major":     "warning",
		"":          "warning",
		"custom":    "warning",
		"info":      "info",
		"none":      "info",
		" Critical": "critical",
	}
	for in, want := range cases {
		if got := severityToLevel(in); got != want {
			t.Errorf("severityToLevel(%q) = %q, want %q", in, got, want)
		}
	}
}
