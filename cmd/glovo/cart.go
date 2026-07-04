package main

import (
	"fmt"
	"math"
	"strconv"

	"github.com/jgalea/glovo-cli/internal/glovo"
)

// storeCategoryDefault is the provisional storeCategoryId sent when a menu
// lookup can't confirm one (no reliable key was found — see Task 18).
const storeCategoryDefault = 1

func cmdCart(args []string) error {
	fs, c := newCommonFlags("cart")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) == 0 {
		return fmt.Errorf("usage: glovo cart get|add|set|clear <store-slug> [product-id] [qty]")
	}
	cl := glovo.NewClient(stderrLogf)
	// The authenticated basket endpoints require the delivery location's
	// glovo-* headers. Resolve coords the same way search does (flags/env/
	// default Barcelona); city/country default to BCN/ES.
	cl.SetLocation("BCN", "ES",
		resolveCoord(math.NaN(), "GLOVO_LAT", defaultSearchLat),
		resolveCoord(math.NaN(), "GLOVO_LNG", defaultSearchLng))
	sub := rest[0]

	switch sub {
	case "get":
		if len(rest) < 2 {
			return fmt.Errorf("usage: glovo cart get <store-slug>")
		}
		menu, err := cl.StoreMenu(defaultCity, rest[1])
		if err != nil {
			return err
		}
		b, err := cl.BasketGet(menu.StoreID)
		if err != nil {
			return err
		}
		fillLineNames(b, menu)
		return emit(c, b, basketText(b))
	case "add", "set":
		if len(rest) < 3 {
			return fmt.Errorf("usage: glovo cart %s <store-slug> <product-id> [qty]", sub)
		}
		pid, err := strconv.ParseInt(rest[2], 10, 64)
		if err != nil {
			return err
		}
		qty := 1
		if len(rest) >= 4 {
			q, err := strconv.Atoi(rest[3])
			if err != nil {
				return err
			}
			qty = q
		} else if sub == "set" {
			return fmt.Errorf("usage: glovo cart set <store-slug> <product-id> <qty>")
		}
		menu, err := cl.StoreMenu(defaultCity, rest[1])
		if err != nil {
			return err
		}
		item, err := findMenuItem(menu, pid)
		if err != nil {
			return err
		}
		p := glovo.CartProduct{ID: item.StoreProductID, ExternalID: item.ExternalID, StoreProductUUID: item.StoreProductUUID}
		var b *glovo.Basket
		if sub == "add" {
			b, err = cl.BasketAdd(menu.StoreID, menu.StoreAddressID, storeCategoryDefault, p, qty)
		} else {
			b, err = cl.BasketSet(menu.StoreID, menu.StoreAddressID, storeCategoryDefault, p, qty)
		}
		if err != nil {
			return err
		}
		fillLineNames(b, menu)
		return emit(c, b, basketText(b))
	case "clear":
		if len(rest) < 2 {
			return fmt.Errorf("usage: glovo cart clear <store-slug>")
		}
		menu, err := cl.StoreMenu(defaultCity, rest[1])
		if err != nil {
			return err
		}
		b, err := cl.BasketClear(menu.StoreID)
		if err != nil {
			return err
		}
		fillLineNames(b, menu)
		return emit(c, b, basketText(b))
	default:
		return fmt.Errorf("unknown cart subcommand %q", sub)
	}
}

// findMenuItem looks up a product by its numeric store-product id across all
// of the menu's categories.
func findMenuItem(menu *glovo.Menu, productID int64) (glovo.MenuItem, error) {
	for _, cat := range menu.Categories {
		for _, it := range cat.Items {
			if it.StoreProductID == productID {
				return it, nil
			}
		}
	}
	return glovo.MenuItem{}, fmt.Errorf("product %d not found on this store's menu — run: glovo menu <store-slug>", productID)
}

// fillLineNames joins basket lines back to the menu for display: the real
// basket response doesn't carry a product name, only ids/quantity/price.
func fillLineNames(b *glovo.Basket, menu *glovo.Menu) {
	if b == nil || menu == nil {
		return
	}
	names := make(map[int64]string)
	for _, cat := range menu.Categories {
		for _, it := range cat.Items {
			names[it.StoreProductID] = it.Name
		}
	}
	for i := range b.Lines {
		if name, ok := names[b.Lines[i].ProductID]; ok {
			b.Lines[i].Name = name
		}
	}
}
