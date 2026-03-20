package versionutils_test

import (
	"reflect"
	"testing"

	"github.com/kamilludwinski/runtimzzz/internal/utils/versionutils"
)

func TestResolveVersionLatest(t *testing.T) {
	candidates := []string{"1.2.3", "2.0.0", "1.10.0"}

	got, ok := versionutils.ResolveVersion("latest", candidates)
	if !ok {
		t.Fatalf("expected match for latest, got none")
	}
	if want := "2.0.0"; got != want {
		t.Fatalf("ResolveVersion(latest) = %q, want %q", got, want)
	}
}

func TestResolveVersionExactMatch(t *testing.T) {
	candidates := []string{"1.2.3", "1.2.4"}

	got, ok := versionutils.ResolveVersion("1.2.3", candidates)
	if !ok || got != "1.2.3" {
		t.Fatalf("expected exact match 1.2.3, got %q (ok=%v)", got, ok)
	}

	if _, ok := versionutils.ResolveVersion("1.2.5", candidates); ok {
		t.Fatalf("expected no match for 1.2.5, got ok=true")
	}
}

func TestResolveVersionPrefixMajorMinor(t *testing.T) {
	candidates := []string{"1.2.0", "1.2.3", "1.3.0", "2.0.0"}

	got, ok := versionutils.ResolveVersion("1.2", candidates)
	if !ok {
		t.Fatalf("expected match for 1.2, got none")
	}
	if want := "1.2.3"; got != want {
		t.Fatalf("ResolveVersion(1.2) = %q, want %q", got, want)
	}
}

func TestResolveVersionPrefixMajor(t *testing.T) {
	candidates := []string{"0.9.9", "1.0.0", "1.2.3", "2.0.0"}

	got, ok := versionutils.ResolveVersion("1", candidates)
	if !ok {
		t.Fatalf("expected match for 1, got none")
	}
	if want := "1.2.3"; got != want {
		t.Fatalf("ResolveVersion(1) = %q, want %q", got, want)
	}
}

func TestResolveVersionNoCandidatesOrMalformed(t *testing.T) {
	if _, ok := versionutils.ResolveVersion("latest", nil); ok {
		t.Fatalf("expected no match for latest with no candidates")
	}
	if _, ok := versionutils.ResolveVersion("", []string{"1.0.0"}); ok {
		t.Fatalf("expected no match for empty input")
	}
	if _, ok := versionutils.ResolveVersion("abc", []string{"1.0.0"}); ok {
		t.Fatalf("expected no match for malformed version")
	}
	if _, ok := versionutils.ResolveVersion("1.2.3.4", []string{"1.2.3"}); ok {
		t.Fatalf("expected no match for 1.2.3.4")
	}
}

func TestSortVersions(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		desc     bool
		expected []string
	}{
		{
			name:     "ascending basic",
			input:    []string{"1.2.10", "1.3.0", "1.2.2", "2.0.0"},
			desc:     false,
			expected: []string{"1.2.2", "1.2.10", "1.3.0", "2.0.0"},
		},
		{
			name:     "descending basic",
			input:    []string{"1.2.10", "1.3.0", "1.2.2", "2.0.0"},
			desc:     true,
			expected: []string{"2.0.0", "1.3.0", "1.2.10", "1.2.2"},
		},
		{
			name:     "same major different minor",
			input:    []string{"1.10.0", "1.2.0", "1.1.5"},
			desc:     false,
			expected: []string{"1.1.5", "1.2.0", "1.10.0"},
		},
		{
			name:     "same major minor different patch",
			input:    []string{"1.2.3", "1.2.10", "1.2.1"},
			desc:     false,
			expected: []string{"1.2.1", "1.2.3", "1.2.10"},
		},
		{
			name:     "already sorted ascending",
			input:    []string{"1.0.0", "1.1.0", "1.2.0"},
			desc:     false,
			expected: []string{"1.0.0", "1.1.0", "1.2.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := make([]string, len(tt.input))
			copy(input, tt.input)

			versionutils.SortVersions(input, tt.desc)

			if !reflect.DeepEqual(input, tt.expected) {
				t.Fatalf("expected %v, got %v", tt.expected, input)
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.0", 1},
		{"0.3.0", "1.0.0", -1},
		{"2.0.0", "1.9.9", 1},
	}
	for _, tt := range tests {
		got := versionutils.CompareVersions(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
