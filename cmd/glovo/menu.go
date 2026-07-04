package main

import (
	"fmt"
	"strings"

	"github.com/jgalea/glovo-cli/internal/glovo"
)

func cmdMenu(args []string) error {
	fs, c := newCommonFlags("menu")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: glovo menu <store-slug>")
	}
	slug := fs.Arg(0)
	cl := glovo.NewClient(stderrLogf)
	menu, err := cl.StoreMenu(defaultCity, slug)
	if err != nil {
		return err
	}
	var text strings.Builder
	fmt.Fprintf(&text, "store %d (address %d)\n", menu.StoreID, menu.StoreAddressID)
	for _, cat := range menu.Categories {
		text.WriteString(cat.Name + "\n")
		for _, it := range cat.Items {
			text.WriteString(itemLine(it) + "\n")
		}
	}
	return emit(c, menu, strings.TrimRight(text.String(), "\n"))
}
