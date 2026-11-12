package parse

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"

	ts "github.com/odvcencio/gotreesitter"
)

// A guarded macro mask for C, C++ and Objective-C.
//
// WHY. Tree-sitter parses the text as written and nothing runs the
// preprocessor. A declaration written behind a macro the grammar has never
// heard of is therefore not a declaration to it:
//
//	class LEVELDB_EXPORT Comparator { ... };   // the class is named LEVELDB_EXPORT
//	JK_STATIC_INLINE void jk_foo(int a) { }    // an error, and the function is lost
//	static REDIS_NO_SANITIZE("bounds") int f(void) { }
//
// Every public class of google/leveldb is written the first way, so none of
// them were symbols and every subclass bound to an unrelated type of the same
// name.
//
// WHAT. The source is scanned as text (comments, strings and preprocessor
// lines skipped) for the three places such a macro is written, and each
// candidate is overwritten with spaces of the SAME BYTE LENGTH, newlines kept,
// so no byte offset and no line number moves. The masked text is parsed and,
// if it is measurably better, replaces the plain parse.
//
// THE GUARD is what makes a wrong candidate harmless, and it must not be
// weakened. An upper-case identifier is often a type (`FILE *open(void)`,
// `BOOL WINAPI Foo(void)`, `DWORD x;`), an enum value or a macro call
// statement, and masking one of those would delete real text. So:
//
//   - the plain source is parsed first and a file whose plain parse shows no
//     trouble where a candidate sits costs nothing more and is returned as it
//     is;
//   - only then is the masked text parsed, once, and it is kept only if it has
//     fewer error nodes, no fewer definitions, no definition named after a
//     masked token, and every definition the plain parse had (other than one
//     named after a masked token) is still there;
//   - otherwise the plain result stands, byte for byte.
//
// A file is never parsed more than twice by this mechanism.

// maskedLanguages are the grammars the mask applies to.
var maskedLanguages = map[string]bool{"c": true, "cpp": true, "objc": true}

// macroCandidate is a span of the source that may be a macro the grammar does
// not know. name is the macro's identifier; the span also covers a
// parenthesised argument list when there is one.
type macroCandidate struct {
	start, end int
	name       string

	// expect is the byte offset of the name the declaration behind the macro
	// should define, when the text says it defines one (a class with a body, a
	// function with one), and -1 when it does not say. A plain parse that has
	// no definition named there is in trouble even if it reports no error:
	// `class EXPORT Foo { };` can parse as a function called Foo whose return
	// type is a class, without a single error node.
	expect int
}

type ctokKind uint8

const (
	ctIdent ctokKind = iota
	ctPunct
	ctScope // "::"
	ctOther // number, string or character literal
	ctDir   // a whole preprocessor directive
)

type ctok struct {
	kind       ctokKind
	start, end int
	c          byte // the character, for a ctPunct
}

func (t ctok) text(src []byte) string { return string(src[t.start:t.end]) }

