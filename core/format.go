package core

import (
	"encoding/csv"
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// FormatPrice formats cents as "$1,234.56".
func FormatPrice(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return sign + "$" + groupDigits(strconv.FormatInt(cents/100, 10)) + "." + pad2(cents%100)
}

// PriceDecimal formats cents as a plain decimal "1234.56" (used in CSV).
func PriceDecimal(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return sign + strconv.FormatInt(cents/100, 10) + "." + pad2(cents%100)
}

func pad2(n int64) string {
	if n < 10 {
		return "0" + strconv.FormatInt(n, 10)
	}
	return strconv.FormatInt(n, 10)
}

// ErrPriceFormat is returned by ParsePrice. Its text is meant for display.
var ErrPriceFormat = errors.New("Enter a price like 12.34.")

var priceRe = regexp.MustCompile(`^(\d{1,9})(?:\.(\d{1,2}))?$`)

// ParsePrice accepts "1234.56", "$1,234.56", "12" or "12.5" and returns cents.
// On failure it returns ErrPriceFormat ("Enter a price like 12.34.").
func ParsePrice(s string) (int64, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "$")
	s = strings.ReplaceAll(s, ",", "")
	m := priceRe.FindStringSubmatch(s)
	if m == nil {
		return 0, ErrPriceFormat
	}
	dollars, _ := strconv.ParseInt(m[1], 10, 64)
	var cents int64
	if m[2] != "" {
		c, _ := strconv.ParseInt(m[2], 10, 64)
		if len(m[2]) == 1 {
			c *= 10
		}
		cents = c
	}
	return dollars*100 + cents, nil
}

// FormatRating formats a rating with one decimal place, "4.3".
func FormatRating(r float32) string {
	return strconv.FormatFloat(float64(ratingTenths(r))/10, 'f', 1, 64)
}

func ratingTenths(r float32) int { return int(math.Round(float64(r) * 10)) }

// RatingFromTenths converts an integer number of tenths (for example a
// slider position 0..50) to a rating value.
func RatingFromTenths(t int) float32 { return float32(t) / 10 }

// RatingTenths returns the rating as an integer number of tenths (0..50).
func RatingTenths(r float32) int { return ratingTenths(r) }

// FormatUpdated formats a timestamp as "2026-01-02 15:04" in UTC.
func FormatUpdated(t time.Time) string { return t.UTC().Format("2006-01-02 15:04") }

// FormatCount formats an integer with thousands separators, "1,000,000".
func FormatCount(n int) string {
	if n < 0 {
		return "-" + groupDigits(strconv.Itoa(-n))
	}
	return groupDigits(strconv.Itoa(n))
}

func groupDigits(s string) string {
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	pre := len(s) % 3
	if pre > 0 {
		b.WriteString(s[:pre])
	}
	for i := pre; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

// CSVHeader lists the export columns.
var CSVHeader = []string{"id", "sku", "name", "category", "price", "quantity", "status", "rating", "featured", "restock_date", "updated_at", "notes"}

// CSVHeaderLine returns the header as one CSV line without a newline.
func CSVHeaderLine() string { return csvLine(CSVHeader) }

// ItemCSVRow returns one CSV line (no trailing newline) with the same
// columns as Export.
func ItemCSVRow(it Item) string {
	restock := ""
	if it.RestockDate != nil {
		restock = it.RestockDate.String()
	}
	return csvLine([]string{
		strconv.FormatInt(it.ID, 10),
		it.SKU,
		it.Name,
		it.Category.String(),
		PriceDecimal(it.PriceCents),
		strconv.FormatInt(int64(it.Quantity), 10),
		it.Status.String(),
		FormatRating(it.Rating),
		strconv.FormatBool(it.Featured),
		restock,
		it.UpdatedAt.UTC().Format(time.RFC3339),
		it.Notes,
	})
}

func csvLine(fields []string) string {
	var b strings.Builder
	w := csv.NewWriter(&b)
	_ = w.Write(fields)
	w.Flush()
	return strings.TrimRight(b.String(), "\r\n")
}

// StockCap is the quantity at which the stock bar is full.
const StockCap = 1000

// StockLowFraction is the fraction below which the stock bar uses the low color.
const StockLowFraction = 0.2

// StockFraction returns min(qty, 1000) / 1000, clamped at 0, for the stock bar.
func StockFraction(qty int32) float64 {
	if qty <= 0 {
		return 0
	}
	if qty >= StockCap {
		return 1
	}
	return float64(qty) / StockCap
}

// StockLow reports whether the stock bar should use the low-stock fill color.
func StockLow(qty int32) bool { return StockFraction(qty) < StockLowFraction }

// StockLabel returns "Stock 432".
func StockLabel(qty int32) string { return "Stock " + strconv.FormatInt(int64(qty), 10) }

// Stars rounds the rating to the nearest 0.5 and returns how many full, half
// and empty stars to draw; full+half+empty == 5.
func Stars(r float32) (full, half, empty int) {
	h := int(math.Round(float64(r) * 2))
	h = max(0, min(10, h))
	full, half = h/2, h%2
	return full, half, 5 - full - half
}

// CategoryInitial returns the first letter of the category name, "E".
func CategoryInitial(c Category) string {
	n := c.String()
	r, _ := utf8.DecodeRuneInString(n)
	return string(r)
}

// NotesPreview returns the first line of notes, cut to 90 characters plus
// "…" when longer, or "No notes" when empty.
func NotesPreview(notes string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(notes), "\n")
	line = strings.TrimSpace(line)
	if line == "" {
		return "No notes"
	}
	if utf8.RuneCountInString(line) > 90 {
		return string([]rune(line)[:90]) + "…"
	}
	return line
}
