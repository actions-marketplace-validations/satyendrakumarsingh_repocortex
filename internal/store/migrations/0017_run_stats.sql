-- Per-run binding statistics, computed when the index is written.
--
-- Every relationship answer opens with a coverage statement ("only 59% of the
-- TypeScript references that name something this index defines are bound"). The
-- numbers behind it depend on the index revision and on nothing the caller
-- asked, but they were recomputed by every process that answered: four GROUP BYs
-- that read every reference row and every edge. On a 1.9-million-reference
-- index that was 6 to 8 seconds of the 9 to 17 a `find_callers` took, and none of
-- it was specific to the question.
--
-- run_language_stats holds one row per language of the run: what registry's
-- LanguageBindings returns, so the run-wide totals (ResolutionStats) are its
-- sums. run_stats says whether those rows still describe the run.
--
-- VALIDITY. The rows are a cache of the tables they summarise, and a cache that
-- can be stale is worse than none: an answer would state a coverage the index no
-- longer has. So every write that could change a figure clears run_stats.valid
-- (the triggers below), and a reader trusts the rows only while valid = 1.
-- RefreshRunStats rewrites the rows and sets it again; `resolve`, `scip import`
-- and the incremental update call it when they finish. An index written by an
-- older build has no run_stats row at all, which a reader treats as "not
-- computed": it falls back to the live query, exactly as before this migration.
--
-- The triggers are statement-cheap on purpose: one primary-key probe that
-- matches no row once the flag is already clear, so a bulk write pays for it
-- once per row and not once per figure.
CREATE TABLE run_stats (
    index_run_id  INTEGER PRIMARY KEY REFERENCES index_runs(id) ON DELETE CASCADE,
    valid         INTEGER NOT NULL DEFAULT 0 CHECK (valid IN (0,1)),
    by_compiler   INTEGER NOT NULL DEFAULT 0,
    computed_unix INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE run_language_stats (
    index_run_id  INTEGER NOT NULL REFERENCES index_runs(id) ON DELETE CASCADE,
    language      TEXT    NOT NULL,
    files         INTEGER NOT NULL DEFAULT 0,
    definitions   INTEGER NOT NULL DEFAULT 0,
    relationships INTEGER NOT NULL DEFAULT 0,
    hierarchy     INTEGER NOT NULL DEFAULT 0,
    total         INTEGER NOT NULL DEFAULT 0,
    resolved      INTEGER NOT NULL DEFAULT 0,
    ambiguous     INTEGER NOT NULL DEFAULT 0,
    unresolved    INTEGER NOT NULL DEFAULT 0,
    ignored       INTEGER NOT NULL DEFAULT 0,
    pending       INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (index_run_id, language)
) WITHOUT ROWID;

CREATE TRIGGER run_stats_files_ins AFTER INSERT ON files
BEGIN UPDATE run_stats SET valid = 0 WHERE index_run_id = NEW.index_run_id AND valid = 1; END;
CREATE TRIGGER run_stats_files_del AFTER DELETE ON files
BEGIN UPDATE run_stats SET valid = 0 WHERE index_run_id = OLD.index_run_id AND valid = 1; END;
CREATE TRIGGER run_stats_files_upd AFTER UPDATE OF language ON files
BEGIN UPDATE run_stats SET valid = 0 WHERE index_run_id = NEW.index_run_id AND valid = 1; END;

CREATE TRIGGER run_stats_symbols_ins AFTER INSERT ON symbols
BEGIN UPDATE run_stats SET valid = 0 WHERE index_run_id = NEW.index_run_id AND valid = 1; END;
CREATE TRIGGER run_stats_symbols_del AFTER DELETE ON symbols
BEGIN UPDATE run_stats SET valid = 0 WHERE index_run_id = OLD.index_run_id AND valid = 1; END;

CREATE TRIGGER run_stats_refs_ins AFTER INSERT ON symbol_references
BEGIN UPDATE run_stats SET valid = 0 WHERE index_run_id = NEW.index_run_id AND valid = 1; END;
CREATE TRIGGER run_stats_refs_del AFTER DELETE ON symbol_references
BEGIN UPDATE run_stats SET valid = 0 WHERE index_run_id = OLD.index_run_id AND valid = 1; END;
CREATE TRIGGER run_stats_refs_upd AFTER UPDATE OF resolution, resolution_source ON symbol_references
BEGIN UPDATE run_stats SET valid = 0 WHERE index_run_id = NEW.index_run_id AND valid = 1; END;

CREATE TRIGGER run_stats_edges_ins AFTER INSERT ON edges
BEGIN UPDATE run_stats SET valid = 0 WHERE index_run_id = NEW.index_run_id AND valid = 1; END;
CREATE TRIGGER run_stats_edges_del AFTER DELETE ON edges
BEGIN UPDATE run_stats SET valid = 0 WHERE index_run_id = OLD.index_run_id AND valid = 1; END;
