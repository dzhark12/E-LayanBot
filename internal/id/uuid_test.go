package id

import (
	"regexp"
	"testing"
)

func TestUUIDFormatAndUniqueness(t *testing.T) {
	pattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	first, err := UUID()
	if err != nil {
		t.Fatal(err)
	}
	second, err := UUID()
	if err != nil {
		t.Fatal(err)
	}
	if !pattern.MatchString(first) || !pattern.MatchString(second) {
		t.Fatalf("UUIDs have invalid format: %q, %q", first, second)
	}
	if first == second {
		t.Fatalf("UUID() returned the same value twice: %q", first)
	}
}
