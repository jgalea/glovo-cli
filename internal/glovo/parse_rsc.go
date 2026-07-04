package glovo

import (
	"encoding/json"
	"strings"
)

// extractNextChunks pulls every self.__next_f.push([1,"..."]) string chunk out of
// the page and concatenates the unescaped contents into one blob.
func extractNextChunks(html string) string {
	const marker = "self.__next_f.push([1,"
	var b strings.Builder
	i := 0
	for {
		j := strings.Index(html[i:], marker)
		if j < 0 {
			break
		}
		i += j + len(marker)
		// The quote must immediately follow the marker (past whitespace); an
		// unbounded search here could latch onto an unrelated quote much
		// later in the document if a marker is ever malformed.
		k := i
		for k < len(html) && (html[k] == ' ' || html[k] == '\t' || html[k] == '\n' || html[k] == '\r') {
			k++
		}
		if k >= len(html) || html[k] != '"' {
			continue
		}
		// The next JSON value is a quoted string; find its bounds respecting escapes.
		start := k
		end := scanQuotedString(html, start)
		if end < 0 {
			break
		}
		var chunk string
		if err := json.Unmarshal([]byte(html[start:end]), &chunk); err == nil {
			b.WriteString(chunk)
		}
		i = end
	}
	return b.String()
}

// scanQuotedString returns the index just past the closing quote of the JSON
// string starting at html[start] == '"', honoring backslash escapes.
func scanQuotedString(html string, start int) int {
	for i := start + 1; i < len(html); i++ {
		switch html[i] {
		case '\\':
			i++
		case '"':
			return i + 1
		}
	}
	return -1
}

// scanJSONObjects walks blob and returns every balanced {...} object that both
// parses as JSON and contains mustHaveKey at the top level.
func scanJSONObjects(blob, mustHaveKey string) []map[string]any {
	var out []map[string]any
	for i := 0; i < len(blob); i++ {
		if blob[i] != '{' {
			continue
		}
		end := matchBrace(blob, i)
		if end < 0 {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(blob[i:end]), &m); err == nil {
			if _, ok := m[mustHaveKey]; ok {
				out = append(out, m)
				i = end - 1
			}
		}
	}
	return out
}

// matchBrace returns the index just past the '}' matching blob[open] == '{',
// tracking string literals so braces inside strings do not count.
func matchBrace(blob string, open int) int {
	depth := 0
	inStr := false
	for i := open; i < len(blob); i++ {
		ch := blob[i]
		if inStr {
			if ch == '\\' {
				i++
			} else if ch == '"' {
				inStr = false
			}
			continue
		}
		switch ch {
		case '"':
			inStr = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return -1
}
