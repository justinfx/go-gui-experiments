// Command loc counts non-blank source lines of one app by language and by
// target category (shared, desktop, web, bench, test), following the
// marking rules in SPEC.md section 3.3, and reports the dual-target
// overhead. Output is JSON on stdout.
//
//	go run ./loc -app ../apps/fyne [-vendor 'frontend/src/lib/components/ui/**'] [-custom file1,file2]
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var skipDirs = map[string]bool{"node_modules": true, "bin": true, "dist": true, ".git": true, ".task": true, "build": true, ".svelte-kit": true, "test-results": true, "playwright-report": true}

var langByExt = map[string]string{
	".go": "go", ".ts": "ts", ".svelte": "svelte", ".js": "js", ".mjs": "js", ".templ": "templ",
	".css": "css", ".html": "html", ".c": "c", ".h": "c", ".cpp": "cpp", ".yml": "yaml", ".yaml": "yaml",
}

var (
	markStart = regexp.MustCompile(`(?://|#|<!--|/\*)\s*gguix:target\s+(web|desktop)`)
	markEnd   = regexp.MustCompile(`(?://|#|<!--|/\*)\s*gguix:end`)
	buildTag  = regexp.MustCompile(`^//go:build `)
	generated = regexp.MustCompile(`(?m)^// Code generated .* DO NOT EDIT\.$`)
)

type counts map[string]int

type report struct {
	App             string            `json:"app"`
	Files           int               `json:"files"`
	ByCategory      counts            `json:"by_category"`
	ByLanguage      counts            `json:"by_language"`
	ByCatLang       map[string]counts `json:"by_category_language"`
	OverheadLines   int               `json:"dual_target_overhead_lines"`
	OverheadPct     float64           `json:"dual_target_overhead_pct"`
	TaggedFiles     int               `json:"build_tagged_files"`
	InlineBlocks    int               `json:"inline_marker_blocks"`
	Generated       counts            `json:"excluded_generated_lines"`
	Vendored        counts            `json:"excluded_vendored_lines"`
	CustomRendering int               `json:"custom_rendering_lines"`
	LongestFunc     string            `json:"longest_go_func"`
	LongestFuncLen  int               `json:"longest_go_func_lines"`
	Unmatched       []string          `json:"marker_errors,omitempty"`
}

