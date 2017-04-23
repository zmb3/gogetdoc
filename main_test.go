package main

import "testing"

func TestParseValidPos(t *testing.T) {
	fname, offset, err := parsePos("foo.go:#123")
	if fname != "foo.go" {
		t.Errorf("want foo.go, got %v", fname)
	}
	if offset != 123 {
		t.Errorf("want 123, got %v", 123)
	}
	if err != nil {
		t.Error(err)
	}
}

func TestParseEmptyPos(t *testing.T) {
	_, _, err := parsePos("")
	if err == nil {
		t.Error("expected error")
	}
}

func TestParseInvalidPos(t *testing.T) {
	for _, input := range []string{
		"foo.go:123",
		"foo.go#123",
		"foo.go#:123",
		"123",
		"foo.go::123",
		"foo.go##123",
		"#:123",
	} {
		if _, _, err := parsePos(input); err == nil {
			t.Errorf("expected %v to be invalid", input)
		}
	}
}