// scanCTokens splits src into tokens, dropping comments and keeping each
// preprocessor directive as one token, so a `#define` body is never read as
// code.
func scanCTokens(src []byte) []ctok {
	toks := make([]ctok, 0, len(src)/5)
	n := len(src)
	lineStart := true
	for i := 0; i < n; {
		c := src[i]
		switch {
		case c == '\n':
			lineStart = true
			i++
		case c == ' ' || c == '\t' || c == '\r' || c == '\f' || c == '\v':
			i++
		case c == '/' && i+1 < n && src[i+1] == '/':
			for i < n && src[i] != '\n' {
				if src[i] == '\\' && i+1 < n {
					i++
				}
				i++
			}
		case c == '/' && i+1 < n && src[i+1] == '*':
			i += 2
			for i < n && !(src[i] == '*' && i+1 < n && src[i+1] == '/') {
				i++
			}
			i += 2
			if i > n {
				i = n
			}
		case c == '#' && lineStart:
			start := i
			for i < n && src[i] != '\n' {
				if src[i] == '\\' && i+1 < n {
					i++
					if src[i] == '\r' && i+1 < n {
						i++
					}
				} else if src[i] == '/' && i+1 < n && src[i+1] == '*' {
					i += 2
					for i < n && !(src[i] == '*' && i+1 < n && src[i+1] == '/') {
						i++
					}
					i++
				}
				i++
			}
			if i > n {
				i = n
			}
			toks = append(toks, ctok{kind: ctDir, start: start, end: i})
		case c == '"' || c == '\'':
			start := i
			i = skipQuoted(src, i)
			toks = append(toks, ctok{kind: ctOther, start: start, end: i})
			lineStart = false
		case isIdentStart(c):
			start := i
			for i < n && isIdentByte(src[i]) {
				i++
			}
			// A string prefix (L"x", u8"x", R"(x)") is part of the literal.
			if i < n && (src[i] == '"' || src[i] == '\'') && i-start <= 2 {
				switch string(src[start:i]) {
				case "L", "u", "U", "u8":
					i = skipQuoted(src, i)
					toks = append(toks, ctok{kind: ctOther, start: start, end: i})
					lineStart = false
					continue
				case "R", "LR", "uR", "UR", "u8R":
					if src[i] == '"' {
						i = skipRawString(src, i)
						toks = append(toks, ctok{kind: ctOther, start: start, end: i})
						lineStart = false
						continue
					}
				}
			}
			toks = append(toks, ctok{kind: ctIdent, start: start, end: i})
			lineStart = false
		case c >= '0' && c <= '9':
			start := i
			for i < n && (isIdentByte(src[i]) || src[i] == '.' || src[i] == '\'') {
				i++
			}
			toks = append(toks, ctok{kind: ctOther, start: start, end: i})
			lineStart = false
		case c == ':' && i+1 < n && src[i+1] == ':':
			toks = append(toks, ctok{kind: ctScope, start: i, end: i + 2, c: ':'})
			i += 2
			lineStart = false
		default:
			toks = append(toks, ctok{kind: ctPunct, start: i, end: i + 1, c: c})
			i++
			lineStart = false
		}
	}
	return toks
}

func isIdentStart(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= 0x80
}

func isIdentByte(c byte) bool { return isIdentStart(c) || c >= '0' && c <= '9' }

// skipQuoted returns the index after the string or character literal that
// opens at i. An unterminated literal ends at the end of its line.
func skipQuoted(src []byte, i int) int {
	q := src[i]
	i++
	for i < len(src) && src[i] != '\n' {
		if src[i] == '\\' && i+1 < len(src) {
			i += 2
			continue
		}
		if src[i] == q {
			return i + 1
		}
		i++
	}
	return i
}

// skipRawString returns the index after the raw string literal whose opening
// quote is at i: R"delim( ... )delim".
func skipRawString(src []byte, i int) int {
	j := i + 1
	for j < len(src) && src[j] != '(' && src[j] != '\n' && j-i <= 17 {
		j++
	}
	if j >= len(src) || src[j] != '(' {
		return skipQuoted(src, i)
	}
	closer := ")" + string(src[i+1:j]) + "\""
	if k := strings.Index(string(src[j:]), closer); k >= 0 {
		return j + k + len(closer)
	}
	return len(src)
}

// isMacroName reports whether an identifier is written the way a macro is:
// upper case, digits and underscores, at least two characters, at least one
// letter. `FILE`, `BOOL` and `DWORD` match, which is why the guard exists.
func isMacroName(s string) bool {
	if len(s) < 2 {
		return false
	}
	letters := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'A' && c <= 'Z':
			letters++
		case c == '_' || c >= '0' && c <= '9':
		default:
			return false
		}
	}
	return letters > 0 && s[0] != '0' && !(s[0] >= '0' && s[0] <= '9')
}

// cStorage are the words that may precede an attribute macro in a declaration.
var cStorage = map[string]bool{
	"static": true, "inline": true, "extern": true, "virtual": true, "constexpr": true,
	"friend": true, "const": true, "volatile": true, "explicit": true, "_Noreturn": true,
}

// notDeclaration are words that start a statement, not a declaration.
var notDeclaration = map[string]bool{
	"return": true, "else": true, "goto": true, "case": true, "default": true, "new": true,
	"delete": true, "throw": true, "sizeof": true, "do": true, "while": true, "if": true,
	"for": true, "switch": true, "using": true, "typedef": true, "break": true, "continue": true,
	"final": true, "override": true, "alignas": true, "operator": true,
}

