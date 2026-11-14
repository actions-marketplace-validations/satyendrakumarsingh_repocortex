package registry

import (
	"context"
	"fmt"
)

// PruneImpossibleEdges removes heuristic edges that violate fundamental language invariants:
//   - A CALLS edge whose target is a configuration key, table, column, dependency line,
//     field, or deployment unit. None of these are callable in any language.
//   - An EXTENDS or IMPLEMENTS edge whose target is not a type (class, interface, package, type_alias).
//
// Any symbol_references that were bound to those impossible targets are reset to 'unresolved'
// so that the reference table and edge table remain in sync.
func (r *Registry) PruneImpossibleEdges(ctx context.Context, runID int64) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("registry: prune impossible edges: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `CREATE TEMP TABLE IF NOT EXISTS impossible_edge (id INTEGER PRIMARY KEY)`); err != nil {
		return 0, fmt.Errorf("registry: prune impossible edges: create temp: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM impossible_edge`); err != nil {
		return 0, fmt.Errorf("registry: prune impossible edges: clear temp: %w", err)
	}

	res, err := tx.ExecContext(ctx, `
		INSERT INTO impossible_edge (id)
		SELECT e.id
		  FROM edges e
		  JOIN symbols s ON s.id = e.dst_id
		 WHERE e.index_run_id = ?
		   AND e.source <> 'compiler'
		   AND (
		       (e.kind = 'CALLS' AND s.kind IN ('configuration', 'column', 'table', 'dependency', 'field', 'deployment_unit'))
		       OR
		       (e.kind IN ('EXTENDS', 'IMPLEMENTS') AND s.kind NOT IN ('class', 'interface', 'package', 'type_alias'))
		   )`, runID)
	if err != nil {
		return 0, fmt.Errorf("registry: prune impossible edges: select: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return 0, nil
	}

	// Reset any reference.call that pointed to an impossible target
	if _, err := tx.ExecContext(ctx, `
		UPDATE symbol_references
		   SET resolution = 'unresolved', resolved_symbol_id = NULL, resolution_source = ''
		 WHERE index_run_id = ?
		   AND kind = 'reference.call'
		   AND resolved_symbol_id IN (
		       SELECT id FROM symbols
		        WHERE index_run_id = ?
		          AND kind IN ('configuration', 'column', 'table', 'dependency', 'field', 'deployment_unit')
		   )`, runID, runID); err != nil {
		return 0, fmt.Errorf("registry: prune impossible edges: reset references: %w", err)
	}

	// Delete the impossible edges
	if _, err := tx.ExecContext(ctx, `DELETE FROM edges WHERE id IN (SELECT id FROM impossible_edge)`); err != nil {
		return 0, fmt.Errorf("registry: prune impossible edges: delete: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("registry: prune impossible edges: commit: %w", err)
	}
	return int(n), nil
}

// ResetHeuristicResolutions resets references resolved heuristically back to 'pending'
// and drops all non-compiler edges for the run, allowing a full re-resolution.
func (r *Registry) ResetHeuristicResolutions(ctx context.Context, runID int64) (refsReset, edgesDropped int, err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("registry: reset heuristic: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	resRefs, err := tx.ExecContext(ctx, `
		UPDATE symbol_references
		   SET resolution = 'pending', resolved_symbol_id = NULL, resolution_source = ''
		 WHERE index_run_id = ?
		   AND resolution = 'resolved'
		   AND (resolution_source <> 'compiler' OR resolution_source IS NULL OR resolution_source = '')`, runID)
	if err != nil {
		return 0, 0, fmt.Errorf("registry: reset heuristic: update references: %w", err)
	}
	nRefs, _ := resRefs.RowsAffected()

	resEdges, err := tx.ExecContext(ctx, `
		DELETE FROM edges
		 WHERE index_run_id = ?
		   AND source <> 'compiler'`, runID)
	if err != nil {
		return 0, 0, fmt.Errorf("registry: reset heuristic: delete edges: %w", err)
	}
	nEdges, _ := resEdges.RowsAffected()

	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("registry: reset heuristic: commit: %w", err)
	}
	return int(nRefs), int(nEdges), nil
}
