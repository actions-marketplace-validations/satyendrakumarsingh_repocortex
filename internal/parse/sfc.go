package parse

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/satyendrakumarsingh/repocortex/pkg/model"
)

// sfcParser extracts a single-file component's <script> block and hands it to
// the parser for the language that block is actually written in.
//
// A .vue file is not one language, it is three in a trenchcoat: markup, a
// script written in TypeScript or JavaScript, and CSS. The Vue grammar parses
// the SFC envelope and leaves the script body as opaque text, so a tags query
// over it finds no functions, no imports and no components — which is why 514
// .vue files on a real Nuxt frontend were recorded as
// `no tags query available for "vue"` and contributed nothing at all.
//
// Everything a developer asks RepoCortex about a component — what does it import,
// what does it call, where is this function defined — lives in that script.
// Reading it as the language it declares is the difference between indexing the
// frontend and skipping it.
//
// Line numbers are offset back to the .vue file, so every reported location is
// a place the reader can actually open. Getting that wrong would be worse than
// skipping the file: a confident location that lands on the wrong line costs
// more than an honest gap.
type sfcParser struct {
	language string
	// delegates are chosen by the block's lang attribute, defaulting to ts.
	registry *registry
}

// scriptBlock matches an SFC <script> element, capturing its attributes and
// body. Non-greedy so that a file with several script blocks yields each in
// turn rather than one span from the first open tag to the last close.
var scriptBlock = regexp.MustCompile(`(?is)<script([^>]*)>(.*?)</script>`)

// langAttr pulls `lang="ts"` out of a script tag's attributes.
var langAttr = regexp.MustCompile(`(?i)\blang\s*=\s*["']([a-z]+)["']`)

func (p *sfcParser) Language() string { return p.language }

func (p *sfcParser) Parse(ctx context.Context, path string, src []byte) (*Result, error) {
	res := &Result{Language: p.language, Status: StatusOK}

	if len(src) == 0 {
		skip(res, model.ParseReasonEmpty, "")
		return res, nil
	}
	// The size limit is on the FILE. Each script block would pass it on its
	// own, and the regular expression below would still scan every byte of a
	// component nobody wrote by hand.
	if limit := p.registry.cfg.sourceLimit(); len(src) > limit {
		skip(res, model.ParseReasonTooLarge, tooLargeDetail(int64(len(src)), int64(limit)))
		return res, nil
	}

	blocks := scriptBlock.FindAllSubmatchIndex(src, -1)
	if len(blocks) == 0 {
		// A component with no script is template and style only. That is a
		// complete parse of a file that declares nothing, not a failure.
		return res, nil
	}

	for _, b := range blocks {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		attrs := string(src[b[2]:b[3]])
		body := src[b[4]:b[5]]
		if len(strings.TrimSpace(string(body))) == 0 {
			continue
		}

		inner := p.registry.ForFile("sfc." + scriptExt(attrs))
		if inner == nil {
			continue
		}
		// Lines before the body start, so ranges can be moved back into the
		// .vue file's own coordinates.
		offset := strings.Count(string(src[:b[4]]), "\n")

		sub, err := inner.Parse(ctx, path, body)
		if err != nil {
			// Only cancellation comes back as an error; a block that would
			// not parse is reported in sub.
			return nil, err
		}
		if sub == nil {
			continue
		}
		adoptBlockStatus(res, sub, offset)
		mergeShifted(res, sub, offset)
	}
	return res, nil
}

// adoptBlockStatus folds one script block's parse verdict into the file's.
//
// The component's status is the worst any of its blocks reported: a block the
// parser stopped in outranks one with syntax errors, which outranks a clean
// one. A block that failed outright leaves the component partial, not failed,
// because the other blocks and the component itself are still in the index.
// The first-error line is moved into the file's coordinates.
func adoptBlockStatus(res, sub *Result, offset int) {
	rank := func(r *Result) int {
		switch {
		case r.Status == StatusFailed:
			return 3
		case r.Status == StatusPartial && r.Reason != model.ParseReasonSyntaxErrors:
			return 2
		case r.Status == StatusPartial:
			return 1
		}
		return 0
	}
	res.ErrorNodes += sub.ErrorNodes
	res.MissingTokens += sub.MissingTokens
	if sub.FirstErrorLine > 0 {
		if line := sub.FirstErrorLine + offset; res.FirstErrorLine == 0 || line < res.FirstErrorLine {
			res.FirstErrorLine = line
		}
	}
	if sub.Status == StatusSkipped || rank(sub) < rank(res) || rank(sub) == 0 {
		return
	}
	res.Status = StatusPartial
	switch {
	case sub.Status == StatusFailed:
		// A block the parser produced nothing for: the component's index
		// entry stops short, which is what stopped_early says.
		res.Reason = model.ParseReasonStoppedEarly
		detail := "a script block could not be parsed"
		if sub.Err != nil {
			detail += ": " + sub.Err.Error()
		}
		res.Err = errors.New(model.FormatParseError(res.Reason, detail))
	case sub.Reason == model.ParseReasonSyntaxErrors:
		res.Reason = sub.Reason
		res.Err = errors.New(model.FormatParseError(res.Reason, fmt.Sprintf(
			"%s and %s in the script, first at line %d; definitions inside the regions the parser could not read may be missing",
			plural(res.ErrorNodes, "error node"), plural(res.MissingTokens, "missing token"), res.FirstErrorLine)))
	default:
		res.Reason = sub.Reason
		res.Err = sub.Err
		res.ParsedBytes = sub.ParsedBytes
	}
}

// scriptExt maps a script block's lang attribute to a file extension the
// registry can resolve. TypeScript is the default because `<script setup>` in a
// TypeScript project carries no lang attribute, and parsing TS source as
// JavaScript loses every type declaration in it.
func scriptExt(attrs string) string {
	m := langAttr.FindStringSubmatch(attrs)
	if m == nil {
		return "ts"
	}
	switch strings.ToLower(m[1]) {
	case "js", "javascript":
		return "js"
	case "jsx":
		return "jsx"
	case "tsx":
		return "tsx"
	default:
		return "ts"
	}
}

// mergeShifted appends sub's facts to res, moving every line into the outer
// file's coordinates.
func mergeShifted(res, sub *Result, offset int) {
	for _, s := range sub.Symbols {
		s.Range.Start.Line += offset
		s.Range.End.Line += offset
		res.Symbols = append(res.Symbols, s)
	}
	for _, r := range sub.References {
		r.Range.Start.Line += offset
		r.Range.End.Line += offset
		res.References = append(res.References, r)
	}
	for _, i := range sub.Imports {
		i.Range.Start.Line += offset
		i.Range.End.Line += offset
		res.Imports = append(res.Imports, i)
	}
}

// interface assertion: sfcParser is a Parser.
var _ Parser = (*sfcParser)(nil)
