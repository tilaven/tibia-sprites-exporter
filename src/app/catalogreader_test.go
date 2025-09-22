package app

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempFile(t *testing.T, dir, name, contents string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile %s: %v", path, err)
	}
	return path
}

func TestStreamCatalogContentReadsElementsSequentially(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "catalog.json", `[
                {"type":"sprite","file":"foo.png","spritetype":1,"firstspriteid":100,"lastspriteid":101,"area":64},
                {"type":"effect","file":"bar.png","spritetype":2,"firstspriteid":200,"lastspriteid":205,"area":128}
        ]`)

	out, errs := StreamCatalogContent(path)

	var got []CatalogElem
	for elem := range out {
		got = append(got, elem)
	}

	if err, ok := <-errs; ok && err != nil {
		t.Fatalf("StreamCatalogContent error: %v", err)
	}

	want := []CatalogElem{
		{Type: "sprite", File: "foo.png", SpriteType: 1, FirstSpriteId: 100, LastSpriteId: 101, Area: 64},
		{Type: "effect", File: "bar.png", SpriteType: 2, FirstSpriteId: 200, LastSpriteId: 205, Area: 128},
	}

	if len(got) != len(want) {
		t.Fatalf("got %d elements, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("element %d = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func TestStreamCatalogContentReturnsErrorForInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "invalid.json", `{"type":"sprite"}`)

	out, errs := StreamCatalogContent(path)

	for range out {
		t.Fatalf("unexpected element emitted for invalid JSON")
	}

	if err, ok := <-errs; !ok || err == nil {
		t.Fatalf("expected error for invalid JSON, got %v (ok=%v)", err, ok)
	}
}

func TestCountSpriteEntries(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "catalog.json", `
                {"type":"effect"}
                ["type": "not-sprite"]
                {"type" :    "sprite"}
                {"type":"sprite"}
                {"type"  :"sprite"}
        `)

	count, err := CountSpriteEntries(path)
	if err != nil {
		t.Fatalf("CountSpriteEntries error: %v", err)
	}

	const want = 3
	if count != want {
		t.Fatalf("CountSpriteEntries = %d, want %d", count, want)
	}
}

func TestStreamCatalogContentReturnsErrorWhenFileMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.json")

	out, errs := StreamCatalogContent(path)

	for range out {
		t.Fatalf("unexpected element emitted for missing file")
	}

	if err, ok := <-errs; !ok || err == nil {
		t.Fatalf("expected error for missing file, got %v (ok=%v)", err, ok)
	}
}