// findMacroCandidates returns the spans that look like an unknown macro in a
// declaration, in source order and without overlap. It reads text only.
func findMacroCandidates(src []byte, language string) []macroCandidate {
	if !maskedLanguages[language] {
		return nil
	}
	toks := scanCTokens(src)
	var out []macroCandidate
	add := func(c macroCandidate) {
		if len(out) > 0 && c.start < out[len(out)-1].end {
			return
		}
		out = append(out, c)
	}
	s := &candScan{src: src, toks: toks, cpp: language != "c"}
	for i := range toks {
		t := toks[i]
		if t.kind != ctIdent {
			continue
		}
		switch w := t.text(src); w {
		case "class", "struct", "union", "enum":
			for _, c := range s.typeKeyword(i, w) {
				add(c)
			}
		}
		if s.atDeclStart(i) {
			for _, c := range s.declMacros(i) {
				add(c)
			}
		}
		for _, c := range s.trailingMacros(i) {
			add(c)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].start < out[b].start })
	kept := out[:0]
	for _, c := range out {
		if len(kept) > 0 && c.start < kept[len(kept)-1].end {
			continue
		}
		kept = append(kept, c)
	}
	return kept
}

type candScan struct {
	src  []byte
	toks []ctok
	cpp  bool
}

func (s *candScan) word(i int) string {
	if i < 0 || i >= len(s.toks) || s.toks[i].kind != ctIdent {
		return ""
	}
	return s.toks[i].text(s.src)
}

func (s *candScan) punct(i int, c byte) bool {
	return i >= 0 && i < len(s.toks) && s.toks[i].kind == ctPunct && s.toks[i].c == c
}

// skipParens returns the index after the parenthesised list that opens at i,
// or -1 when it does not close within a sane distance.
func (s *candScan) skipParens(i int) int {
	depth := 0
	for j := i; j < len(s.toks) && j < i+400; j++ {
		t := s.toks[j]
		if t.kind == ctDir {
			return -1
		}
		if t.kind != ctPunct {
			continue
		}
		switch t.c {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return j + 1
			}
		case ';', '{', '}':
			return -1
		}
	}
	return -1
}

// macroAt returns the macro written at token i, with its argument list when it
// has one, and the index of the token after it.
func (s *candScan) macroAt(i int) (macroCandidate, int, bool) {
	w := s.word(i)
	if !isMacroName(w) {
		return macroCandidate{expect: -1}, i, false
	}
	t := s.toks[i]
	c := macroCandidate{start: t.start, end: t.end, name: w, expect: -1}
	next := i + 1
	if s.punct(next, '(') {
		after := s.skipParens(next)
		if after < 0 {
			return macroCandidate{expect: -1}, i, false
		}
		c.end = s.toks[after-1].end
		next = after
	}
	return c, next, true
}

// typeKeyword handles `class MACRO Name`, `struct MACRO(args) Name`,
// `enum class MACRO Name` and `class Name MACRO {`.
func (s *candScan) typeKeyword(i int, kw string) []macroCandidate {
	j := i + 1
	if kw == "enum" {
		if w := s.word(j); w == "class" || w == "struct" {
			j++
		}
	}
	var found []macroCandidate
	for {
		c, next, ok := s.macroAt(j)
		if !ok || s.word(next) == "" {
			break
		}
		// MACRO followed by a word: the macro, and then the name or a second
		// macro.
		found = append(found, c)
		j = next
	}
	name := s.word(j)
	if name == "" || notDeclaration[name] || cStorage[name] {
		return nil
	}
	after := j + 1
	if len(found) == 0 {
		// `class Name MACRO {`: the macro after the name.
		c, next, ok := s.macroAt(after)
		if ok && !isMacroName(name) && (s.punct(next, '{') || s.punct(next, ':') && !s.punct(next+1, ':')) {
			return []macroCandidate{c}
		}
		return nil
	}
	expect := -1
	switch {
	case s.punct(after, '{'), s.punct(after, ':') && !s.punct(after+1, ':'), s.word(after) == "final":
		expect = s.toks[j].start
	case s.punct(after, ';') && s.cpp && (kw == "class" || kw == "struct"):
	default:
		return nil
	}
	for i := range found {
		found[i].expect = expect
	}
	return found
}

// atDeclStart reports whether token i can begin a declaration: it is the first
// token, or follows `;`, `{`, `}`, a preprocessor line or an access label.
func (s *candScan) atDeclStart(i int) bool {
	if i == 0 {
		return true
	}
	p := s.toks[i-1]
	switch p.kind {
	case ctDir:
		return true
	case ctPunct:
		switch p.c {
		case ';', '{', '}':
			return true
		case ':':
			// `public:` and `private:`, not the second half of `a ? b : c`.
			w := s.word(i - 2)
			return w == "public" || w == "private" || w == "protected"
		}
	}
	return false
}

