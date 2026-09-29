package mcp

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"photofield/internal/image"
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

func TestApplyResolvedPhotoInfoUsesCachedInfoAndConstructsPreviewURL(t *testing.T) {
	item := resolvedPhotoPath{Path: "/photos/Trip/IMG_0123.JPG", FileId: 42}
	info := image.Info{
		Width:    6000,
		Height:   4000,
		DateTime: time.Date(2026, 9, 29, 14, 3, 2, 0, time.FixedZone("CST", 8*60*60)),
	}
	applyResolvedPhotoInfo(&item, info, "http://127.0.0.1:8080", "/api")
	if item.Width != 6000 || item.Height != 4000 {
		t.Fatalf("dimensions=%dx%d", item.Width, item.Height)
	}
	if item.CreatedAt != "2026-09-29T14:03:02+08:00" {
		t.Fatalf("created_at=%q", item.CreatedAt)
	}
	want := "http://127.0.0.1:8080/api/files/42/previews/IMG_0123_preview.jpg?w=400"
	if item.PreviewUrl != want {
		t.Fatalf("preview_url=%q want %q", item.PreviewUrl, want)
	}
	wantCached := "http://127.0.0.1:8080/api/files/42/previews/IMG_0123_preview.jpg?cache_only=true"
	if item.CachedPreviewUrl != wantCached {
		t.Fatalf("cached_preview_url=%q want %q", item.CachedPreviewUrl, wantCached)
	}
}
