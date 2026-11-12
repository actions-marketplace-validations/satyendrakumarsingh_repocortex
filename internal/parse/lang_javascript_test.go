package parse

import (
	"context"
	"sort"
	"strings"
	"testing"
)

func parseScript(t *testing.T, language, file, src string) *Result {
	t.Helper()
	p, err := parserForLanguage(language)
	if err != nil {
		t.Fatal(err)
	}
	res, err := p.Parse(context.Background(), file, []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if res.Status == StatusFailed {
		t.Fatalf("parse failed: %v", res.Err)
	}
	return res
}

func scriptSymbols(res *Result) []string {
	var out []string
	for _, s := range res.Symbols {
		out = append(out, string(s.Kind)+" "+strings.TrimPrefix(s.FQN, "")+"|"+s.Name)
	}
	sort.Strings(out)
	return out
}

// TestCommonJSImportBindings pins the rows a top-level `require` produces.
func TestCommonJSImportBindings(t *testing.T) {
	src := `'use strict';
var parse = require('./parse');
const { a, b: c } = require('./lib');
var q = require('./q').y;
var lazy;
function f() { var inner = require('./inner'); }
`
	res := parseScript(t, "javascript", "x.js", src)
	got := map[string]bool{}
	for _, imp := range res.Imports {
		got[imp.Path+"|"+imp.Alias] = true
	}
	for _, want := range []string{
		"./parse|parse", "./parse|",
		"./lib|.a", "./lib|.c", "./lib|",
		"./q|.q", "./q|",
		"./inner|",
	} {
		if !got[want] {
			t.Errorf("missing import row %q; have %v", want, got)
		}
	}
	for _, unwanted := range []string{"./inner|inner", "./inner|.inner", "./lib|.b", "./lib|b"} {
		if got[unwanted] {
			t.Errorf("unexpected import row %q", unwanted)
		}
	}
}

// TestStemName pins the rule that names an anonymous module.exports.
func TestStemName(t *testing.T) {
	for file, want := range map[string]string{
		"lib/view.js":                 "view",
		"/abs/lib/merge-options.js":   "mergeOptions",
		"router/index.js":             "router",
		"index.js":                    "index",
		"a/b/foo.config.js":           "fooConfig",
		"lib/9lives.js":               "",
		"C:\\proj\\lib\\thing.min.js": "thingMin",
	} {
		if got := stemName(file); got != want {
			t.Errorf("stemName(%q) = %q, want %q", file, got, want)
		}
	}
}
