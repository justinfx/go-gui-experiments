package main

import (
	"testing"
)

func TestPageBatchRangesInOrderAndLimits(t *testing.T) {
	s, _, _, _ := newTestService(t, -1)
	out, err := s.PageBatch(BatchArg{
		Query:  QueryArg{SortBy: "name"},
		Ranges: []RangeArg{{Offset: 400, Limit: 200}, {Offset: 0, Limit: 200}, {Offset: 800, Limit: 200}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 || out[0].Offset != 400 || out[1].Offset != 0 || out[2].Offset != 800 {
		t.Fatalf("ranges out of order: %+v", out)
	}
	if len(out[0].Items) != 200 || len(out[2].Items) != 200 || out[0].Total != 1000 {
		t.Fatal("page sizes or total")
	}
	single, _ := s.Page(PageArg{Query: QueryArg{SortBy: "name"}, Offset: 400, Limit: 200})
	if single.Items[0].ID != out[0].Items[0].ID {
		t.Fatal("batch row differs from single page")
	}
	for _, bad := range [][]RangeArg{nil, {{Offset: 0, Limit: 201}}, {{Offset: -1, Limit: 10}}, make([]RangeArg, maxBatchRanges+1)} {
		if _, err := s.PageBatch(BatchArg{Ranges: bad}); err == nil {
			t.Fatalf("expected an error for %v", bad)
		}
	}
}
