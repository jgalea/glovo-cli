package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jgalea/glovo-cli/internal/glovo"
)

func TestApplyExcludesMatchesNamesCaseInsensitively(t *testing.T) {
	stores := []glovo.Store{
		{Name: "Domino's Pizza"},
		{Name: "Na Pizza"},
		{Name: "PIZZA HUT"},
		{Name: "Pizzaghetti"},
	}
	kept, hidden := applyExcludes(stores, []string{"domino", "pizza hut"})
	if hidden != 2 {
		t.Fatalf("hidden = %d, want 2", hidden)
	}
	var names []string
	for _, s := range kept {
		names = append(names, s.Name)
	}
	if want := []string{"Na Pizza", "Pizzaghetti"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("kept = %v, want %v", names, want)
	}
}

func TestApplyExcludesWithNoPatternsKeepsEverything(t *testing.T) {
	stores := []glovo.Store{{Name: "Na Pizza"}, {Name: "Domino's Pizza"}}
	kept, hidden := applyExcludes(stores, nil)
	if hidden != 0 || len(kept) != 2 {
		t.Fatalf("kept %d, hidden %d", len(kept), hidden)
	}
}

func TestExcludeListPrefersTheFlagOverTheFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GLOVO_CONFIG_DIR", dir)
	if err := os.WriteFile(filepath.Join(dir, "exclude.txt"), []byte("# chains\nTelepizza\n\n  Pizza Hut  \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := excludeList(""); !reflect.DeepEqual(got, []string{"telepizza", "pizza hut"}) {
		t.Errorf("from file = %v", got)
	}
	if got := excludeList("Domino, KFC"); !reflect.DeepEqual(got, []string{"domino", "kfc"}) {
		t.Errorf("from flag = %v", got)
	}
}

func TestExcludeListNoneIgnoresTheStandingFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GLOVO_CONFIG_DIR", dir)
	if err := os.WriteFile(filepath.Join(dir, "exclude.txt"), []byte("Telepizza\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := excludeList("none"); len(got) != 0 {
		t.Errorf("got %v, want nothing", got)
	}
}

func TestExcludeListIsEmptyWithoutAFile(t *testing.T) {
	t.Setenv("GLOVO_CONFIG_DIR", t.TempDir())
	if got := excludeList(""); len(got) != 0 {
		t.Errorf("got %v, want nothing", got)
	}
}
