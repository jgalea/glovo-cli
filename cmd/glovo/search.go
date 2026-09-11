package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/jgalea/glovo-cli/internal/glovo"
)

func cmdSearch(args []string) error {
	fs, c := newCommonFlags("search")
	loc := addLocationFlags(fs)
	exclude := fs.String("exclude", "", excludeUsage())
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `usage: glovo search [flags] <query...>

search calls Glovo's authenticated store-search API and needs your OWN
account: run "glovo login" first. It also needs delivery coordinates — pass
--lat/--lng, or set GLOVO_LAT/GLOVO_LNG, or it defaults to Barcelona centre
(41.3874, 2.1686). The city and country codes are resolved from those
coordinates unless you pass them.

--exclude hides stores by name, for the places you never want to see.
Put a standing list in ~/.glovo/exclude.txt, one name per line.

FLAGS:
`)
		fs.PrintDefaults()
	}
	if err := parseArgs(fs, args); err != nil {
		return err
	}
	query := strings.Join(fs.Args(), " ")
	if query == "" {
		fs.Usage()
		return fmt.Errorf("usage: glovo search [flags] <query...>")
	}
	cl := glovo.NewClient(stderrLogf)
	where, err := loc.resolve(cl)
	if err != nil {
		return err
	}
	stores, err := cl.Search(query, where.Lat, where.Lng, where.CityCode, where.CountryCode)
	if err != nil {
		return err
	}
	stores, hidden := applyExcludes(stores, excludeList(*exclude))
	if hidden > 0 {
		stderrLogf("hid %d store(s) matching your exclude list", hidden)
	}
	var text strings.Builder
	for _, s := range stores {
		text.WriteString(storeLine(s) + "\n")
	}
	return emit(c, stores, strings.TrimRight(text.String(), "\n"))
}
