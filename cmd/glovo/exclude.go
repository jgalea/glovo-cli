package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jgalea/glovo-cli/internal/glovo"
)

// Glovo publishes nothing that marks a store as a chain, a ghost kitchen or
// anything else worth skipping, so the list is the caller's to keep: --exclude
// for one search, ~/.glovo/exclude.txt for the standing one.
func excludeList(flagValue string) []string {
	if strings.TrimSpace(flagValue) != "" {
		var out []string
		for _, p := range strings.Split(flagValue, ",") {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, strings.ToLower(p))
			}
		}
		return out
	}
	return excludeFile()
}

func excludeFile() []string {
	dir := os.Getenv("GLOVO_CONFIG_DIR")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		dir = filepath.Join(home, ".glovo")
	}
	f, err := os.Open(filepath.Join(dir, "exclude.txt"))
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, strings.ToLower(line))
	}
	return out
}

// applyExcludes drops stores whose name matches any pattern, and says how many
// went, so a short result list is never quietly a filtered one.
func applyExcludes(stores []glovo.Store, patterns []string) ([]glovo.Store, int) {
	if len(patterns) == 0 {
		return stores, 0
	}
	kept := make([]glovo.Store, 0, len(stores))
	for _, s := range stores {
		name := strings.ToLower(s.Name)
		skip := false
		for _, p := range patterns {
			if strings.Contains(name, p) {
				skip = true
				break
			}
		}
		if !skip {
			kept = append(kept, s)
		}
	}
	return kept, len(stores) - len(kept)
}

func excludeUsage() string {
	return fmt.Sprintf("comma-separated names to hide, e.g. --exclude %q (defaults to ~/.glovo/exclude.txt)", "domino,pizza hut")
}
