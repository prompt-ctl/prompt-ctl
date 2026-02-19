package discover

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestDiscover_IncludeTxt(t *testing.T) {
	dir := filepath.Join("testdata")
	got, err := Discover(dir, []string{"*.txt"}, nil)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	want := []string{"a.txt", "b.txt", "skip-me.txt"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Discover(%q, []string{\"*.txt\"}, nil) = %q, want %q", dir, got, want)
	}
}

func TestDiscover_IgnoreExcludesMatches(t *testing.T) {
	dir := filepath.Join("testdata")
	got, err := Discover(dir, []string{"*.txt"}, []string{"*skip*"})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	want := []string{"a.txt", "b.txt"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Discover(%q, []string{\"*.txt\"}, []string{\"*skip*\"}) = %q, want %q", dir, got, want)
	}
}
