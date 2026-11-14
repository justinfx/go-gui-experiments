package main

// Bench support (category: bench). The bench script itself is TypeScript
// under frontend/src/bench; these are the Go methods it calls.

import (
	"encoding/json"
	"os"

	"gguix/core"
)

// EmitMetric forwards a metric line from the frontend to stdout through
// core.EmitMetric. fieldsJSON is the JSON object of the metric fields.
func (s *ItemService) EmitMetric(event string, fieldsJSON string) {
	var fields map[string]any
	if err := json.Unmarshal([]byte(fieldsJSON), &fields); err != nil {
		fields = map[string]any{"raw": fieldsJSON}
	}
	core.EmitMetric(event, fields)
}

// benchExportPath is where bench mode exports to (desktop, no dialog).
func (s *ItemService) benchExportPath() string {
	if !s.bench {
		return ""
	}
	return os.Getenv("GGUIX_BENCH_EXPORT")
}
