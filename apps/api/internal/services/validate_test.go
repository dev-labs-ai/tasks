package services

import (
	"strings"
	"testing"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		want      string
		wantValid bool
	}{
		{"plain", "ana@example.com", "ana@example.com", true},
		{"mixed case is lowercased", "Ana.Silva@Example.COM", "ana.silva@example.com", true},
		{"surrounding spaces are trimmed", "  ana@example.com\t", "ana@example.com", true},
		{"254 characters", strings.Repeat("a", 64) + "@" + strings.Repeat("b", 185) + ".com", strings.Repeat("a", 64) + "@" + strings.Repeat("b", 185) + ".com", true},
		{"255 characters", strings.Repeat("a", 64) + "@" + strings.Repeat("b", 186) + ".com", "", false},
		{"empty", "", "", false},
		{"only spaces", "   ", "", false},
		{"display name", "Ana <ana@example.com>", "", false},
		{"no at sign", "ana.example.com", "", false},
		{"no domain", "ana@", "", false},
		{"two addresses", "ana@example.com, bob@example.com", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, problem := ValidateEmail(tt.raw)
			if (problem == "") != tt.wantValid {
				t.Fatalf("ValidateEmail(%q) problem = %q, want valid = %v", tt.raw, problem, tt.wantValid)
			}
			if got != tt.want {
				t.Errorf("ValidateEmail(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name      string
		password  string
		wantValid bool
	}{
		{"7 characters", "abcdefg", false},
		{"8 characters", "abcdefgh", true},
		{"64 characters", strings.Repeat("a", 64), true},
		{"65 characters", strings.Repeat("a", 65), false},
		{"8 emoji count as 8", strings.Repeat("🔑", 8), true},
		{"64 emoji count as 64", strings.Repeat("🔑", 64), true},
		{"spaces are not trimmed", "  abcdef", true},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if problem := ValidatePassword(tt.password); (problem == "") != tt.wantValid {
				t.Errorf("problem = %q, want valid = %v", problem, tt.wantValid)
			}
		})
	}
}

func TestValidateTitle(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		want      string
		wantValid bool
	}{
		{"plain", "Buy milk", "Buy milk", true},
		{"trimmed", "\t Buy milk \n", "Buy milk", true},
		{"inner spaces kept", "Buy  milk", "Buy  milk", true},
		{"empty", "", "", false},
		{"whitespace only", " \t\n 　", "", false},
		{"200 characters", strings.Repeat("a", 200), strings.Repeat("a", 200), true},
		{"201 characters", strings.Repeat("a", 201), "", false},
		{"200 emoji", strings.Repeat("😀", 200), strings.Repeat("😀", 200), true},
		{"201 emoji", strings.Repeat("😀", 201), "", false},
		{"200 after trimming", "  " + strings.Repeat("a", 200) + "  ", strings.Repeat("a", 200), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, problem := ValidateTitle(tt.raw)
			if (problem == "") != tt.wantValid {
				t.Fatalf("problem = %q, want valid = %v", problem, tt.wantValid)
			}
			if got != tt.want {
				t.Errorf("title = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseStatus(t *testing.T) {
	tests := []struct {
		raw       string
		want      TaskStatus
		wantValid bool
	}{
		{"", StatusAll, true},
		{"pending", StatusPending, true},
		{"completed", StatusCompleted, true},
		{"all", StatusAll, false},
		{"Pending", StatusAll, false},
		{"done", StatusAll, false},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, problem := ParseStatus(tt.raw)
			if (problem == "") != tt.wantValid || got != tt.want {
				t.Errorf("ParseStatus(%q) = %v, %q; want %v, valid = %v", tt.raw, got, problem, tt.want, tt.wantValid)
			}
		})
	}
}

func TestFieldErrors(t *testing.T) {
	f := fieldErrors{}
	f.add("email", "")
	if f.err() != nil {
		t.Fatal("no problems, want a nil error")
	}

	f.add("password", "too short")
	f.add("email", "invalid")
	err := f.err()
	ve, ok := err.(*ValidationError)
	if !ok || len(ve.Fields) != 2 {
		t.Fatalf("err = %v, want a *ValidationError with two fields", err)
	}
	if got := err.Error(); got != "invalid fields: email, password" {
		t.Errorf("Error() = %q", got)
	}
}
