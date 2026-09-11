package main

import (
	"flag"
	"math"
	"os"
	"strconv"

	"github.com/jgalea/glovo-cli/internal/glovo"
)

// Barcelona centre, used when no coordinates are given anywhere.
const (
	defaultLat = 41.3874
	defaultLng = 2.1686
)

type locFlags struct {
	lat, lng float64
	city     string
	country  string
	citySlug string
}

func addLocationFlags(fs *flag.FlagSet) *locFlags {
	l := &locFlags{}
	fs.Float64Var(&l.lat, "lat", math.NaN(), "delivery latitude (falls back to GLOVO_LAT, then Barcelona)")
	fs.Float64Var(&l.lng, "lng", math.NaN(), "delivery longitude (falls back to GLOVO_LNG, then Barcelona)")
	fs.StringVar(&l.city, "city", "", "delivery city code, e.g. LIS (resolved from the coordinates when unset)")
	fs.StringVar(&l.country, "country", "", "delivery country code, e.g. PT (resolved from the coordinates when unset)")
	fs.StringVar(&l.citySlug, "city-slug", "", "city segment of Glovo's store URLs, e.g. lisboa (resolved when unset)")
	return l
}

// resolve fills in whatever the user didn't pass, asking Glovo which city
// serves the coordinates.
func (l *locFlags) resolve(cl *glovo.Client) (glovo.Location, error) {
	loc := glovo.Location{
		Lat:         resolveCoord(l.lat, "GLOVO_LAT", defaultLat),
		Lng:         resolveCoord(l.lng, "GLOVO_LNG", defaultLng),
		CityCode:    l.city,
		CountryCode: l.country,
		CitySlug:    l.citySlug,
	}
	if loc.CityCode == "" || loc.CountryCode == "" {
		city, country, err := cl.ResolveCityCodes(loc.Lat, loc.Lng)
		if err != nil {
			return loc, err
		}
		if loc.CityCode == "" {
			loc.CityCode = city
		}
		if loc.CountryCode == "" {
			loc.CountryCode = country
		}
	}
	return loc, nil
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
