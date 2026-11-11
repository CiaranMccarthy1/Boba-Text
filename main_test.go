package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveStart(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name     string
		arg      string
		wantDir  string
		wantFile string
	}{
		{"no argument", "", cwd, ""},
		{"file", file, dir, file},
		{"directory", dir, dir, ""},
		{"missing file", filepath.Join(dir, "new.txt"), dir, filepath.Join(dir, "new.txt")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotDir, gotFile, err := resolveStart(c.arg)
			if err != nil {
				t.Fatal(err)
			}
			if gotDir != c.wantDir || gotFile != c.wantFile {
				t.Errorf("got (%q, %q), want (%q, %q)", gotDir, gotFile, c.wantDir, c.wantFile)
			}
		})
	}
}
