package parse

import (
	"bytes"
	"strings"
	"testing"
)

// spanTexts returns the source text each span covers (a span's replacement is
// as long as the text it replaces).
func spanTexts(src string, spans []maskSpan) []string {
	var out []string
	for _, s := range spans {
		out = append(out, src[s.start:s.start+len(s.repl)])
	}
	return out
}

func TestFSharpConstraintCallForms(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want []string
	}{
		{"the minimal call", "let a = (^T: (member GetAwaiter: unit -> ^A) (task))\n",
			[]string{"(^T: (member GetAwaiter: unit -> ^A) (task))"}},
		{"spaces around the colon", "let a = (^T : (member GetAwaiter : unit -> ^A) (task))\n",
			[]string{"(^T : (member GetAwaiter : unit -> ^A) (task))"}},
		{"no spaces at all", "let a = (^T:(member GetAwaiter:unit-> ^A)(task))\n",
			[]string{"(^T:(member GetAwaiter:unit-> ^A)(task))"}},
		{"arguments without parentheses", "if (^A: (member get_IsCompleted: unit -> bool) awaiter) then 1 else 2\n",
			[]string{"(^A: (member get_IsCompleted: unit -> bool) awaiter)"}},
		{"static member", "let f x = (^a : (static member op_Implicit : ^b -> ^a) x)\n",
			[]string{"(^a : (static member op_Implicit : ^b -> ^a) x)"}},
		{"two type variables", "let f x = (^T or ^U: (static member Combine: ^T * ^U -> ^V) (x, y))\n",
			[]string{"(^T or ^U: (static member Combine: ^T * ^U -> ^V) (x, y))"}},
		{"three type variables", "let f x = (^T or ^U or ^V : (static member Combine: ^T -> ^V) x)\n",
			[]string{"(^T or ^U or ^V : (static member Combine: ^T -> ^V) x)"}},
		{"an operator member", "let f a b = (^T: (static member (+) : ^T * ^T -> ^T) (a, b))\n",
			[]string{"(^T: (static member (+) : ^T * ^T -> ^T) (a, b))"}},
		{"a signature with parentheses and generics", "let f b m g = (^M: (member Bind: 'd -> ('e -> 'c) -> 'c) (b, m, g))\n",
			[]string{"(^M: (member Bind: 'd -> ('e -> 'c) -> 'c) (b, m, g))"}},
		{"a byref signature", "let f d k = (^D: (member TryGetValue: 'key * byref< ^value > -> bool) (d, k))\n",
			[]string{"(^D: (member TryGetValue: 'key * byref< ^value > -> bool) (d, k))"}},
		{"arguments over two lines", "let p =\n    (^v: (static member TryParse: string * byref< ^v > -> bool) (text,\n                                                                 &output))\nlet q = 1\n",
			[]string{"(^v: (static member TryParse: string * byref< ^v > -> bool) (text,\n                                                                 &output))"}},
		{"the call alone on a line after a pipe", "x\n|> fun v ->\n    (^T: (member Run: unit -> int) v)\n",
			[]string{"(^T: (member Run: unit -> int) v)"}},
		{"wrapped in extra parentheses", "let imp arg = ((^a : (static member op_Implicit : ^b -> ^a) arg))\n",
			[]string{"(^a : (static member op_Implicit : ^b -> ^a) arg)"}},
		{"inside a negation", "if not (^T: (static member IsNaN: 'T -> bool) x) then 1 else 2\n",
			[]string{"(^T: (static member IsNaN: 'T -> bool) x)"}},
		{"in a lambda", "xs |> Seq.iter (fun (k, v) -> (^a: (member Add: string * string -> unit) (c, k, v)))\n",
			[]string{"(^a: (member Add: string * string -> unit) (c, k, v))"}},
		{"several in one file", "let a = (^T: (member A: unit -> int) x)\nlet b = (^T: (member B: unit -> int) x)\n",
			[]string{"(^T: (member A: unit -> int) x)", "(^T: (member B: unit -> int) x)"}},
		{"a call inside the arguments of another is one span", "let a = (^T: (member A: unit -> int) (x, (^U: (member B: unit -> int) y)))\n",
			[]string{"(^T: (member A: unit -> int) (x, (^U: (member B: unit -> int) y)))"}},
		{"a closing parenthesis in a string argument", "let a = (^T: (member A: string -> int) (\")\"))\nlet b = 1\n",
			[]string{"(^T: (member A: string -> int) (\")\"))"}},
		{"a parenthesis character literal argument", "let a = (^T: (member A: char -> int) ('('))\nlet b = 1\n",
			[]string{"(^T: (member A: char -> int) ('('))"}},
		{"a comment in the arguments", "let a = (^T: (member A: int -> int) (* ) *) (x))\nlet b = 1\n",
			[]string{"(^T: (member A: int -> int) (* ) *) (x))"}},
		{"after a type variable that is not a character", "let f (x: 'a) (y: 'b) = (^T: (member M: unit -> int) x)\n",
			[]string{"(^T: (member M: unit -> int) x)"}},
		{"after a character literal", "let c = '('\nlet a = (^T: (member M: unit -> int) x)\n",
			[]string{"(^T: (member M: unit -> int) x)"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := spanTexts(c.src, findFSharpConstraintCalls([]byte(c.src)))
			if strings.Join(got, "\x00") != strings.Join(c.want, "\x00") {
				t.Errorf("spans\n got %q\nwant %q", got, c.want)
			}
		})
	}
}

