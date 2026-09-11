package main

import (
	"flag"
	"reflect"
	"testing"
)

func testFlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	fs.Bool("json", false, "")
	fs.Float64("lat", 0, "")
	fs.String("city", "", "")
	return fs
}

func TestPermuteLiftsFlagsPastPositionals(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"flags after a subcommand", []string{"get", "--lat", "38.7", "na-pizza"}, []string{"--lat", "38.7", "get", "na-pizza"}},
		{"trailing bool flag", []string{"pizza", "--json"}, []string{"--json", "pizza"}},
		{"equals form", []string{"pizza", "--city=LIS"}, []string{"--city=LIS", "pizza"}},
		{"already in order", []string{"--lat", "38.7", "pizza"}, []string{"--lat", "38.7", "pizza"}},
		{"terminator keeps the rest positional", []string{"--json", "--", "--lat", "x"}, []string{"--json", "--lat", "x"}},
	}
	for _, tc := range cases {
		if got := permute(testFlagSet(), tc.in); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: permute(%v) = %v, want %v", tc.name, tc.in, got, tc.want)
		}
	}
}

func TestParseArgsReadsFlagsAfterPositionals(t *testing.T) {
	fs := testFlagSet()
	if err := parseArgs(fs, []string{"get", "na-pizza", "--lat", "38.7", "--json"}); err != nil {
		t.Fatal(err)
	}
	if got := fs.Lookup("lat").Value.String(); got != "38.7" {
		t.Errorf("lat = %s", got)
	}
	if got := fs.Lookup("json").Value.String(); got != "true" {
		t.Errorf("json = %s", got)
	}
	if want := []string{"get", "na-pizza"}; !reflect.DeepEqual(fs.Args(), want) {
		t.Errorf("args = %v, want %v", fs.Args(), want)
	}
}
