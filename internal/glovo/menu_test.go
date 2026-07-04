package glovo

import (
	"os"
	"testing"
)

func TestParseMenu(t *testing.T) {
	html, err := os.ReadFile("testdata/menu_min.html")
	if err != nil {
		t.Fatal(err)
	}
	menu := parseMenu(string(html))
	if menu == nil {
		t.Fatal("parseMenu returned nil")
	}
	if menu.StoreID != 27121 {
		t.Fatalf("menu.StoreID = %d, want 27121", menu.StoreID)
	}
	if menu.StoreAddressID != 434279 {
		t.Fatalf("menu.StoreAddressID = %d, want 434279", menu.StoreAddressID)
	}
	cats := menu.Categories
	if len(cats) != 1 {
		t.Fatalf("got %d categories, want 1", len(cats))
	}
	cat := cats[0]
	if cat.Name != "Pizzas" || len(cat.Items) != 2 {
		t.Fatalf("cat = %+v", cat)
	}

	boscaiola := cat.Items[0]
	if boscaiola.Name != "Boscaiola" {
		t.Fatalf("item0 name = %q, want Boscaiola", boscaiola.Name)
	}
	if boscaiola.StoreProductID != 42257009854 {
		t.Fatalf("item0 StoreProductID = %d, want the numeric productSKU 42257009854 (not the storeProductId UUID)", boscaiola.StoreProductID)
	}
	if boscaiola.ExternalID != "p_1179695" {
		t.Fatalf("item0 ExternalID = %q", boscaiola.ExternalID)
	}
	if boscaiola.StoreProductUUID != "7db2260e-0883-4154-b940-e4ac8188d43c" {
		t.Fatalf("item0 StoreProductUUID = %q", boscaiola.StoreProductUUID)
	}
	if boscaiola.Price != 15.4 {
		t.Fatalf("item0 Price = %v, want 15.4 (from priceInfo.amount)", boscaiola.Price)
	}
	if boscaiola.Currency != "EUR" {
		t.Fatalf("item0 Currency = %q", boscaiola.Currency)
	}
	if boscaiola.PromoPrice != 10.78 {
		t.Fatalf("item0 PromoPrice = %v, want 10.78 (from promotions[0].priceInfo.amount)", boscaiola.PromoPrice)
	}
	if boscaiola.CategoryID != cat.ID {
		t.Fatalf("item0 CategoryID = %d, want %d", boscaiola.CategoryID, cat.ID)
	}

	margherita := cat.Items[1]
	if margherita.Name != "Margherita" || margherita.StoreProductID != 42257009855 {
		t.Fatalf("item1 = %+v", margherita)
	}
	if margherita.StoreProductUUID != "a1830fe7-6654-43f7-8fd1-b6ef7c1d44a7" {
		t.Fatalf("item1 StoreProductUUID = %q", margherita.StoreProductUUID)
	}
	if margherita.Price != 8.5 {
		t.Fatalf("item1 Price = %v, want 8.5", margherita.Price)
	}
	if margherita.PromoPrice != 0 {
		t.Fatalf("item1 PromoPrice = %v, want 0 (no promotions)", margherita.PromoPrice)
	}
}

func TestParseMenuUnparseable(t *testing.T) {
	if cats := parseMenu("<html><body>nothing here</body></html>"); cats != nil {
		t.Fatalf("got %v, want nil for unparseable page", cats)
	}
}
