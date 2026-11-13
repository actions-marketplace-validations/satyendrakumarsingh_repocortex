package parse

// F#: the statically resolved member constraint call.
//
// `(^T: (member GetAwaiter: unit -> ^A) (task))` is how an inline function or
// a computation-expression builder calls a member on a type chosen at the call
// site. gotreesitter's F# parser loops on it without consuming a token; the
// library ends the loop with its no-progress guard and reports an iteration
// limit, and everything after the call is lost. In FsToolkit.ErrorHandling
// that is the second half of two builder files. See
// fixtures/langs/fsharp/KNOWN_GAPS.md and lang_fsharp_test.go.
//
// The mask overwrites the whole parenthesised expression, from the opening
// parenthesis before `^T` to its matching closing one (the arguments are
// inside it), with spaces, keeping newlines, and puts `()` at its start so that
// `let a = ` and `if not ... then` still have an operand. What the call
// returned is not a definition, so nothing is lost by reading it as unit.
//
// Forms (all seen in real code):
//
//	(^T: (member GetAwaiter: unit -> ^A) (task))
//	(^T: (member GetAwaiter: unit -> ^A) task)
//	(^a : (static member op_Implicit : ^b -> ^a) arg)
//	(^T or ^U : (static member (+) : ^T * ^U -> ^V) (a, b))
//	(^M: (member Bind: 'd -> ('e -> 'c) -> 'c) (builder,
//	                                           m, f))
//
// The constraint written after a signature (`when ^T : (member Name: int)`) is
// not an expression and is not touched.

// findFSharpConstraintCalls returns one span per member constraint call
// outside comments, strings and character literals. A call inside another
// one is covered by the outer span.
func findFSharpConstraintCalls(src []byte) []maskSpan {
	var out []maskSpan
	for i := 0; i < len(src); {
		if next, ok := fsSkipTrivia(src, i); ok {
			i = next
			continue
		}
		if src[i] == '(' {
			if end, ok := fsConstraintCallEnd(src, i); ok {
				out = append(out, maskSpan{start: i, repl: fsFiller(src[i:end])})
				i = end
				continue
			}
		}
		i++
	}
	return out
}

// fsFiller is the replacement for a constraint call: spaces, newlines kept,
// `()` first.
func fsFiller(text []byte) []byte {
	out := make([]byte, len(text))
	for i, c := range text {
		if c == '\n' || c == '\r' {
			out[i] = c
		} else {
			out[i] = ' '
		}
	}
	if len(out) >= 2 {
		out[0], out[1] = '(', ')'
	}
	return out
}

// fsConstraintCallEnd reports where the constraint call that opens at src[open]
// ends (one past its closing parenthesis), when src[open:] is one.
func fsConstraintCallEnd(src []byte, open int) (int, bool) {
	i := fsSkipSpace(src, open+1)
	// `^T`, then any number of `or ^U`.
	for {
		if i >= len(src) || src[i] != '^' {
			return 0, false
		}
		j := i + 1
		for j < len(src) && fsIdentByte(src[j]) {
			j++
		}
		if j == i+1 {
			return 0, false
		}
		i = fsSkipSpace(src, j)
		if i+2 < len(src) && src[i] == 'o' && src[i+1] == 'r' && (src[i+2] == ' ' || src[i+2] == '\t' || src[i+2] == '\n' || src[i+2] == '\r') {
			i = fsSkipSpace(src, i+2)
			continue
		}
		break
	}
	if i >= len(src) || src[i] != ':' {
		return 0, false
	}
	i = fsSkipSpace(src, i+1)
	if i >= len(src) || src[i] != '(' {
		return 0, false
	}
	i = fsSkipSpace(src, i+1)
	if fsWordAt(src, i, "static") {
		i = fsSkipSpace(src, i+len("static"))
	}
	if !fsWordAt(src, i, "member") {
		return 0, false
	}
	// The matching parenthesis of the outer one.
	depth := 0
	for k := open; k < len(src); {
		if next, ok := fsSkipTrivia(src, k); ok {
			k = next
			continue
		}
		switch src[k] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return k + 1, true
			}
		}
		k++
	}
	return 0, false
}

