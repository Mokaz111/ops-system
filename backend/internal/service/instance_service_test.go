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

func TestValidateGrafanaAdminPassword(t *testing.T) {
	cases := []struct {
		name    string
		merged  map[string]interface{}
		wantErr bool
	}{
		{
			name:    "no grafana key",
			merged:  map[string]interface{}{},
			wantErr: false,
		},
		{
			name: "empty password",
			merged: map[string]interface{}{
				"grafana": map[string]interface{}{
					"grafana.ini": map[string]interface{}{
						"security": map[string]interface{}{"admin_password": ""},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "literal placeholder",
			merged: map[string]interface{}{
				"grafana": map[string]interface{}{
					"grafana.ini": map[string]interface{}{
						"security": map[string]interface{}{"admin_password": "${GRAFANA_ADMIN_PASSWORD}"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "real password",
			merged: map[string]interface{}{
				"grafana": map[string]interface{}{
					"grafana.ini": map[string]interface{}{
						"security": map[string]interface{}{"admin_password": "s3cret"},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateGrafanaAdminPassword(c.merged)
			if (err != nil) != c.wantErr {
				t.Fatalf("validateGrafanaAdminPassword err=%v, wantErr=%v", err, c.wantErr)
			}
		})
	}
}