func main() {
	app := flag.String("app", "", "app directory")
	vendor := flag.String("vendor", "", "comma-separated globs (relative, ** allowed) of vendored library sources to exclude")
	custom := flag.String("custom", "", "comma-separated relative paths of custom-rendering files")
	flag.Parse()
	if *app == "" {
		fmt.Fprintln(os.Stderr, "usage: loc -app <dir>")
		os.Exit(2)
	}
	r := report{App: filepath.Base(*app), ByCategory: counts{}, ByLanguage: counts{}, ByCatLang: map[string]counts{}, Generated: counts{}, Vendored: counts{}}
	vendorRes := globs(*vendor)
	customSet := map[string]bool{}
	for _, c := range strings.Split(*custom, ",") {
		if c = strings.TrimSpace(c); c != "" {
			customSet[filepath.ToSlash(c)] = true
		}
	}
	fset := token.NewFileSet()
	err := filepath.WalkDir(*app, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(*app, path)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			// The top-level build/ dir (Wails) is walked only for its Taskfiles,
			// which count as build scripts like a Makefile (audit B9).
			if rel == "build" {
				return nil
			}
			if path != *app && (skipDirs[d.Name()] || strings.HasPrefix(d.Name(), ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		lang := language(d.Name())
		if lang == "" {
			return nil
		}
		if strings.HasPrefix(rel, "build/") && lang != "make" {
			return nil // generated templates and assets under build/
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lines := nonBlank(string(data))
		switch {
		case isGenerated(d.Name(), rel, string(data)):
			r.Generated[lang] += lines
			return nil
		case matchAny(vendorRes, rel) || strings.HasSuffix(d.Name(), ".min.js"):
			r.Vendored[lang] += lines
			return nil
		}
		r.Files++
		if customSet[rel] {
			r.CustomRendering += lines
		}
		fileCat := category(rel, d.Name())
		if lang == "go" && hasBuildTag(string(data)) {
			r.TaggedFiles++
		}
		if lang == "go" && fileCat != "test" {
			if f, err := parser.ParseFile(fset, path, data, 0); err == nil {
				for _, decl := range f.Decls {
					if fd, ok := decl.(*ast.FuncDecl); ok && fd.Body != nil {
						n := fset.Position(fd.End()).Line - fset.Position(fd.Pos()).Line + 1
						if n > r.LongestFuncLen {
							r.LongestFuncLen, r.LongestFunc = n, rel+":"+fd.Name.Name
						}
					}
				}
			}
		}
		cur := ""
		sc := bufio.NewScanner(strings.NewReader(string(data)))
		sc.Buffer(make([]byte, 1<<20), 1<<24)
		ln := 0
		for sc.Scan() {
			ln++
			line := sc.Text()
			if m := markStart.FindStringSubmatch(line); m != nil {
				if cur != "" {
					r.Unmatched = append(r.Unmatched, fmt.Sprintf("%s:%d nested marker", rel, ln))
				}
				cur = m[1]
				r.InlineBlocks++
				continue
			}
			if markEnd.MatchString(line) {
				if cur == "" {
					r.Unmatched = append(r.Unmatched, fmt.Sprintf("%s:%d end without start", rel, ln))
				}
				cur = ""
				continue
			}
			if strings.TrimSpace(line) == "" {
				continue
			}
			cat := fileCat
			if cat == "shared" && cur != "" {
				cat = cur
			}
			r.ByCategory[cat]++
			r.ByLanguage[lang]++
			if r.ByCatLang[cat] == nil {
				r.ByCatLang[cat] = counts{}
			}
			r.ByCatLang[cat][lang]++
		}
		if cur != "" {
			r.Unmatched = append(r.Unmatched, rel+": unterminated marker")
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	r.OverheadLines = r.ByCategory["desktop"] + r.ByCategory["web"]
	if base := r.ByCategory["shared"] + r.OverheadLines; base > 0 {
		r.OverheadPct = float64(int(float64(r.OverheadLines)/float64(base)*1000+0.5)) / 10
	}
	sort.Strings(r.Unmatched)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(r)
}

func language(name string) string {
	if name == "Makefile" || strings.HasSuffix(name, ".mk") || name == "Taskfile.yml" {
		return "make"
	}
	if strings.HasSuffix(name, ".d.ts") {
		return ""
	}
	return langByExt[filepath.Ext(name)]
}

func isGenerated(name, rel, data string) bool {
	if strings.HasSuffix(name, "_templ.go") || generated.MatchString(data) {
		return true
	}
	return strings.Contains(rel, "frontend/bindings/") || strings.HasSuffix(name, "wasm_exec.js")
}

func category(rel, name string) string {
	parts := strings.Split(rel, "/")
	dirs := parts[:len(parts)-1]
	switch {
	case strings.HasSuffix(name, "_test.go") || strings.Contains(name, ".test.") || strings.Contains(name, ".spec.") || hasDir(dirs, "tests") || hasDir(dirs, "e2e"):
		return "test"
	case strings.HasSuffix(name, "_bench.go") || strings.Contains(name, ".bench.") || hasDir(dirs, "bench"):
		return "bench"
	case strings.HasSuffix(name, "_desktop.go") || hasDir(dirs, "desktop"):
		return "desktop"
	case strings.HasSuffix(name, "_web.go") || hasDir(dirs, "web"):
		return "web"
	}
	return "shared"
}

func hasDir(dirs []string, d string) bool {
	for _, x := range dirs {
		if x == d {
			return true
		}
	}
	return false
}

func hasBuildTag(data string) bool {
	for _, l := range strings.SplitN(data, "\n", 20) {
		if buildTag.MatchString(l) {
			return true
		}
	}
	return false
}

func nonBlank(s string) int {
	n := 0
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			n++
		}
	}
	return n
}

func globs(spec string) []*regexp.Regexp {
	var out []*regexp.Regexp
	for _, g := range strings.Split(spec, ",") {
		g = strings.TrimSpace(g)
		if g == "" {
			continue
		}
		re := regexp.QuoteMeta(g)
		re = strings.ReplaceAll(re, `\*\*`, `.*`)
		re = strings.ReplaceAll(re, `\*`, `[^/]*`)
		out = append(out, regexp.MustCompile("^"+re+"$"))
	}
	return out
}

func matchAny(res []*regexp.Regexp, rel string) bool {
	for _, re := range res {
		if re.MatchString(rel) {
			return true
		}
	}
	return false
}
