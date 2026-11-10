package core

import (
	"fmt"
	"strings"
	"time"
)

// Category is an item category. Values are ordered alphabetically by name,
// so numeric order equals display-string order.
type Category uint8

const (
	Electronics Category = iota
	Garden
	Kitchen
	Office
	Outdoor
	Sports
	Tools
	Toys
)

var categoryNames = [...]string{"Electronics", "Garden", "Kitchen", "Office", "Outdoor", "Sports", "Tools", "Toys"}

// AllCategories lists every category in display order.
var AllCategories = []Category{Electronics, Garden, Kitchen, Office, Outdoor, Sports, Tools, Toys}

func (c Category) String() string {
	if c.Valid() {
		return categoryNames[c]
	}
	return fmt.Sprintf("Category(%d)", uint8(c))
}

// Valid reports whether c is one of the defined categories.
func (c Category) Valid() bool { return int(c) < len(categoryNames) }

// ParseCategory parses a display name (case-insensitive).
func ParseCategory(s string) (Category, error) {
	s = strings.TrimSpace(s)
	for i, n := range categoryNames {
		if strings.EqualFold(n, s) {
			return Category(i), nil
		}
	}
	return 0, fmt.Errorf("unknown category %q", s)
}

func (c Category) MarshalText() ([]byte, error) {
	if !c.Valid() {
		return nil, fmt.Errorf("invalid category %d", uint8(c))
	}
	return []byte(c.String()), nil
}

func (c *Category) UnmarshalText(b []byte) error {
	v, err := ParseCategory(string(b))
	if err != nil {
		return err
	}
	*c = v
	return nil
}

// Status is an item's stock status. Numeric order equals display-string order.
type Status uint8

const (
	Active Status = iota
	Backordered
	Discontinued
)

var statusNames = [...]string{"Active", "Backordered", "Discontinued"}

// AllStatuses lists every status in display order.
var AllStatuses = []Status{Active, Backordered, Discontinued}

func (s Status) String() string {
	if s.Valid() {
		return statusNames[s]
	}
	return fmt.Sprintf("Status(%d)", uint8(s))
}

// Valid reports whether s is one of the defined statuses.
func (s Status) Valid() bool { return int(s) < len(statusNames) }

// ParseStatus parses a display name (case-insensitive).
func ParseStatus(s string) (Status, error) {
	s = strings.TrimSpace(s)
	for i, n := range statusNames {
		if strings.EqualFold(n, s) {
			return Status(i), nil
		}
	}
	return 0, fmt.Errorf("unknown status %q", s)
}

func (s Status) MarshalText() ([]byte, error) {
	if !s.Valid() {
		return nil, fmt.Errorf("invalid status %d", uint8(s))
	}
	return []byte(s.String()), nil
}

func (s *Status) UnmarshalText(b []byte) error {
	v, err := ParseStatus(string(b))
	if err != nil {
		return err
	}
	*s = v
	return nil
}

// StatusPtr returns a pointer to s, for building Query and Prefs values.
func StatusPtr(s Status) *Status { return &s }

// Date is a civil date without a time zone. JSON/text form is "YYYY-MM-DD".
type Date struct {
	Year  int
	Month time.Month
	Day   int
}

// String returns "YYYY-MM-DD".
func (d Date) String() string { return fmt.Sprintf("%04d-%02d-%02d", d.Year, int(d.Month), d.Day) }

// ParseDate parses "YYYY-MM-DD". The error text is meant for display:
// "Enter a date like 2026-03-07."
func ParseDate(s string) (Date, error) {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(s))
	if err != nil {
		return Date{}, fmt.Errorf("Enter a date like 2026-03-07.")
	}
	return DateOf(t), nil
}

// DateOf returns the civil date of t in t's location.
func DateOf(t time.Time) Date {
	y, m, d := t.Date()
	return Date{Year: y, Month: m, Day: d}
}

// Time returns midnight UTC of d.
func (d Date) Time() time.Time { return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC) }

func (d Date) MarshalText() ([]byte, error) { return []byte(d.String()), nil }

func (d *Date) UnmarshalText(b []byte) error {
	v, err := ParseDate(string(b))
	if err != nil {
		return err
	}
	*d = v
	return nil
}

// Item is one inventory record.
type Item struct {
	ID          int64     `json:"id"`  // 1..N, read-only
	SKU         string    `json:"sku"` // "SKU-0000001", read-only
	Name        string    `json:"name"`
	Category    Category  `json:"category"`
	PriceCents  int64     `json:"priceCents"`
	Quantity    int32     `json:"quantity"`
	Status      Status    `json:"status"`
	Rating      float32   `json:"rating"` // 0.0..5.0, one decimal place
	Featured    bool      `json:"featured"`
	RestockDate *Date     `json:"restockDate"` // optional
	UpdatedAt   time.Time `json:"updatedAt"`   // UTC, read-only, set by the service on Update
	Notes       string    `json:"notes"`
}

// SKUFor returns the SKU for an item ID: "SKU-" plus 7 zero-padded digits.
func SKUFor(id int64) string { return fmt.Sprintf("SKU-%07d", id) }
