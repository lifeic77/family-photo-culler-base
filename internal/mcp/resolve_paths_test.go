package mcp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCanonicalCollectionPath(t *testing.T) {
	root := t.TempDir()
	insideDir := filepath.Join(root, "photos")
	outsideDir := filepath.Join(root, "outside")
	if err := os.MkdirAll(insideDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outsideDir, 0o755); err != nil {
		t.Fatal(err)
	}
	inside := filepath.Join(insideDir, "a.jpg")
	outside := filepath.Join(outsideDir, "b.jpg")
	if err := os.WriteFile(inside, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := canonicalCollectionPath(inside, []string{insideDir})
	if err != nil {
		t.Fatalf("inside path rejected: %v", err)
	}
	want, _ := filepath.EvalSymlinks(inside)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}

	if _, err := canonicalCollectionPath(outside, []string{insideDir}); err == nil {
		t.Fatal("outside path accepted")
	}
	if _, err := canonicalCollectionPath("relative.jpg", []string{insideDir}); err == nil {
		t.Fatal("relative path accepted")
	}
}

func TestCanonicalCollectionPathRejectsEscapingSymlink(t *testing.T) {
	root := t.TempDir()
	insideDir := filepath.Join(root, "photos")
	outsideDir := filepath.Join(root, "outside")
	if err := os.MkdirAll(insideDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outsideDir, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(outsideDir, "secret.jpg")
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(insideDir, "link.jpg")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := canonicalCollectionPath(link, []string{insideDir}); err == nil {
		t.Fatal("symlink escaping collection was accepted")
	}
}
