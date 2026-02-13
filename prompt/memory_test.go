package prompt

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestListPromptsInDir_Empty(t *testing.T) {
	dir := t.TempDir()
	entries, err := ListPromptsInDir(dir)
	if err != nil {
		t.Fatalf("ListPromptsInDir() err = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestListPromptsInDir_Nonexistent(t *testing.T) {
	entries, err := ListPromptsInDir(filepath.Join(t.TempDir(), "nonexistent"))
	if err != nil {
		t.Fatalf("ListPromptsInDir() err = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for nonexistent dir, got %d", len(entries))
	}
}

func TestListPromptsInDir_Flat(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a", "b", "review"} {
		path := filepath.Join(dir, name+".yaml")
		if err := os.WriteFile(path, []byte("name: "+name+"\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := ListPromptsInDir(dir)
	if err != nil {
		t.Fatalf("ListPromptsInDir() err = %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Folder != entries[j].Folder {
			return entries[i].Folder < entries[j].Folder
		}
		return entries[i].Name < entries[j].Name
	})
	for i, want := range []string{"a", "b", "review"} {
		if entries[i].Name != want || entries[i].Folder != "" {
			t.Errorf("entry %d: Name=%q Folder=%q, want Name=%q Folder=\"\"", i, entries[i].Name, entries[i].Folder, want)
		}
	}
}

func TestListPromptsInDir_WithSubdirs(t *testing.T) {
	dir := t.TempDir()
	// flat
	if err := os.WriteFile(filepath.Join(dir, "root.yaml"), []byte("name: root\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// one subdir
	sub := filepath.Join(dir, "work")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "task.yaml"), []byte("name: task\n"), 0644); err != nil {
		t.Fatal(err)
	}
	entries, err := ListPromptsInDir(dir)
	if err != nil {
		t.Fatalf("ListPromptsInDir() err = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Folder != entries[j].Folder {
			return entries[i].Folder < entries[j].Folder
		}
		return entries[i].Name < entries[j].Name
	})
	if entries[0].Name != "root" || entries[0].Folder != "" {
		t.Errorf("first entry: Name=%q Folder=%q, want root and \"\"", entries[0].Name, entries[0].Folder)
	}
	if entries[1].Name != "task" || entries[1].Folder != "work" {
		t.Errorf("second entry: Name=%q Folder=%q, want task and work", entries[1].Name, entries[1].Folder)
	}
}
