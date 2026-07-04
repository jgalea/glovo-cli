package main

import (
	"testing"

	"github.com/jgalea/glovo-cli/internal/glovo"
)

func testMenu() *glovo.Menu {
	return &glovo.Menu{
		StoreID:        27121,
		StoreAddressID: 434279,
		Categories: []glovo.Category{
			{
				ID:   1,
				Name: "Mains",
				Items: []glovo.MenuItem{
					{StoreProductID: 111, Name: "Burger", ExternalID: "p_111", StoreProductUUID: "uuid-111"},
					{StoreProductID: 222, Name: "Fries", ExternalID: "p_222", StoreProductUUID: "uuid-222"},
				},
			},
		},
	}
}

func TestFindMenuItemMatch(t *testing.T) {
	menu := testMenu()
	item, err := findMenuItem(menu, 222)
	if err != nil {
		t.Fatal(err)
	}
	if item.Name != "Fries" {
		t.Fatalf("Name = %q, want Fries", item.Name)
	}
	if item.StoreProductID != 222 {
		t.Fatalf("StoreProductID = %d, want 222", item.StoreProductID)
	}
}

func TestFindMenuItemNoMatch(t *testing.T) {
	menu := testMenu()
	if _, err := findMenuItem(menu, 999); err == nil {
		t.Fatal("expected an error for a product id not on the menu")
	}
}

func TestCmdCartSetRequiresQty(t *testing.T) {
	// qty is checked before any network call (the menu lookup happens
	// after), so this must fail fast with a clear usage error rather than
	// silently defaulting to 1 like "add" does.
	err := cmdCart([]string{"set", "some-slug", "123"})
	if err == nil {
		t.Fatal("expected an error when qty is omitted for cart set")
	}
}
