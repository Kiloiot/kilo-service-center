package blueprints

import "testing"

func TestIsCommitConflict422(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected bool
	}{
		{"fast forward conflict", `{"message":"Update is not a fast forward"}`, true},
		{"SHA mismatch", `{"message":"main is at abc123 but expected def456"}`, true},
		{"non-conflict 422", `{"message":"path contains a malformed path component"}`, false},
		{"sha not supplied", `{"message":"Invalid request.\n\n\"sha\" wasn't supplied."}`, true},
		{"case-insensitive fast forward", `{"message":"UPDATE IS NOT A FAST FORWARD"}`, true},
		{"sha only no conflict", `{"message":"sha"}`, false},
		{"empty body", ``, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isCommitConflict422([]byte(tc.body))
			if got != tc.expected {
				t.Errorf("isCommitConflict422(%q) = %v, want %v", tc.body, got, tc.expected)
			}
		})
	}
}
