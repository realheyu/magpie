package configapi

import "testing"

func TestBuildETagChangesWithVersionAndContent(t *testing.T) {
	first := buildETag("app1-dev", 1, "a: 1")
	if first == "" {
		t.Fatal("etag should not be empty")
	}
	if first != buildETag("app1-dev", 1, "a: 1") {
		t.Fatal("etag should be stable for same input")
	}
	if first == buildETag("app1-dev", 2, "a: 1") {
		t.Fatal("etag should change when version changes")
	}
	if first == buildETag("app1-dev", 1, "a: 2") {
		t.Fatal("etag should change when content changes")
	}
}
