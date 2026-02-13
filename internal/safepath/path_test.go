package safepath

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSafeFilePath(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	_ = os.MkdirAll(sub, 0755)
	f := filepath.Join(sub, "f.txt")
	_ = os.WriteFile(f, []byte("x"), 0644)

	// under base: ok
	got, err := SafeFilePath(dir, filepath.Join("sub", "f.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got != f {
		t.Errorf("got %q want %q", got, f)
	}

	// absolute under base: ok
	got, err = SafeFilePath(dir, f)
	if err != nil {
		t.Fatal(err)
	}
	if got != f {
		t.Errorf("got %q want %q", got, f)
	}

	// path outside base: rejected
	_, err = SafeFilePath(dir, filepath.Join(dir, "..", "..", "etc", "passwd"))
	if err != ErrPathOutsideBase {
		t.Errorf("want ErrPathOutsideBase, got %v", err)
	}

	// directory given as file: error
	_, err = SafeFilePath(dir, "sub")
	if err == nil {
		t.Error("expected error when path is directory")
	}
}

func TestSafeDirPath(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	_ = os.MkdirAll(sub, 0755)

	got, err := SafeDirPath(dir, "sub")
	if err != nil {
		t.Fatal(err)
	}
	if got != sub {
		t.Errorf("got %q want %q", got, sub)
	}

	_, err = SafeDirPath(dir, filepath.Join("..", ".."))
	if err != ErrPathOutsideBase {
		t.Errorf("want ErrPathOutsideBase, got %v", err)
	}
}
