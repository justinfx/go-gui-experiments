package core

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

// SortField names a sortable column.
type SortField string

const (
	SortID       SortField = "id"
	SortSKU      SortField = "sku"
	SortName     SortField = "name"
	SortCategory SortField = "category"
	SortPrice    SortField = "price"
	SortQuantity SortField = "quantity"
	SortStatus   SortField = "status"
	SortRating   SortField = "rating"
	SortUpdated  SortField = "updated"
	SortFeatured SortField = "featured"
)

// AllSortFields lists every sort field in table column order.
var AllSortFields = []SortField{SortID, SortSKU, SortName, SortCategory, SortPrice, SortQuantity, SortStatus, SortRating, SortUpdated, SortFeatured}

// Valid reports whether f is a known sort field.
func (f SortField) Valid() bool { return slices.Contains(AllSortFields, f) }

// ParseSortField parses a sort field name; "" means SortID.
func ParseSortField(s string) (SortField, error) {
	if s == "" {
		return SortID, nil
	}
	f := SortField(strings.ToLower(strings.TrimSpace(s)))
	if !f.Valid() {
		return "", fmt.Errorf("unknown sort field %q", s)
	}
	return f, nil
}

// Query selects, filters and orders items.
type Query struct {
	Search     string     `json:"search"`     // case-insensitive substring of Name or SKU; trimmed; "" = all
	Categories []Category `json:"categories"` // empty = all
	Status     *Status    `json:"status"`     // nil = all
	SortBy     SortField  `json:"sortBy"`     // "" means SortID
	Desc       bool       `json:"desc"`
}

// Normalize trims the search text, sorts and de-duplicates categories and
// defaults SortBy to SortID. Services normalize queries themselves; callers
// only need this to compare queries.
func (q Query) Normalize() Query {
	q.Search = strings.TrimSpace(q.Search)
	if len(q.Categories) > 0 {
		c := slices.Clone(q.Categories)
		slices.Sort(c)
		q.Categories = slices.Compact(c)
	} else {
		q.Categories = nil
	}
	if q.SortBy == "" {
		q.SortBy = SortID
	}
	return q
}

// Filtered reports whether the query has any filter (search, category or status).
func (q Query) Filtered() bool {
	return strings.TrimSpace(q.Search) != "" || len(q.Categories) > 0 || q.Status != nil
}

// Key returns a stable string identifying the normalized query.
func (q Query) Key() string {
	q = q.Normalize()
	var b strings.Builder
	b.WriteString("s=")
	b.WriteString(strconv.Quote(strings.ToLower(q.Search)))
	b.WriteString("|c=")
	for i, c := range q.Categories {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(c.String())
	}
	b.WriteString("|st=")
	if q.Status != nil {
		b.WriteString(q.Status.String())
	}
	b.WriteString("|o=")
	b.WriteString(string(q.SortBy))
	if q.Desc {
		b.WriteString("|desc")
	}
	return b.String()
}

// Matches reports whether it passes the query's filters (ordering is ignored).
func (q Query) Matches(it Item) bool {
	if len(q.Categories) > 0 && !slices.Contains(q.Categories, it.Category) {
		return false
	}
	if q.Status != nil && it.Status != *q.Status {
		return false
	}
	s := strings.ToLower(strings.TrimSpace(q.Search))
	if s == "" {
		return true
	}
	return strings.Contains(strings.ToLower(it.Name), s) || strings.Contains(strings.ToLower(it.SKU), s)
}

// Values encodes the query as URL parameters: search, cat (comma-separated
// category names), status, sort, desc (1/0).
func (q Query) Values() url.Values {
	q = q.Normalize()
	v := url.Values{}
	if q.Search != "" {
		v.Set("search", q.Search)
	}
	if len(q.Categories) > 0 {
		names := make([]string, len(q.Categories))
		for i, c := range q.Categories {
			names[i] = c.String()
		}
		v.Set("cat", strings.Join(names, ","))
	}
	if q.Status != nil {
		v.Set("status", q.Status.String())
	}
	v.Set("sort", string(q.SortBy))
	if q.Desc {
		v.Set("desc", "1")
	} else {
		v.Set("desc", "0")
	}
	return v
}

// QueryFromValues decodes parameters produced by Query.Values.
func QueryFromValues(v url.Values) (Query, error) {
	var q Query
	q.Search = v.Get("search")
	if cats := strings.TrimSpace(v.Get("cat")); cats != "" {
		for _, name := range strings.Split(cats, ",") {
			c, err := ParseCategory(name)
			if err != nil {
				return Query{}, err
			}
			q.Categories = append(q.Categories, c)
		}
	}
	if st := strings.TrimSpace(v.Get("status")); st != "" {
		s, err := ParseStatus(st)
		if err != nil {
			return Query{}, err
		}
		q.Status = &s
	}
	f, err := ParseSortField(v.Get("sort"))
	if err != nil {
		return Query{}, err
	}
	q.SortBy = f
	q.Desc = v.Get("desc") == "1" || v.Get("desc") == "true"
	return q.Normalize(), nil
}

