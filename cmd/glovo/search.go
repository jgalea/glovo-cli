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
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `usage: glovo search [flags] <query...>

search calls Glovo's authenticated store-search API and needs your OWN
account: run "glovo login" first. It also needs delivery coordinates — pass
--lat/--lng, or set GLOVO_LAT/GLOVO_LNG, or it defaults to Barcelona centre
(41.3874, 2.1686). The city and country codes are resolved from those
coordinates unless you pass them.

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
	var text strings.Builder
	for _, s := range stores {
		text.WriteString(storeLine(s) + "\n")
	}
	return emit(c, stores, strings.TrimRight(text.String(), "\n"))
}
