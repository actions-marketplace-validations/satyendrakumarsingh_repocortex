package parse

import "bytes"

// Groovy: the one-line doc comment.
//
// The grammar's groovy_doc rule reads the text after `/**` up to the end of the
// line as the comment's first line, whether or not a `*/` stands on it, and
// goes on reading lines until the next `*/`. A comment that opens and closes on
// one line (`/** the size */`) is therefore never closed: everything after it
// is comment text, up to the next `*/` or the end of the file. In grails-core's
// DefaultConstrainedProperty.groovy that is 49 of 50 definitions. See
// fixtures/langs/groovy/KNOWN_GAPS.md.
//
// The mask blanks the comment's second `*`, which makes it an ordinary block
// comment (`/* the size */`) that the grammar reads correctly. A doc comment
// carries no definition, so nothing is lost. `/**/` is the one exception: the
// grammar fails on it too, but it is an ordinary empty comment already, and
// blanking its second `*` would leave it unclosed, so it is blanked whole. Multi-line doc comments, which the grammar reads properly,
// are left alone.
//
// Not followed: a GString hole that holds a quote of its own kind
// ("${ "x" }"), which can put the scanner out of step with the string it is in;
// the guard refuses a mask that does not help.

// findGroovyOneLineDocs returns one span for each `/** ... */` that opens and
// closes on one line, outside strings and other comments.
func findGroovyOneLineDocs(src []byte) []maskSpan {
	var out []maskSpan
	prev := byte(0) // last significant byte before i, for slashy strings
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case c == '/' && i+1 < len(src) && src[i+1] == '/':
			i = gvLineEnd(src, i)
			continue
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			end := bytes.Index(src[i+2:], []byte("*/"))
			if end < 0 {
				return out
			}
			end += i + 4 // one past the closing */
			if i+2 < len(src) && src[i+2] == '*' && bytes.IndexByte(src[i:end], '\n') < 0 {
				if end-i == 4 { // /**/ is an ordinary comment already; unclosing it would swallow the file
					out = append(out, maskSpan{start: i, repl: []byte("    ")})
				} else {
					out = append(out, maskSpan{start: i + 2, repl: []byte(" ")})
				}
			}
			i = end
			continue
		case c == '"' || c == '\'':
			i = gvSkipString(src, i)
			prev = c
			continue
		case c == '$' && i+1 < len(src) && src[i+1] == '/':
			// $/ dollar-slashy /$
			if end := bytes.Index(src[i+2:], []byte("/$")); end >= 0 {
				i += 2 + end + 2
			} else {
				i = len(src)
			}
			prev = '"'
			continue
		case c == '/' && gvSlashyMayStart(prev):
			i = gvSkipSlashy(src, i)
			prev = '"'
			continue
		}
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			prev = c
		}
		i++
	}
	return out
}

func gvLineEnd(src []byte, i int) int {
	for i < len(src) && src[i] != '\n' {
		i++
	}
	return i
}

// gvSkipString returns the end of the string that opens at src[i], a ' or a ":
// single-line, or triple-quoted, with backslash escapes.
func gvSkipString(src []byte, i int) int {
	q := src[i]
	if i+2 < len(src) && src[i+1] == q && src[i+2] == q {
		for k := i + 3; k < len(src); k++ {
			if src[k] == '\\' {
				k++
			} else if src[k] == q && k+2 < len(src) && src[k+1] == q && src[k+2] == q {
				return k + 3
			}
		}
		return len(src)
	}
	for k := i + 1; k < len(src); k++ {
		switch src[k] {
		case '\\':
			k++
		case q:
			return k + 1
		case '\n':
			// A single-quoted string does not span lines: the scanner is out
			// of step, so resume at the line.
			return k
		}
	}
	return len(src)
}

// gvSlashyMayStart reports whether a `/` after the significant byte prev opens
// a slashy string rather than dividing: it does after an operator or an opening
// bracket, where a division would have no left operand.
func gvSlashyMayStart(prev byte) bool {
	switch prev {
	case '=', '(', ',', '[', ':', '~', '{', '!', '&', '|', '?', '+', '-', '*', '<', '>', ';':
		return true
	}
	return false
}

// gvSkipSlashy returns the end of the slashy string that opens at src[i].
func gvSkipSlashy(src []byte, i int) int {
	for k := i + 1; k < len(src); k++ {
		switch src[k] {
		case '\\':
			k++
		case '/':
			return k + 1
		}
	}
	return len(src)
}
