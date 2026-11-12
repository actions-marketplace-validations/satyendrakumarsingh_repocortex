package parse

import "strings"

// supertypeQualifierLanguages are the languages whose inheritance query
// captures the last segment of a qualified supertype and that write the
// qualifier in front of it: `java.io.OutputStream`, `ns::Base`, `\Ns\Base`,
// `pkg.Base[T]`, `Mod::Base`, `path::Trait`.
var supertypeQualifierLanguages = map[string]bool{
	"scala": true, "java": true, "kotlin": true, "c_sharp": true, "cpp": true,
	"php": true, "python": true, "ruby": true, "typescript": true, "tsx": true,
	"rust": true,
}

// supertypeQualifier returns the qualifier a supertype is written with, read
// from the source text in front of the captured last segment (which starts at
// nameStart): `java.io` for `extends java.io.OutputStream`. The separators `.`,
// `::` and `\` are all returned as `.`. It is empty for an unqualified name.
//
// Text, not syntax, because every grammar spells a qualified type with its own
// node types and the captured name is the same last identifier in all of them;
// the characters before it are an identifier, a separator, an identifier, until
// something else. Generic arguments are never in front of the name.
//
// Path words that mean "this crate or package" and name no directory (`crate`,
// `self`, `super`, `global`) are dropped; if nothing is left the name is
// treated as unqualified, as before.
func supertypeQualifier(language string, src []byte, nameStart uint32) string {
	if !supertypeQualifierLanguages[language] || int(nameStart) > len(src) {
		return ""
	}
	i := int(nameStart)
	var segs []string
	for {
		// A separator directly in front of the segment just read.
		j := i
		switch {
		case j >= 2 && src[j-1] == ':' && src[j-2] == ':':
			j -= 2
		case j >= 1 && (src[j-1] == '.' || src[j-1] == '\\'):
			j--
		default:
			j = -1
		}
		if j < 0 {
			break
		}
		k := j
		for k > 0 && isQualifierIdent(src[k-1]) {
			k--
		}
		if k == j {
			break // a leading separator (`\Ns\Base`, `::Base`): nothing before it
		}
		segs = append(segs, string(src[k:j]))
		i = k
	}
	// segs holds the segments last to first.
	for l, r := 0, len(segs)-1; l < r; l, r = l+1, r-1 {
		segs[l], segs[r] = segs[r], segs[l]
	}
	for len(segs) > 0 {
		switch segs[0] {
		case "crate", "self", "super", "global":
			segs = segs[1:]
			continue
		}
		break
	}
	return strings.Join(segs, ".")
}

func isQualifierIdent(b byte) bool {
	return b == '_' || b == '$' || b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= 0x80
}
