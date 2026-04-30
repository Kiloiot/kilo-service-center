package version

import (
	"testing"
)

func TestGet(t *testing.T) {
	info, err := Get()
	if err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}
	if info == nil {
		t.Fatal("Get() returned nil Info")
	}

	// Manifest should have basic fields
	if info.Version == "" {
		t.Error("Version is empty")
	}
	if info.BuildTime == "" {
		t.Error("BuildTime is empty")
	}
	if info.SchemaVersion <= 0 {
		t.Errorf("SchemaVersion is %d, want > 0", info.SchemaVersion)
	}
}

func TestString(t *testing.T) {
	info, err := Get()
	if err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}

	str := info.String()
	if str == "" {
		t.Error("String() returned empty string")
	}
	if !contains(str, info.Version) {
		t.Errorf("String() = %q, want to contain %q", str, info.Version)
	}
	if !contains(str, "built") {
		t.Errorf("String() = %q, want to contain 'built'", str)
	}
}

func TestIsProduction(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"1.0.0", true},
		{"v2.3.4", true},
		{"dev", false},
		{"dev-local", false},
		{"1.0.0-dirty", false},
		{"v1.0.0-rc1-8-g90e07cd", true},
		{"v1.0.0-rc1-8-g90e07cd-dirty", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			info := &Info{Version: tt.version}
			got := info.IsProduction()
			if got != tt.want {
				t.Errorf("IsProduction() for version %q = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestGetSingleton(t *testing.T) {
	// Call Get() multiple times, should return same instance
	info1, err1 := Get()
	if err1 != nil {
		t.Fatalf("First Get() returned error: %v", err1)
	}

	info2, err2 := Get()
	if err2 != nil {
		t.Fatalf("Second Get() returned error: %v", err2)
	}

	// Should be same instance (pointer equality)
	if info1 != info2 {
		t.Error("Get() did not return singleton instance")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || s[0:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
