package core

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
)

// ProcessStart is set when the core package initializes, which is at
// program start. Bench "ready" times are measured from it.
var ProcessStart = time.Now()

// MetricPrefix starts every metric line.
const MetricPrefix = "GGUIX_METRIC "

// FormatMetric returns one metric line without a newline:
// GGUIX_METRIC {"event":"...", ...fields} (fields in sorted key order).
func FormatMetric(event string, fields map[string]any) string {
	var b strings.Builder
	b.WriteString(MetricPrefix)
	ev, _ := json.Marshal(event)
	b.WriteString(`{"event":`)
	b.Write(ev)
	for _, k := range slices.Sorted(maps.Keys(fields)) {
		if k == "event" {
			continue
		}
		kb, _ := json.Marshal(k)
		vb, err := json.Marshal(fields[k])
		if err != nil {
			vb, _ = json.Marshal(fmt.Sprint(fields[k]))
		}
		b.WriteByte(',')
		b.Write(kb)
		b.WriteByte(':')
		b.Write(vb)
	}
	b.WriteByte('}')
	return b.String()
}

// EmitMetric writes one metric line to stdout. In js/wasm builds Go's stdout
// goes to the browser console (console.log).
func EmitMetric(event string, fields map[string]any) {
	fmt.Println(FormatMetric(event, fields))
}

// Ms converts a duration to milliseconds rounded to 0.1 ms, for metric fields.
func Ms(d time.Duration) float64 {
	return float64(d.Round(100*time.Microsecond)) / float64(time.Millisecond)
}
