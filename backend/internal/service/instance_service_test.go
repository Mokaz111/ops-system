package service

import "testing"

func TestEscapePromQLLabel(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"normal-name", "normal-name"},
		{`name"with"quotes`, `name\"with\"quotes`},
		{`back\slash`, `back\\slash`},
		{"multi\nline", `multi\nline`},
		{`"; drop`, `\"; drop`},
	}
	for _, c := range cases {
		got := escapePromQLLabel(c.in)
		if got != c.want {
			t.Errorf("escapePromQLLabel(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