func fsIdentByte(c byte) bool {
	return c == '_' || c == '\'' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= 0x80
}

func fsSkipSpace(src []byte, i int) int {
	for i < len(src) && (src[i] == ' ' || src[i] == '\t' || src[i] == '\n' || src[i] == '\r') {
		i++
	}
	return i
}

// fsWordAt reports whether src[i:] starts with word as a whole word.
func fsWordAt(src []byte, i int, word string) bool {
	if i+len(word) > len(src) || string(src[i:i+len(word)]) != word {
		return false
	}
	return i+len(word) == len(src) || !fsIdentByte(src[i+len(word)])
}

// fsSkipTrivia returns the end of the comment, string or character literal
// that starts at src[i], when one does.
//
// The forms: `// ...` to the end of the line; `(* ... *)`, which nests (`(*)`
// is the multiplication operator, not a comment); "..." with backslash
// escapes; @"..." verbatim, where `""` is a quote; """...""" triple-quoted;
// $"...", $@"..." and $"""...""" interpolated, read as the string they begin
// with (a hole holding a quote is not followed); and 'x', '\n', '\” and
// friends. A single quote that does not close like that is a type variable
// (`'T`) and is not trivia.
func fsSkipTrivia(src []byte, i int) (int, bool) {
	c := src[i]
	switch {
	case c == '/' && i+1 < len(src) && src[i+1] == '/':
		for i < len(src) && src[i] != '\n' {
			i++
		}
		return i, true
	case c == '(' && i+1 < len(src) && src[i+1] == '*':
		if i+2 < len(src) && src[i+2] == ')' {
			return i + 3, true
		}
		depth := 0
		for k := i; k < len(src); {
			switch {
			case src[k] == '(' && k+1 < len(src) && src[k+1] == '*' && (k+2 >= len(src) || src[k+2] != ')'):
				depth++
				k += 2
			case src[k] == '*' && k+1 < len(src) && src[k+1] == ')':
				depth--
				k += 2
				if depth == 0 {
					return k, true
				}
			default:
				k++
			}
		}
		return len(src), true
	case c == '"':
		return fsSkipString(src, i, false), true
	case c == '@' && i+1 < len(src) && src[i+1] == '"':
		return fsSkipString(src, i+1, true), true
	case c == '$' && i+1 < len(src) && src[i+1] == '"':
		return fsSkipString(src, i+1, false), true
	case c == '$' && i+2 < len(src) && src[i+1] == '@' && src[i+2] == '"', c == '@' && i+2 < len(src) && src[i+1] == '$' && src[i+2] == '"':
		return fsSkipString(src, i+2, true), true
	case c == '\'':
		// 'x'
		if i+2 < len(src) && src[i+1] != '\\' && src[i+1] != '\n' && src[i+2] == '\'' {
			return i + 3, true
		}
		// '\n', '\'', '\\', 'A', '\065'
		if i+3 < len(src) && src[i+1] == '\\' {
			for k := i + 3; k < len(src) && k <= i+8 && src[k-1] != '\n'; k++ {
				if src[k] == '\'' {
					return k + 1, true
				}
			}
		}
	}
	return i, false
}

// fsSkipString returns the end of the string whose opening quote is src[q].
func fsSkipString(src []byte, q int, verbatim bool) int {
	if !verbatim && q+2 < len(src) && src[q+1] == '"' && src[q+2] == '"' {
		for k := q + 3; k+2 < len(src); k++ {
			if src[k] == '"' && src[k+1] == '"' && src[k+2] == '"' {
				return k + 3
			}
		}
		return len(src)
	}
	for k := q + 1; k < len(src); k++ {
		switch {
		case src[k] == '\\' && !verbatim:
			k++
		case src[k] == '"' && verbatim && k+1 < len(src) && src[k+1] == '"':
			k++
		case src[k] == '"':
			return k + 1
		}
	}
	return len(src)
}
