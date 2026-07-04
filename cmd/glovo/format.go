package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jgalea/glovo-cli/internal/glovo"
)

func money(amount float64, currency string) string {
	s := strconv.FormatFloat(amount, 'f', 2, 64)
	if c := strings.ToUpper(currency); c != "" && c != "EUR" {
		return s + " " + currency
	}
	return s + "€"
}

func storeLine(s glovo.Store) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[%d] %s", s.ID, s.Name)
	if s.Rating > 0 {
		fmt.Fprintf(&b, "  %d%%", s.Rating)
	}
	if !s.Open {
		if s.ScheduledOpenAt != "" {
			fmt.Fprintf(&b, "  (closed until %s)", s.ScheduledOpenAt)
		} else {
			b.WriteString("  (closed)")
		}
	}
	return b.String()
}

func itemLine(m glovo.MenuItem) string {
	price := m.Price
	if m.PromoPrice > 0 && m.PromoPrice < m.Price {
		price = m.PromoPrice
	}
	return fmt.Sprintf("  [%d] %s — %s", m.StoreProductID, strings.TrimSpace(m.Name), money(price, m.Currency))
}

func basketText(b *glovo.Basket) string {
	if len(b.Lines) == 0 {
		return "(empty)"
	}
	var sb strings.Builder
	for _, l := range b.Lines {
		label := fmt.Sprintf("[%d]", l.ProductID)
		if l.Name != "" {
			label += " " + l.Name
		}
		fmt.Fprintf(&sb, "  %dx %s — %s\n", l.Qty, label, money(l.UnitPrice, b.Currency))
	}
	fmt.Fprintf(&sb, "Total: %s", money(b.Total, b.Currency))
	return sb.String()
}