// declMacros handles a macro in front of a function declaration or definition:
//
//	MACRO void f(        MACRO(type) f(        static MACRO("x") int f(
//
// and a macro standing alone on its line in front of a directive or an
// Objective-C declaration (`G_BEGIN_DECLS`, `NS_ASSUME_NONNULL_BEGIN`).
func (s *candScan) declMacros(i int) []macroCandidate {
	j := i
	for cStorage[s.word(j)] {
		j++
	}
	var found []macroCandidate
	hasParen := false
	lastFrom := j
	for {
		c, next, ok := s.macroAt(j)
		if !ok {
			break
		}
		if next-1 > j && s.toks[next-1].kind == ctPunct && s.toks[next-1].c == ')' {
			hasParen = true
		}
		// A macro CALL statement is not a candidate: a word must follow, or
		// the macro must stand alone on its line in front of a directive.
		if s.word(next) == "" && !s.aloneOnLine(j, next) {
			break
		}
		found = append(found, c)
		lastFrom = j
		j = next
	}
	if len(found) == 0 && s.word(j) == "" {
		return nil
	}
	if len(found) > 0 && s.word(j) == "" {
		// The run ended on a macro alone on its line, in front of a directive,
		// an Objective-C `@` declaration or a closing brace.
		if s.aloneOnLine(lastFrom, j) {
			return found[len(found)-1:]
		}
		return nil
	}
	// A lone macro on a line of its own in front of the next declaration
	// (`FMT_BEGIN_NAMESPACE`, `G_BEGIN_DECLS`): it takes no arguments, nothing
	// else is on its line, and the next token starts a later one.
	if len(found) == 1 && !hasParen && j < len(s.toks) && bytes.IndexByte(s.src[s.toks[j-1].end:s.toks[j].start], '\n') >= 0 &&
		(lastFrom == 0 || bytes.IndexByte(s.src[s.toks[lastFrom-1].end:s.toks[lastFrom].start], '\n') >= 0) {
		return found
	}
	if notDeclaration[s.word(j)] {
		return nil
	}
	// What follows must be a function declarator: words, pointers, references,
	// scopes and template arguments, then `(`.
	words := 0
	k := j
loop:
	for k < len(s.toks) {
		t := s.toks[k]
		switch {
		case t.kind == ctIdent:
			// A macro in the middle of the head, between the return type and
			// the name (`void *JEMALLOC_ATTR(malloc) je_malloc(`): one that is
			// followed by more of the head, never the name in front of `(`.
			if c, next, ok := s.macroAt(k); ok && (s.word(next) != "" || s.punct(next, '*') || s.punct(next, '&')) {
				found = append(found, c)
				if next-1 > k {
					hasParen = true
				}
				k = next
				continue
			}
			words++
			k++
		case t.kind == ctScope:
			k++
		case t.kind == ctPunct && (t.c == '*' || t.c == '&' || t.c == '~'):
			k++
		case t.kind == ctPunct && t.c == '<':
			depth := 0
			for ; k < len(s.toks) && k < j+200; k++ {
				tk := s.toks[k]
				if tk.kind == ctPunct && tk.c == '<' {
					depth++
				} else if tk.kind == ctPunct && tk.c == '>' {
					depth--
					if depth == 0 {
						k++
						break
					}
				} else if tk.kind == ctPunct && (tk.c == ';' || tk.c == '{' || tk.c == '}') || tk.kind == ctDir {
					return nil
				}
			}
		case t.kind == ctPunct && t.c == '(':
			break loop
		default:
			return nil
		}
	}
	if k >= len(s.toks) || !s.punct(k, '(') {
		return nil
	}
	// `MACRO name(`: one word is the return type's absence; that is a call or
	// a constructor-like macro use, unless the macro took arguments of its own
	// (`MACRO(type) name(`).
	if len(found) == 0 || words < 2 && !hasParen {
		return nil
	}
	if name := k - 1; s.toks[name].kind == ctIdent && s.definesBody(k) {
		for i := range found {
			found[i].expect = s.toks[name].start
		}
	}
	return found
}

// trailingKeywords may stand between a parameter list and an annotation macro.
var trailingKeywords = map[string]bool{
	"const": true, "noexcept": true, "override": true, "final": true, "volatile": true,
}