// Everything that looks like a call but is not one, or is not code.
func TestFSharpConstraintCallsInCommentsAndStringsAreLeftAlone(t *testing.T) {
	for name, src := range map[string]string{
		"a string":                 "let s = \"(^T: (member GetAwaiter: unit -> ^A) (task))\"\n",
		"a string with an escape":  "let s = \"a\\\" (^T: (member M: unit -> int) x)\"\n",
		"a verbatim string":        "let s = @\"(^T: (member M: unit -> int) x)\"\n",
		"a verbatim string, quote": "let s = @\"say \"\"hi\"\" (^T: (member M: unit -> int) x)\"\n",
		"a triple-quoted string":   "let s = \"\"\"\n(^T: (member M: unit -> int) x)\n\"\"\"\n",
		"an interpolated string":   "let s = $\"(^T: (member M: unit -> int) x)\"\n",
		"a line comment":           "// (^T: (member M: unit -> int) x)\nlet a = 1\n",
		"a doc comment":            "/// Calls (^T: (member M: unit -> int) x) on it.\nlet a = 1\n",
		"a block comment":          "(* (^T: (member M: unit -> int) x) *)\nlet a = 1\n",
		"a nested block comment":   "(* a (* b *) (^T: (member M: unit -> int) x) *)\nlet a = 1\n",
		"a type annotation":        "let f (x: ^T) = x\n",
		"a plain parenthesis":      "let f x = (x: int)\n",
		"a hat without a colon":    "let f x = (^T)\n",
		"a different construct":    "let f x = (^T: int) x\n",
		"a when constraint":        "let inline f (t: ^T) : ^A when ^T: (member GetAwaiter: unit -> ^A) = t.GetAwaiter()\n",
		"the multiplication op":    "let m = (*)\nlet n = 1\n",
		"no closing parenthesis":   "let a = (^T: (member M: unit -> int) x\n",
	} {
		if got := findFSharpConstraintCalls([]byte(src)); len(got) != 0 {
			t.Errorf("%s: %q", name, spanTexts(src, got))
		}
	}
	// A call after a comment or string that holds one is still found, once.
	src := "// (^T: (member A: unit -> int) x)\nlet s = \"(^T: (member B: unit -> int) x)\"\nlet a = (^T: (member C: unit -> int) x)\n"
	got := spanTexts(src, findFSharpConstraintCalls([]byte(src)))
	if len(got) != 1 || got[0] != "(^T: (member C: unit -> int) x)" {
		t.Errorf("got %q", got)
	}
}

