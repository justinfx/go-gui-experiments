package core

import (
	"math"
	"strings"
	"unicode/utf8"
)

// Validation messages, exactly as the UI must show them.
const (
	MsgName           = "Name must be 3 to 80 characters."
	MsgPrice          = "Price must be between $0.00 and $100,000.00."
	MsgQuantity       = "Quantity must be between 0 and 1,000,000."
	MsgRating         = "Rating must be between 0.0 and 5.0."
	MsgRestock        = "Restock date is required for backordered items."
	MsgDiscontinued   = "Discontinued items must have quantity 0."
	MsgNotes          = "Notes must be at most 500 characters."
	MsgCategory       = "Choose a category."
	MsgStatus         = "Choose a status."
	MaxPriceCents     = 10_000_000
	MaxQuantity       = 1_000_000
	MaxNotesRunes     = 500
	MaxImportFileSize = 64 << 10
)

// ValidateItem checks every rule and returns all failures, or nil if valid.
// Errors are in field order: name, category, priceCents, quantity, status,
// rating, restockDate, notes. Category and status errors only occur for
// out-of-range enum values, which a UI cannot normally produce.
func ValidateItem(it Item) *ValidationError {
	var errs []FieldError
	add := func(f, m string) { errs = append(errs, FieldError{Field: f, Message: m}) }

	if n := utf8.RuneCountInString(strings.TrimSpace(it.Name)); n < 3 || n > 80 {
		add("name", MsgName)
	}
	if !it.Category.Valid() {
		add("category", MsgCategory)
	}
	if it.PriceCents < 0 || it.PriceCents > MaxPriceCents {
		add("priceCents", MsgPrice)
	}
	if it.Quantity < 0 || it.Quantity > MaxQuantity {
		add("quantity", MsgQuantity)
	}
	if it.Status == Discontinued && it.Quantity != 0 {
		add("quantity", MsgDiscontinued)
	}
	if !it.Status.Valid() {
		add("status", MsgStatus)
	}
	r := float64(it.Rating)
	if r < -1e-6 || r > 5+1e-6 || math.Abs(r*10-math.Round(r*10)) > 1e-3 {
		add("rating", MsgRating)
	}
	if it.Status == Backordered && it.RestockDate == nil {
		add("restockDate", MsgRestock)
	}
	if utf8.RuneCountInString(it.Notes) > MaxNotesRunes {
		add("notes", MsgNotes)
	}
	if len(errs) == 0 {
		return nil
	}
	return &ValidationError{Errors: errs}
}
