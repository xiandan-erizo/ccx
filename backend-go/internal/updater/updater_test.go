package updater

import (
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"v2.7.0", "v2.6.97", 1},
		{"v2.6.97", "v2.6.97", 0},
		{"v2.6.96", "v2.6.97", -1},
		{"v3.0.0", "v2.99.99", 1},
		{"v1.0.0", "v1.0.0-beta", 0},
		{"2.6.97", "v2.6.97", 0},
	}
	for _, tt := range tests {
		got := compareVersions(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestIsUpgradeableVersion(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"v2.6.97", true},
		{"2.6.97", true},
		{"v0.0.0-dev", false},
		{"unknown", false},
		{"", false},
	}
	for _, tt := range tests {
		got := isUpgradeableVersion(tt.version)
		if got != tt.want {
			t.Errorf("isUpgradeableVersion(%q) = %v, want %v", tt.version, got, tt.want)
		}
	}
}

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"v2.6.97", "2.6.97"},
		{"2.6.97", "2.6.97"},
		{"v1.0.0-beta", "1.0.0"},
		{"v3.0.0-rc.1", "3.0.0"},
	}
	for _, tt := range tests {
		got := normalizeVersion(tt.input)
		if got != tt.want {
			t.Errorf("normalizeVersion(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestBuildAssetName(t *testing.T) {
	name := buildAssetName()
	if name == "" {
		t.Fatal("buildAssetName() returned empty string")
	}
	if !contains(name, "ccx-") {
		t.Errorf("buildAssetName() = %q, expected prefix 'ccx-'", name)
	}
}

func TestDetectDocker(t *testing.T) {
	// Just ensure it doesn't panic
	_ = detectDocker()
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s[:len(substr)] == substr || containsInner(s, substr))
}

func containsInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
