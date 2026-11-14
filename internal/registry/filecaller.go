package registry

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/satyendrakumarsingh/repocortex/pkg/model"
)

// FileCallerKind is the kind of the synthetic symbol that stands for the code of
// a file that sits outside every extracted definition.
//
// WHY IT EXISTS. A call needs a caller, and an edge needs a symbol at each end.
// In a JavaScript or TypeScript test file nearly every call sits inside
// `describe(...)` and `it(...)` callbacks, which are not definitions; a Python
// module runs its top level on import; a Go package initialises its variables
// outside any function. Those calls were recorded as references and bound, and
// had no edge, so "who calls this" never listed them. The answer layer counted
// them as unlisted uses and could not say it was complete.
//
// WHAT IT IS. One row per file that has such code, created by `resolve` (never
// by `index`, so the parser's definitions and the conformance fixtures stay as
// they are), of kind `file`, named FileCallerName(path) so a caller list reads
// "(top level of src/foo.test.ts)", with the qualified name
// "<path>:(top level of <path>)". It is the SOURCE of edges only: nothing is ever bound to
// it, and no tool that lists, searches or counts definitions shows it (see
// registry.NotFileCaller). Anonymous callbacks nested inside a named function
// stay with that function: the symbol is only the fallback when no extracted
// definition encloses the line.
const FileCallerKind = string(model.KindFile)

// NotFileCaller is the SQL condition, for a symbols table aliased alias, that
// leaves the synthetic per-file callers out. Every query that LISTS, SEARCHES
// or COUNTS definitions carries it; a query that follows an edge to its source
// does not, since the caller must be nameable.
func NotFileCaller(alias string) string {
	if alias == "" {
		return "kind <> 'file'"
	}
	return alias + ".kind <> 'file'"
}

// FileCallerName names the synthetic caller of a file the way an answer reads
// it.
func FileCallerName(path string) string { return "(top level of " + path + ")" }

// FileCallerFQN is its qualified name: unique per file, and not the spelling of
// anything a reference can name.
func FileCallerFQN(path string) string { return path + ":" + FileCallerName(path) }

const insertFileCallerSQL = `
	INSERT OR IGNORE INTO symbols
		(index_run_id, file_id, kind, name, fqn, start_line, start_col, end_line, end_col,
		 signature, doc, exported, confidence, source)`

// EnsureFileCallers creates the synthetic caller of every file that still holds
// a reference to be resolved outside every extracted definition, and returns
// how many it created. It is idempotent: a file that has one keeps it.
//
// A definition spans start_line to end_line, and a definition whose span is
// unknown (end_line 0) covers only its own line, exactly as the resolver reads
// it (infer.enclosing), so the two agree on what "outside" means.
func (r *Registry) EnsureFileCallers(ctx context.Context, runID int64) (int, error) {
	res, err := r.db.ExecContext(ctx, insertFileCallerSQL+`
		SELECT f.index_run_id, f.id, 'file', '(top level of ' || f.path || ')', f.path || ':(top level of ' || f.path || ')',
		       1, 0, 1, 0, '', '', 0, 'high', 'source_code'
		  FROM files f
		 WHERE f.index_run_id = ?
		   AND EXISTS (
		       SELECT 1 FROM symbol_references sr
		        WHERE sr.file_id = f.id AND sr.resolution = 'pending' AND sr.start_line > 0
		          AND NOT EXISTS (
		              SELECT 1 FROM symbols s INDEXED BY idx_symbols_file
		               WHERE s.file_id = f.id AND s.kind <> 'file'
		                 AND s.start_line <= sr.start_line
		                 AND MAX(s.end_line, s.start_line) >= sr.start_line))`, runID)
	if err != nil {
		return 0, fmt.Errorf("registry: create file callers: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// FileCaller returns the synthetic caller of one file, creating it when the
// file has none.
func (r *Registry) FileCaller(ctx context.Context, runID, fileID int64, path string) (int64, error) {
	if _, err := r.db.ExecContext(ctx, insertFileCallerSQL+`
		VALUES (?, ?, 'file', ?, ?, 1, 0, 1, 0, '', '', 0, 'high', 'source_code')`,
		runID, fileID, FileCallerName(path), FileCallerFQN(path)); err != nil {
		return 0, fmt.Errorf("registry: create file caller of %s: %w", path, err)
	}
	var id int64
	err := r.db.QueryRowContext(ctx,
		`SELECT id FROM symbols WHERE index_run_id = ? AND file_id = ? AND kind = 'file' LIMIT 1`,
		runID, fileID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("registry: file caller of %s was not created", path)
	}
	if err != nil {
		return 0, fmt.Errorf("registry: read file caller of %s: %w", path, err)
	}
	return id, nil
}

// PruneFileCallers deletes the synthetic callers that no edge starts or ends
// at, and returns how many went. A file can have code outside its definitions
// whose references all stayed unbound; its caller would be an unused row.
func (r *Registry) PruneFileCallers(ctx context.Context, runID int64) (int, error) {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM symbols
		 WHERE index_run_id = ? AND kind = 'file'
		   AND NOT EXISTS (SELECT 1 FROM edges e WHERE e.src_id = symbols.id)
		   AND NOT EXISTS (SELECT 1 FROM edges e WHERE e.dst_id = symbols.id)`, runID)
	if err != nil {
		return 0, fmt.Errorf("registry: prune file callers: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// FileCallerIDs returns the synthetic caller of every file that has one, keyed
// by the file's repository path.
func (r *Registry) FileCallerIDs(ctx context.Context, runID int64) (map[string]int64, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT f.path, s.id FROM symbols s JOIN files f ON f.id = s.file_id
		 WHERE s.index_run_id = ? AND s.kind = 'file'`, runID)
	if err != nil {
		return nil, fmt.Errorf("registry: file callers: %w", err)
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var p string
		var id int64
		if err := rows.Scan(&p, &id); err != nil {
			return nil, err
		}
		out[p] = id
	}
	return out, rows.Err()
}
