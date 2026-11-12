// Package parse turns source files into symbols and unresolved references.
//
// Two boundaries define this package:
//
//   - It never touches the database, the filesystem, or the network. It takes
//     bytes and returns facts. That makes it trivially testable and safe to run
//     across many goroutines.
//
//   - It does NOT resolve references. Tree-sitter produces a syntax tree per
//     file; it cannot tell you that `svc.updateStatus()` binds to
//     `com.pay.TransactionService#updateStatus`. Parsing emits Reference values
//     carrying the raw name and position, and the resolve layer binds them to
//     symbols with a confidence level. Conflating the two would let syntactic
//     guesses masquerade as resolved facts.
package parse

import (
	"context"

	"github.com/satyendrakumarsingh/repocortex/pkg/model"
)

// Status describes what the parser did with a file. It says nothing about
// what was extracted: a file that parses cleanly and declares nothing is
// StatusOK, and a file full of definitions the parser stumbled through is
// StatusPartial.
//
// The values are the ones stored in files.parse_status; see
// pkg/model/parsestatus.go for the contract and for how old rows read.
type Status string

const (
	// StatusOK means the parser read the whole file and reported no syntax
	// error.
	StatusOK Status = model.ParseStatusOK

	// StatusPartial means a tree was built but it is not a clean reading of
	// the whole file: the parser reported syntax errors, or the parse stopped
	// before the end because it exceeded its budget. Result.Reason says which,
	// and what was extracted is still usable.
	StatusPartial Status = model.ParseStatusPartial

	// StatusFailed means nothing usable came out. The file is recorded so the
	// gap is visible rather than silently absent.
	StatusFailed Status = model.ParseStatusFailed

	// StatusSkipped means no tree was attempted: the file is oversized or
	// empty, or its grammar has no definition query (Result.Reason is then
	// model.ParseReasonTextOnly and the file is indexed for search alone).
	StatusSkipped Status = model.ParseStatusSkipped
)

// Reference is a use of a name that has not yet been bound to a definition.
//
// It deliberately carries no target. Binding happens in the resolve layer,
// which attaches a confidence level; a Reference is raw evidence, not a fact
// about the graph.
type Reference struct {
	// Name is the identifier as written, e.g. "updateStatus".
	Name string

	// Kind mirrors tree-sitter tag kinds, e.g. "reference.call".
	Kind string

	// Receiver is the qualifying expression when the syntax exposes one, e.g.
	// "transactionService" in transactionService.updateStatus(). Empty when
	// unqualified. The resolver uses it to infer a target type.
	Receiver string

	// ArgCount is how many arguments this call site passes, where the parser
	// could determine that. model.ArityUnknown for every reference that is not
	// a call, for a call spreading a slice or a sequence, and for every
	// language with no argument query.
	//
	// Unknown is the default and it means exactly that. A18 is the reason the
	// type carries the distinction rather than using 0: an empty Receiver was
	// read as "the source wrote no receiver" when it meant "the query could not
	// express one", and every chained Go call broke. A count of zero and no
	// count are the same failure one field over.
	ArgCount model.Arity

	Range model.Range
}

// Import is a dependency declared by a file. The resolver uses these to narrow
// candidate targets, which is what separates a high-confidence binding from a
// bare name match.
type Import struct {
	// Path is the module/package as written, e.g. "com.pay.repository" or
	// "./notification-service".
	Path string

	// Alias is the local binding, when the language supports renaming.
	Alias string

	Range model.Range
}

// Result is everything one file yielded.
type Result struct {
	Language   string
	Status     Status
	Symbols    []model.Symbol
	References []Reference
	Imports    []Import

	// Reason is why Status is not StatusOK, as a stable token.
	// model.ParseReasonNone for a clean parse.
	Reason model.ParseReason

	// ErrorNodes and MissingTokens count what the parser reported when
	// Reason is model.ParseReasonSyntaxErrors: ERROR nodes (text it could not
	// fit into the grammar) and MISSING nodes (tokens it had to invent to
	// continue). FirstErrorLine is the 1-based line of the earliest of them.
	ErrorNodes     int
	MissingTokens  int
	FirstErrorLine int

	// ParsedBytes is how far into the source the parser got when Reason is
	// model.ParseReasonBudgetExceeded. Definitions past it are absent.
	ParsedBytes int

	// Err explains a status other than ok. Its text starts with Reason, so it
	// can be stored as files.parse_error unchanged. It is diagnostic text for
	// the file's record, never a reason to abort the indexing run.
	Err error
}

// TextIndexable reports whether the pipeline should put this file's text in
// the full-text index.
func (r *Result) TextIndexable() bool {
	if r == nil {
		return false
	}
	msg := ""
	if r.Err != nil {
		msg = r.Err.Error()
	}
	return model.TextIndexed(string(r.Status), msg)
}

// Parser extracts facts from one language's source.
//
// Implementations must be safe for concurrent use by multiple goroutines.
// The underlying tree-sitter parser is NOT concurrency-safe, so implementations
// are responsible for pooling or per-goroutine state.
type Parser interface {
	// Language is the canonical language name, matching the grammar registry.
	Language() string

	// Parse extracts facts from src. It must honour ctx cancellation, and it
	// must never panic: a malformed or hostile file has to degrade to
	// StatusFailed for that file alone. Spec §13 requires parser failures to
	// be isolated to the affected file.
	Parse(ctx context.Context, path string, src []byte) (*Result, error)
}

// Registry resolves a file to the parser that should handle it.
type Registry interface {
	// ForFile returns a parser for path, or nil when no grammar applies.
	ForFile(path string) Parser

	// Languages lists every language the registry can handle.
	Languages() []string
}