// CompareItems orders a and b by field (descending if desc), with ties
// always broken by ID ascending. String fields compare by byte order of the
// display string. This is the reference ordering every DataService uses.
func CompareItems(a, b Item, field SortField, desc bool) int {
	var c int
	switch field {
	case SortName:
		c = strings.Compare(a.Name, b.Name)
	case SortCategory:
		c = strings.Compare(a.Category.String(), b.Category.String())
	case SortPrice:
		c = cmp.Compare(a.PriceCents, b.PriceCents)
	case SortQuantity:
		c = cmp.Compare(a.Quantity, b.Quantity)
	case SortStatus:
		c = strings.Compare(a.Status.String(), b.Status.String())
	case SortRating:
		c = cmp.Compare(ratingTenths(a.Rating), ratingTenths(b.Rating))
	case SortUpdated:
		c = a.UpdatedAt.Compare(b.UpdatedAt)
	case SortFeatured:
		c = cmp.Compare(boolInt(a.Featured), boolInt(b.Featured))
	default: // id, sku (SKU order equals ID order)
		c = 0
	}
	if desc {
		c = -c
	}
	if c != 0 {
		return c
	}
	if (field == SortID || field == SortSKU || field == "") && desc {
		return cmp.Compare(b.ID, a.ID)
	}
	return cmp.Compare(a.ID, b.ID)
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// MaxPageLimit is the largest PageRequest.Limit a service accepts.
const MaxPageLimit = 500

// PageRequest asks for Limit rows starting at Offset of the query result.
type PageRequest struct {
	Query  Query `json:"query"`
	Offset int   `json:"offset"` // 0-based row index into the filtered, sorted result
	Limit  int   `json:"limit"`  // 1..MaxPageLimit
}

// Page is one slice of a query result.
type Page struct {
	Offset   int    `json:"offset"`
	Items    []Item `json:"items"`    // len <= Limit; fewer only at the end
	Total    int    `json:"total"`    // number of rows matching the query
	QueryKey string `json:"queryKey"` // Query.Key() of the request
}

// FieldError is one validation failure. Field uses the JSON field name.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationError lists every validation failure of an item.
type ValidationError struct {
	Errors []FieldError `json:"errors"`
}

func (e *ValidationError) Error() string {
	msgs := make([]string, len(e.Errors))
	for i, fe := range e.Errors {
		msgs[i] = fe.Message
	}
	return strings.Join(msgs, " ")
}

// For returns the messages for one field joined by a space, or "".
func (e *ValidationError) For(field string) string {
	if e == nil {
		return ""
	}
	var msgs []string
	for _, fe := range e.Errors {
		if fe.Field == field {
			msgs = append(msgs, fe.Message)
		}
	}
	return strings.Join(msgs, " ")
}

// AsValidationError returns the *ValidationError wrapped in err, or nil.
func AsValidationError(err error) *ValidationError {
	var ve *ValidationError
	if errors.As(err, &ve) {
		return ve
	}
	return nil
}

// ErrNotFound is returned for an unknown item ID.
var ErrNotFound = errors.New("item not found")

// DataService is the item backend used by every UI.
type DataService interface {
	Page(ctx context.Context, req PageRequest) (Page, error)
	Get(ctx context.Context, id int64) (Item, error)
	// Update validates, sets UpdatedAt, stores the edit (in memory, process
	// lifetime) and returns the stored item. Returns *ValidationError if invalid.
	Update(ctx context.Context, it Item) (Item, error)
	// Export streams CSV (header + one row per matching item in query order,
	// then a trailing "# checksum <hex>" line) to w. progress is called at most
	// every 100 ms and once at the end, from the exporting goroutine. Returns
	// rows written. Honors ctx cancellation within 50 ms.
	Export(ctx context.Context, q Query, w io.Writer, progress func(done, total int)) (int, error)
}

// FindRow returns the 0-based row index of item id within the first maxRows
// rows of q's result, fetching pages of 200 through svc. ok is false if absent.
func FindRow(ctx context.Context, svc DataService, q Query, id int64, maxRows int) (row int, ok bool, err error) {
	const step = 200
	for off := 0; off < maxRows; off += step {
		p, err := svc.Page(ctx, PageRequest{Query: q, Offset: off, Limit: min(step, maxRows-off)})
		if err != nil {
			return 0, false, err
		}
		for i, it := range p.Items {
			if it.ID == id {
				return off + i, true, nil
			}
		}
		if len(p.Items) == 0 || off+len(p.Items) >= p.Total {
			break
		}
	}
	return 0, false, nil
}
