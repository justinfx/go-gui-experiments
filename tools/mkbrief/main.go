// Command mkbrief generates per-implementer briefs from the master SPEC.md
// and checks them for information leaks (see SPEC.md Appendix C).
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type app struct {
	id, name string
	terms    []string // terms that must not leak into other apps' briefs
}

var apps = []app{
	{"fyne", "Fyne", []string{"Fyne", "fyne"}},
	{"cogent", "Cogent Core", []string{"Cogent", "cogent", "cogentcore"}},
	{"wails", "Wails", []string{"Wails", "wails", "wails3", "Svelte", "svelte", "shadcn-svelte", "TanStack", "tanstack", "Vitest"}},
	{"miqt", "MIQT", []string{"MIQT", "miqt", "Qt", "QTableView", "QStyledItemDelegate"}},
	{"gotth", "GoTTH", []string{"GoTTH", "gotth", "templ", "htmx", "shadcn-templ", "templUI", "webview_go"}},
}

var forbidden = []string{"judge", "rubric", "score", "scoring", "overhead", "measured",
	"compare", "comparison", "experiment", "other implementations", "master"}

var (
	blockStart = regexp.MustCompile(`^\s*<!-- @audience ([a-z,]+) -->\s*$`)
	blockEnd   = regexp.MustCompile(`^\s*<!-- @end -->\s*$`)
	lineTag    = regexp.MustCompile(`\s*<!--@([a-z,]+)-->\s*$`)
	separator  = regexp.MustCompile(`^---\s*$`)
)

func main() {
	spec := flag.String("spec", "SPEC.md", "master spec path")
	out := flag.String("out", "briefs", "output directory")
	flag.Parse()

	lines, err := readLines(*spec)
	if err != nil {
		fatal(err)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fatal(err)
	}
	failed := false
	for _, a := range apps {
		brief, err := render(lines, a)
		if err != nil {
			fatal(fmt.Errorf("%s: %w", a.id, err))
		}
		problems := leaks(brief, a)
		for _, p := range problems {
			fmt.Fprintf(os.Stderr, "LEAK %s: %s\n", a.id, p)
		}
		failed = failed || len(problems) > 0
		path := filepath.Join(*out, a.id+".md")
		if err := os.WriteFile(path, []byte(brief), 0o644); err != nil {
			fatal(err)
		}
		fmt.Printf("%-8s %5d lines  %s\n", a.id, strings.Count(brief, "\n"), path)
	}
	if failed {
		fatal(fmt.Errorf("leak check failed"))
	}
	fmt.Println("leak check passed")
}

func render(lines []string, a app) (string, error) {
	valid := map[string]bool{"master": true, "brief": true}
	for _, x := range apps {
		valid[x.id] = true
	}
	allowed := func(aud string) (bool, error) {
		for _, part := range strings.Split(aud, ",") {
			if !valid[part] {
				return false, fmt.Errorf("unknown audience %q", part)
			}
		}
		switch aud {
		case "master":
			return false, nil
		case "brief":
			return true, nil
		}
		for _, part := range strings.Split(aud, ",") {
			if part == a.id {
				return true, nil
			}
		}
		return false, nil
	}

	var stack []bool
	var kept []string
	for i, line := range lines {
		if m := blockStart.FindStringSubmatch(line); m != nil {
			if len(stack) > 0 {
				return "", fmt.Errorf("line %d: nested audience block", i+1)
			}
			ok, err := allowed(m[1])
			if err != nil {
				return "", fmt.Errorf("line %d: %w", i+1, err)
			}
			stack = append(stack, ok)
			continue
		}
		if blockEnd.MatchString(line) {
			if len(stack) == 0 {
				return "", fmt.Errorf("line %d: @end without block", i+1)
			}
			stack = stack[:len(stack)-1]
			continue
		}
		if len(stack) > 0 && !stack[0] {
			continue
		}
		if m := lineTag.FindStringSubmatch(line); m != nil {
			ok, err := allowed(m[1])
			if err != nil {
				return "", fmt.Errorf("line %d: %w", i+1, err)
			}
			if !ok {
				continue
			}
			line = lineTag.ReplaceAllString(line, "")
		}
		line = strings.ReplaceAll(line, "{{ID}}", a.id)
		line = strings.ReplaceAll(line, "{{NAME}}", a.name)
		kept = append(kept, line)
	}
	if len(stack) > 0 {
		return "", fmt.Errorf("unterminated audience block")
	}
	return tidy(kept), nil
}

// tidy collapses runs of blank lines and repeated separators, and trims the ends.
func tidy(lines []string) string {
	var out []string
	lastSep := true // drop leading separators
	for _, l := range lines {
		blank := strings.TrimSpace(l) == ""
		if blank && (len(out) == 0 || strings.TrimSpace(out[len(out)-1]) == "") {
			continue
		}
		if separator.MatchString(l) {
			if lastSep {
				continue
			}
			lastSep = true
			out = append(out, l)
			continue
		}
		if !blank {
			lastSep = false
		}
		out = append(out, l)
	}
	for len(out) > 0 && (strings.TrimSpace(out[len(out)-1]) == "" || separator.MatchString(out[len(out)-1])) {
		out = out[:len(out)-1]
	}
	return strings.Join(out, "\n") + "\n"
}

func leaks(brief string, self app) []string {
	var problems []string
	for _, other := range apps {
		if other.id == self.id {
			continue
		}
		for _, t := range other.terms {
			re := regexp.MustCompile(`\b` + regexp.QuoteMeta(t) + `\b`)
			if loc := re.FindStringIndex(brief); loc != nil {
				problems = append(problems, fmt.Sprintf("term %q of %s at line %d", t, other.id, lineOf(brief, loc[0])))
			}
		}
	}
	for _, w := range forbidden {
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(w) + `\b`)
		if loc := re.FindStringIndex(brief); loc != nil {
			problems = append(problems, fmt.Sprintf("forbidden word %q at line %d", w, lineOf(brief, loc[0])))
		}
	}
	for _, t := range []string{"@audience", "<!--@", "{{"} {
		if i := strings.Index(brief, t); i >= 0 {
			problems = append(problems, fmt.Sprintf("leftover tag %q at line %d", t, lineOf(brief, i)))
		}
	}
	return problems
}

func lineOf(s string, off int) int { return strings.Count(s[:off], "\n") + 1 }

func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines, sc.Err()
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "mkbrief:", err)
	os.Exit(1)
}
