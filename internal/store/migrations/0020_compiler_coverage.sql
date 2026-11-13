-- 0020_compiler_coverage: which files a compiler indexer read.
--
-- WHY A TABLE. An answer about who calls a definition counts the references
-- that name it and are bound to nothing: any of them might be a call of the
-- definition. A reference in a file a compiler read, spelled like a name the
-- compiler resolved to something else, is not one. To say so the answer layer
-- must know that the compiler READ the file, and the edge table cannot say it:
-- a file that only calls code outside the repository carries no compiler edge,
-- and a file the indexer skipped (outside its tsconfig, an unreadable module)
-- carries none either. The two must not look alike, because treating a skipped
-- file as read is how an answer comes to say "complete" about a file nobody
-- looked at. The Go type check could use its edges for this because it reads
-- every file of a package it loads; a SCIP indexer's document list is the only
-- record of which files it covered.
--
-- One row per file of the run that appeared as a document of an index imported
-- from a compiler indexer this tool ran itself. A file's id changes when the
-- file is re-indexed, so a changed file drops its row (ON DELETE CASCADE) and
-- is no longer counted until the indexer has read it again.
CREATE TABLE compiler_coverage (
    index_run_id INTEGER NOT NULL REFERENCES index_runs(id) ON DELETE CASCADE,
    file_id      INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    tool         TEXT    NOT NULL,
    PRIMARY KEY (index_run_id, file_id)
) WITHOUT ROWID;
