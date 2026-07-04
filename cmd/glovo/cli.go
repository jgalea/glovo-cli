package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	toon "github.com/toon-format/toon-go"
)

var stderrLogf = func(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "glovo: "+format+"\n", args...)
}

type common struct {
	jsonOut bool
	toon    bool
}

func newCommonFlags(name string) (*flag.FlagSet, *common) {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	c := &common{}
	fs.BoolVar(&c.jsonOut, "json", false, "emit raw JSON to stdout")
	fs.BoolVar(&c.toon, "toon", false, "emit TOON to stdout")
	return fs, c
}

func emit(c *common, data any, text string) error {
	switch {
	case c.jsonOut:
		b, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
	case c.toon:
		s, err := toon.Marshal(data)
		if err != nil {
			return err
		}
		fmt.Println(s)
	default:
		fmt.Println(text)
	}
	return nil
}