// controlWords are followed by a parenthesised condition, not a parameter list.
var controlWords = map[string]bool{
	"if": true, "for": true, "while": true, "switch": true, "catch": true, "return": true,
	"sizeof": true, "defined": true, "alignof": true, "decltype": true, "typeof": true,
}

// trailingMacros handles an annotation written after the parameter list:
//
//	void Compact() LOCKS_EXCLUDED(mutex_) {      int f(int a) NORETURN;
//
// Left alone the grammar reads `LOCKS_EXCLUDED(mutex_)` as the function's name.
// The list it follows must belong to a function: an identifier in front of its
// opening parenthesis that is not a control word and is not written as a macro.
func (s *candScan) trailingMacros(i int) []macroCandidate {
	if s.toks[i].kind != ctIdent || i == 0 {
		return nil
	}
	j := i - 1
	for trailingKeywords[s.word(j)] {
		j--
	}
	if !s.punct(j, ')') {
		return nil
	}
	open := j
	for depth := 0; open >= 0 && open > j-400; open-- {
		if s.toks[open].kind == ctDir {
			return nil
		}
		if s.punct(open, ')') {
			depth++
		} else if s.punct(open, '(') {
			depth--
			if depth == 0 {
				break
			}
		}
	}
	if open <= 0 || !s.punct(open, '(') {
		return nil
	}
	name := s.word(open - 1)
	if name == "" || controlWords[name] || notDeclaration[name] || isMacroName(name) {
		return nil
	}
	var found []macroCandidate
	k := i
	for {
		c, next, ok := s.macroAt(k)
		if !ok {
			break
		}
		found = append(found, c)
		k = next
	}
	if len(found) == 0 {
		return nil
	}
	switch {
	case s.punct(k, '{'):
		for n := range found {
			found[n].expect = s.toks[open-1].start
		}
	case s.punct(k, ';'):
	default:
		return nil
	}
	return found
}

// definesBody reports whether the parameter list that opens at token i is
// followed by a function body rather than `;`.
func (s *candScan) definesBody(i int) bool {
	after := s.skipParens(i)
	if after < 0 {
		return false
	}
	for j := after; j < len(s.toks) && j < after+40; j++ {
		t := s.toks[j]
		if t.kind == ctPunct {
			switch t.c {
			case '{':
				return true
			case ';', '=', ',', '}':
				return false
			}
		}
		if t.kind == ctDir {
			return false
		}
	}
	return false
}

// aloneOnLine reports whether the macro occupying tokens [from, to) has
// nothing in front of it and nothing behind it on its lines, and the token
// after it is a directive, `@` or `}`.
func (s *candScan) aloneOnLine(from, to int) bool {
	if to >= len(s.toks) || to <= from {
		return false
	}
	next := s.toks[to]
	if !(next.kind == ctDir || next.kind == ctPunct && (next.c == '@' || next.c == '}')) {
		return false
	}
	if bytes.IndexByte(s.src[s.toks[to-1].end:next.start], '\n') < 0 {
		return false
	}
	return from == 0 || bytes.IndexByte(s.src[s.toks[from-1].end:s.toks[from].start], '\n') >= 0
}

// maskCandidates returns a copy of src with each candidate overwritten by
// spaces. Newlines are kept and no byte is added or removed, so every offset
// and line number is unchanged.
func maskCandidates(src []byte, cands []macroCandidate) []byte {
	out := make([]byte, len(src))
	copy(out, src)
	for _, c := range cands {
		for i := c.start; i < c.end && i < len(out); i++ {
			if out[i] != '\n' && out[i] != '\r' {
				out[i] = ' '
			}
		}
	}
	return out
}

// maskedParse is the result of the guarded second parse.
type maskedParse struct {
	by    *genericParser
	tree  *ts.Tree
	stop  stopInfo
	src   []byte
	tags  []ts.Tag
	count int
}