func TestFSharpMaskKeepsEveryOffset(t *testing.T) {
	src := "namespace N\n\nlet inline p (text: string) =\n    let a = (^T: (member GetAwaiter: unit -> ^A) (task))\n    let b =\r\n        (^v: (static member TryParse: string * byref< ^v > -> bool) (text,\r\n                                                                     &output))\n    a\n"
	spans := findFSharpConstraintCalls([]byte(src))
	if len(spans) != 2 {
		t.Fatalf("spans = %d, want 2", len(spans))
	}
	out := applyMaskSpans([]byte(src), spans)
	if len(out) != len(src) {
		t.Fatalf("length %d, want %d", len(out), len(src))
	}
	for i := range out {
		if (src[i] == '\n' || src[i] == '\r') != (out[i] == '\n' || out[i] == '\r') || (src[i] == '\n' || src[i] == '\r') && src[i] != out[i] {
			t.Fatalf("line structure changed at byte %d", i)
		}
	}
	for _, s := range spans {
		if len(s.repl) < 2 || s.repl[0] != '(' || s.repl[1] != ')' || strings.Trim(string(s.repl[2:]), " \r\n") != "" {
			t.Errorf("filler %q is not `()` and spaces", s.repl)
		}
	}
	// Outside the spans nothing changed.
	in := make([]bool, len(src))
	for _, s := range spans {
		for i := s.start; i < s.start+len(s.repl); i++ {
			in[i] = true
		}
	}
	for i := range out {
		if !in[i] && out[i] != src[i] {
			t.Fatalf("byte %d changed outside the spans", i)
		}
	}
	if bytes.Contains(out, []byte("member")) {
		t.Error("the call survived the mask")
	}
}

// A mask that does not make the file better is not kept. The first stop is a
// different construct (a conditional type body) that lies before the call, so
// masking the call reads no further.
func TestFSharpMaskIsRefusedWhenItDoesNotHelp(t *testing.T) {
	src := "module M\n\ntype ExceptionNull =\n#if NET9_0_OR_GREATER && !FABLE_COMPILER\n    Exception | null\n#else\n    Exception\n#endif\n\nlet f task = (^T: (member GetAwaiter: unit -> ^A) (task))\n\nlet g x = x\n"
	p := fsharpParser(t)
	tree, stop, err := p.parseOnce(t.Context(), []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(findFSharpConstraintCalls([]byte(src))) != 1 {
		t.Fatal("expected a candidate")
	}
	if !stop.early && !tree.RootNode().HasError() {
		t.Fatal("the plain parse shows no trouble, so this input tests nothing")
	}
	tags, _, _ := p.tagTree(tree, []byte(src))
	res, err := p.Parse(t.Context(), "m.fsx", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if mp := p.retryMasked(t.Context(), tree, stop, []byte(src), tags); mp != nil {
		t.Errorf("the mask was kept (parsed to byte %d of %d)", mp.stop.parsedBytes, len(src))
	}
	if res.ConstructsMasked != 0 {
		t.Errorf("ConstructsMasked = %d for a refused mask", res.ConstructsMasked)
	}
}

// A clean file has nothing to mask and is parsed once: no retry, the result
// untouched and without a note.
func TestFSharpCleanFileIsParsedOnceAndUntouched(t *testing.T) {
	src := "module M\n\nlet a = 1\n\nlet f x = x + a\n"
	if got := findFSharpConstraintCalls([]byte(src)); got != nil {
		t.Fatalf("spans in a file with none: %v", got)
	}
	p := fsharpParser(t)
	tree, stop, err := p.parseOnce(t.Context(), []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	tags, _, _ := p.tagTree(tree, []byte(src))
	if mp := p.retryMasked(t.Context(), tree, stop, []byte(src), tags); mp != nil {
		t.Fatal("a clean file was retried")
	}
	res, err := p.Parse(t.Context(), "m.fsx", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if res.ConstructsMasked != 0 || res.Status != StatusOK || res.Err != nil {
		t.Fatalf("masked=%d status=%s err=%v", res.ConstructsMasked, res.Status, res.Err)
	}
}

// A file whose only "calls" are in comments and strings parses clean and is
// not retried.
func TestFSharpCallInACommentOrStringIsNotRetried(t *testing.T) {
	p := fsharpParser(t)
	// The text of a call in a comment or a string is not a candidate at all.
	for _, src := range []string{
		"module M\n\nlet a = 1\n// (^T: (member M: unit -> int) x)\n",
		"module M\n\nlet a = \"(^T: (member M: unit -> int) x)\"\nlet b = 2\n",
	} {
		tree, stop, err := p.parseOnce(t.Context(), []byte(src))
		if err != nil {
			t.Fatal(err)
		}
		tags, _, _ := p.tagTree(tree, []byte(src))
		if mp := p.retryMasked(t.Context(), tree, stop, []byte(src), tags); mp != nil {
			t.Errorf("retried a clean parse of %q", src)
		}
	}
}
