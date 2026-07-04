package main

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/jgalea/glovo-cli/internal/glovo"
)

const (
	defaultCity = "barcelona"

	defaultSearchLat     = 41.3874
	defaultSearchLng     = 2.1686
	defaultSearchCity    = "BCN"
	defaultSearchCountry = "ES"
)

func cmdSearch(args []string) error {
	fs, c := newCommonFlags("search")
	var latFlag, lngFlag float64
	fs.Float64Var(&latFlag, "lat", math.NaN(), "delivery latitude (falls back to GLOVO_LAT, then Barcelona)")
	fs.Float64Var(&lngFlag, "lng", math.NaN(), "delivery longitude (falls back to GLOVO_LNG, then Barcelona)")
	city := fs.String("city", defaultSearchCity, "delivery city code")
	country := fs.String("country", defaultSearchCountry, "delivery country code")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `usage: glovo search [flags] <query...>

search calls Glovo's authenticated store-search API and needs your OWN
account: run "glovo login" first. It also needs delivery coordinates — pass
--lat/--lng, or set GLOVO_LAT/GLOVO_LNG, or it defaults to Barcelona centre
(41.3874, 2.1686).

FLAGS:
`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	query := strings.Join(fs.Args(), " ")
	if query == "" {
		fs.Usage()
		return fmt.Errorf("usage: glovo search [flags] <query...>")
	}
	lat := resolveCoord(latFlag, "GLOVO_LAT", defaultSearchLat)
	lng := resolveCoord(lngFlag, "GLOVO_LNG", defaultSearchLng)
	cl := glovo.NewClient(stderrLogf)
	stores, err := cl.Search(query, lat, lng, *city, *country)
	if err != nil {
		return err
	}
	var text strings.Builder
	for _, s := range stores {
		text.WriteString(storeLine(s) + "\n")
	}
	return emit(c, stores, strings.TrimRight(text.String(), "\n"))
}

func resolveCoord(flagVal float64, envVar string, def float64) float64 {
	if !math.IsNaN(flagVal) {
		return flagVal
	}
	if v := os.Getenv(envVar); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}
