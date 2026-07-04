package glovo

import (
	"os"
	"strings"
	"testing"
)

func TestExtractAndScan(t *testing.T) {
	html, err := os.ReadFile("testdata/next_chunks_min.html")
	if err != nil {
		t.Fatal(err)
	}
	blob := extractNextChunks(string(html))
	if len(blob) == 0 {
		t.Fatal("empty blob")
	}
	objs := scanJSONObjects(blob, "storeProductId")
	if len(objs) != 2 {
		t.Fatalf("got %d objects, want 2", len(objs))
	}
	if objs[0]["name"] != "Boscaiola" {
		t.Fatalf("obj0 name = %v", objs[0]["name"])
	}
	if int64(objs[1]["storeProductId"].(float64)) != 99 {
		t.Fatalf("obj1 id = %v", objs[1]["storeProductId"])
	}
}

func TestScanJSONObjectsBraceInsideString(t *testing.T) {
	blob := `noise {"storeProductId":7,"note":"contains { and } inside a string"} more noise`
	objs := scanJSONObjects(blob, "storeProductId")
	if len(objs) != 1 {
		t.Fatalf("got %d objects, want 1", len(objs))
	}
	if objs[0]["note"] != "contains { and } inside a string" {
		t.Fatalf("note = %v", objs[0]["note"])
	}
}

func TestExtractNextChunksSkipsMarkerWithoutQuote(t *testing.T) {
	html := `<script>self.__next_f.push([1,"{\"storeProductId\":42}"])</script>` +
		`<script>self.__next_f.push([1,</script>` +
		`<div data-note="a distant unrelated quoted value that should never be reached"></div>`
	blob := extractNextChunks(html)
	if blob != `{"storeProductId":42}` {
		t.Fatalf("blob = %q, want well-formed chunk only", blob)
	}
	if strings.Contains(blob, "distant unrelated") {
		t.Fatalf("blob latched onto distant quoted text: %q", blob)
	}
}
