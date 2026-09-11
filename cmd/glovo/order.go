package main

import (
	"fmt"
	"strconv"

	"github.com/jgalea/glovo-cli/internal/glovo"
)

func cmdOrder(args []string) error {
	fs, c := newCommonFlags("order")
	var confirm bool
	fs.BoolVar(&confirm, "confirm", false, "actually place the order (charges your saved payment method)")
	if err := parseArgs(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: glovo order <store-id> [--confirm]")
	}
	sid, err := strconv.ParseInt(fs.Arg(0), 10, 64)
	if err != nil {
		return err
	}
	cl := glovo.NewClient(stderrLogf)

	b, err := cl.OrderDryRun(sid)
	if err != nil {
		return err
	}
	if len(b.Lines) == 0 {
		return fmt.Errorf("basket is empty for store %d — add items first", sid)
	}

	if !confirm {
		text := "DRY RUN — nothing ordered. Add --confirm to place it.\n" + basketText(b)
		return emit(c, b, text)
	}

	orderID, err := cl.PlaceOrder(sid)
	if err != nil {
		return err
	}
	fmt.Printf("Order placed: %s\n", orderID)
	return nil
}
