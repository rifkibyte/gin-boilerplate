package id

import "testing"

func TestNewV7(t *testing.T) {
	id := NewV7()

	if len(id) != 36 {
		t.Fatalf("expected 36-char UUID, got %d: %q", len(id), id)
	}

	// UUID v7 has version 7 in the 13th character (index 14).
	if id[14] != '7' {
		t.Fatalf("expected UUID v7 (version 7), got %q", id)
	}
}

func TestNewV7Unique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		id := NewV7()
		if seen[id] {
			t.Fatalf("duplicate UUID: %s", id)
		}
		seen[id] = true
	}
}

func TestParse(t *testing.T) {
	valid := "01900000-0000-7000-8000-000000000000"
	parsed, err := Parse(valid)
	if err != nil {
		t.Fatalf("Parse(%q): %v", valid, err)
	}
	if parsed != valid {
		t.Fatalf("expected %q, got %q", valid, parsed)
	}

	if _, err := Parse("not-a-uuid"); err == nil {
		t.Fatal("expected error for invalid UUID")
	}
}