// retryMasked is the guarded second parse. plain is the parse already done;
// it returns nil when the plain result stands, which is the answer for every
// file with no candidate and for every file whose plain parse is clean where
// the candidates are.
//
// It parses at most once, on the parser that produced the plain tree, under
// the same work limits and cancellation as any parse: the caller must look at
// ctx afterwards, since a cancelled parse returns nil here.
func (p *genericParser) retryMasked(ctx context.Context, tree *ts.Tree, stop stopInfo, src []byte, tags []ts.Tag) *maskedParse {
	if !maskedLanguages[p.entry.Name] || stop.early || ctx.Err() != nil {
		return nil
	}
	cands := findMacroCandidates(src, p.entry.Name)
	if len(cands) == 0 {
		return nil
	}
	names := make(map[string]bool, len(cands))
	for _, c := range cands {
		names[c.name] = true
	}
	if !troubleAtCandidates(tree.RootNode(), cands, tags, names) {
		return nil
	}

	masked := maskCandidates(src, cands)
	tree2, stop2, err := p.parseOnce(ctx, masked)
	if err != nil || tree2 == nil || stop2.early || ctx.Err() != nil {
		return nil
	}
	tags2, err := p.tagTree(tree2, masked)
	if err != nil {
		return nil
	}
	if !maskIsBetter(tree.RootNode(), tags, tree2.RootNode(), tags2, names) {
		return nil
	}
	return &maskedParse{by: p, tree: tree2, stop: stop2, src: masked, tags: tags2, count: len(cands)}
}

// troubleAtCandidates reports whether the plain parse shows trouble where a
// candidate sits: an error or missing node overlapping one, or a definition
// named after one.
func troubleAtCandidates(root *ts.Node, cands []macroCandidate, tags []ts.Tag, names map[string]bool) bool {
	for _, t := range tags {
		if len(t.Kind) > len("definition") && t.Kind[:len("definition")] == "definition" && names[t.Name] {
			return true
		}
	}
	defined := make(map[int]bool, len(tags))
	for _, t := range tags {
		if isDefinitionTag(t.Kind) {
			defined[int(t.NameRange.StartByte)] = true
		}
	}
	for _, c := range cands {
		if c.expect >= 0 && !defined[c.expect] {
			return true
		}
	}
	if root == nil || !root.HasError() {
		return false
	}
	found := false
	walkErrors(root, func(n *ts.Node) bool {
		a, b := int(n.StartByte()), int(n.EndByte())
		for _, c := range cands {
			if c.start < b && a < c.end || a == b && a >= c.start && a <= c.end {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// walkErrors visits every error and missing node, entering only subtrees the
// parser flagged. visit returns false to stop.
func walkErrors(root *ts.Node, visit func(*ts.Node) bool) {
	stack := []*ts.Node{root}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if n == nil {
			continue
		}
		if (n.IsError() || n.IsMissing()) && !visit(n) {
			return
		}
		if n.IsMissing() || !n.HasError() {
			continue
		}
		for i := n.ChildCount() - 1; i >= 0; i-- {
			if c := n.Child(i); c != nil && (c.HasError() || c.IsError() || c.IsMissing()) {
				stack = append(stack, c)
			}
		}
	}
}

// maskIsBetter is the guard. The masked parse replaces the plain one only when
// it is measurably better and no worse: no more error nodes (a missing token
// counts as one) and no fewer definitions, with at least one of the two
// strictly better; no definition named after a masked token; and every
// definition the plain parse had that was not itself named after a masked
// token still there.
func maskIsBetter(plainRoot *ts.Node, plain []ts.Tag, maskedRoot *ts.Node, masked []ts.Tag, names map[string]bool) bool {
	pe, pm, _ := countSyntaxErrors(plainRoot)
	me, mm, _ := countSyntaxErrors(maskedRoot)
	plainErrs, maskedErrs := pe+pm, me+mm
	if maskedErrs > plainErrs {
		return false
	}
	have := make(map[string]int, len(masked))
	nMasked := 0
	for _, t := range masked {
		if !isDefinitionTag(t.Kind) {
			continue
		}
		if names[t.Name] {
			return false
		}
		have[t.Name]++
		nMasked++
	}
	nPlain := 0
	for _, t := range plain {
		if !isDefinitionTag(t.Kind) {
			continue
		}
		if names[t.Name] {
			// A definition named after a masked token is the macro mistaken
			// for a name; it is not one the mask can lose.
			continue
		}
		nPlain++
		if have[t.Name] == 0 {
			return false
		}
	}
	if nMasked < nPlain {
		return false
	}
	return maskedErrs < plainErrs || nMasked > nPlain
}

func isDefinitionTag(kind string) bool { return strings.HasPrefix(kind, "definition") }

// maskNote is the sentence appended to a parse error when macros were masked,
// so a file read through a mask says so.
func (r *Result) maskNote() string {
	if r.MacrosMasked == 0 {
		return ""
	}
	return fmt.Sprintf("; read with %s masked", plural(r.MacrosMasked, "macro token"))
}
