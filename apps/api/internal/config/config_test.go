package config

import "testing"

func env(vars map[string]string) func(string) string {
	return func(key string) string { return vars[key] }
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(env(map[string]string{"DATABASE_URL": "postgres://localhost/tasks"}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := Config{DatabaseURL: "postgres://localhost/tasks", Port: 8080, Env: EnvDevelopment}
	if cfg != want {
		t.Errorf("got %+v, want %+v", cfg, want)
	}
	if cfg.IsProduction() {
		t.Error("IsProduction() = true for the default environment")
	}
}

func TestLoadExplicitValues(t *testing.T) {
	cfg, err := Load(env(map[string]string{
		"DATABASE_URL": "postgres://db/tasks",
		"PORT":         "9090",
		"APP_ENV":      "production",
	}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := Config{DatabaseURL: "postgres://db/tasks", Port: 9090, Env: EnvProduction}
	if cfg != want {
		t.Errorf("got %+v, want %+v", cfg, want)
	}
	if !cfg.IsProduction() {
		t.Error("IsProduction() = false for APP_ENV=production")
	}
}

func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name string
		vars map[string]string
	}{
		{"missing DATABASE_URL", map[string]string{}},
		{"non-numeric PORT", map[string]string{"DATABASE_URL": "postgres://db", "PORT": "http"}},
		{"PORT zero", map[string]string{"DATABASE_URL": "postgres://db", "PORT": "0"}},
		{"PORT above range", map[string]string{"DATABASE_URL": "postgres://db", "PORT": "65536"}},
		{"unknown APP_ENV", map[string]string{"DATABASE_URL": "postgres://db", "APP_ENV": "prod"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Load(env(tt.vars)); err == nil {
				t.Error("Load succeeded, want an error")
			}
		})
	}
}
