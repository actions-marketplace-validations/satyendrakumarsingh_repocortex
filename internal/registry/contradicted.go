package registry

import (
	"context"
	"fmt"
)

// contradictedSQL selects the heuristic call edges a compiler has proven
// false: in a file a compiler read, on a line where the compiler proved a call
// of the same name to a different definition, and proved as many calls of
// that name on the line as the parser recorded, so no call on the line is left
// for the heuristic edge to stand for; and with no compiler edge on the line
// to the heuristic's own target.
//
// Edges are unique per evidence line, so the line is the unit. Measured on
// honojs/hono: `c.header(...)` at middleware/cors/index.ts:161 had an exact
// edge to Context.header and a medium heuristic edge to HonoRequest.header,
// and find_callers listed both.
const contradictedSQL = `
	SELECT h.id
	  FROM edges h
	  JOIN symbols hd ON hd.id = h.dst_id
	  JOIN files hf ON hf.index_run_id = h.index_run_id AND hf.path = h.evidence_file
	  JOIN compiler_coverage cc ON cc.index_run_id = h.index_run_id AND cc.file_id = hf.id
	 WHERE h.index_run_id = ?
	   AND h.source <> 'compiler'
	   AND h.kind IN ('CALLS','INSTANTIATES')
	   AND NOT EXISTS (
	       SELECT 1 FROM edges x
	        WHERE x.index_run_id = h.index_run_id AND x.evidence_file = h.evidence_file
	          AND x.evidence_line = h.evidence_line AND x.source = 'compiler'
	          AND x.kind IN ('CALLS','INSTANTIATES') AND x.dst_id = h.dst_id)
	   AND (SELECT COUNT(*) FROM edges x JOIN symbols xd ON xd.id = x.dst_id
	         WHERE x.index_run_id = h.index_run_id AND x.evidence_file = h.evidence_file
	           AND x.evidence_line = h.evidence_line AND x.source = 'compiler'
	           AND x.kind IN ('CALLS','INSTANTIATES') AND xd.name = hd.name) >=
	       (SELECT COUNT(*) FROM symbol_references r
	         WHERE r.index_run_id = h.index_run_id AND r.file_id = hf.id
	           AND r.start_line = h.evidence_line AND r.name = hd.name)
	   AND EXISTS (SELECT 1 FROM edges x JOIN symbols xd ON xd.id = x.dst_id
	         WHERE x.index_run_id = h.index_run_id AND x.evidence_file = h.evidence_file
	           AND x.evidence_line = h.evidence_line AND x.source = 'compiler'
	           AND x.kind IN ('CALLS','INSTANTIATES') AND xd.name = hd.name)`

// WithdrawContradictedCalls removes the heuristic call edges a compiler
// proved false (see contradictedSQL) and re-binds each reference that pointed
// at a removed edge's target, on that line, to the one definition of the name
// the compiler proved there. It returns how many edges went and how many
// references were re-bound.
func (r *Registry) WithdrawContradictedCalls(ctx context.Context, runID int64) (withdrawn, rebound int, err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("registry: withdraw contradicted calls: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `CREATE TEMP TABLE IF NOT EXISTS contradicted_edge (id INTEGER PRIMARY KEY)`); err != nil {
		return 0, 0, fmt.Errorf("registry: withdraw contradicted calls: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM contradicted_edge`); err != nil {
		return 0, 0, fmt.Errorf("registry: withdraw contradicted calls: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO contradicted_edge (id) `+contradictedSQL, runID); err != nil {
		return 0, 0, fmt.Errorf("registry: withdraw contradicted calls: select: %w", err)
	}

	// Re-bind first, while the removed edges still say which line and which
	// wrong target: a reference on that line spelled like the target and bound
	// to it moves to the single definition of the name the compiler proved
	// there. Where the compiler proved more than one, it is left alone.
	res, err := tx.ExecContext(ctx, `
		UPDATE symbol_references SET resolved_symbol_id = (
		       SELECT MIN(x.dst_id) FROM edges h
		         JOIN contradicted_edge c ON c.id = h.id
		         JOIN files hf ON hf.index_run_id = h.index_run_id AND hf.path = h.evidence_file
		         JOIN symbols hd ON hd.id = h.dst_id
		         JOIN edges x ON x.index_run_id = h.index_run_id AND x.evidence_file = h.evidence_file
		              AND x.evidence_line = h.evidence_line AND x.source = 'compiler'
		              AND x.kind IN ('CALLS','INSTANTIATES')
		         JOIN symbols xd ON xd.id = x.dst_id AND xd.name = hd.name
		        WHERE hf.id = symbol_references.file_id
		          AND h.evidence_line = symbol_references.start_line
		          AND h.dst_id = symbol_references.resolved_symbol_id
		       HAVING COUNT(DISTINCT x.dst_id) = 1)
		 WHERE index_run_id = ? AND resolution = 'resolved'
		   AND EXISTS (
		       SELECT 1 FROM edges h
		         JOIN contradicted_edge c ON c.id = h.id
		         JOIN files hf ON hf.index_run_id = h.index_run_id AND hf.path = h.evidence_file
		        WHERE hf.id = symbol_references.file_id
		          AND h.evidence_line = symbol_references.start_line
		          AND h.dst_id = symbol_references.resolved_symbol_id
		          AND (SELECT COUNT(DISTINCT x.dst_id) FROM edges x JOIN symbols xd ON xd.id = x.dst_id
		                JOIN symbols hd ON hd.id = h.dst_id
		               WHERE x.index_run_id = h.index_run_id AND x.evidence_file = h.evidence_file
		                 AND x.evidence_line = h.evidence_line AND x.source = 'compiler'
		                 AND x.kind IN ('CALLS','INSTANTIATES') AND xd.name = hd.name) = 1)`, runID)
	if err != nil {
		return 0, 0, fmt.Errorf("registry: withdraw contradicted calls: re-bind: %w", err)
	}
	if n, e := res.RowsAffected(); e == nil {
		rebound = int(n)
	}
	res, err = tx.ExecContext(ctx, `DELETE FROM edges WHERE id IN (SELECT id FROM contradicted_edge)`)
	if err != nil {
		return 0, 0, fmt.Errorf("registry: withdraw contradicted calls: delete: %w", err)
	}
	if n, e := res.RowsAffected(); e == nil {
		withdrawn = int(n)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM contradicted_edge`); err != nil {
		return 0, 0, fmt.Errorf("registry: withdraw contradicted calls: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("registry: withdraw contradicted calls: commit: %w", err)
	}
	return withdrawn, rebound, nil
}
