package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

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

// parseArgs parses flags that may appear anywhere on the line. Go's flag
// package stops at the first positional argument, which would drop the flags
// in "glovo cart get --lat 1 <slug>".
func parseArgs(fs *flag.FlagSet, args []string) error {
	return fs.Parse(permute(fs, args))
}

func permute(fs *flag.FlagSet, args []string) []string {
	var flags, positional []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if len(a) < 2 || a[0] != '-' {
			positional = append(positional, a)
			continue
		}
		flags = append(flags, a)
		name := strings.TrimLeft(a, "-")
		if strings.ContainsRune(name, '=') {
			continue
		}
		if f := fs.Lookup(name); f != nil && !isBoolFlag(f) && i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	return append(flags, positional...)
}

func isBoolFlag(f *flag.Flag) bool {
	b, ok := f.Value.(interface{ IsBoolFlag() bool })
	return ok && b.IsBoolFlag()
}
