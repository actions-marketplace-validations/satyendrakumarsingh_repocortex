-- Indexes behind the per-question reads of a relationship answer, and the count
-- the dead-code scan needs.
--
-- Once the coverage figures are stored (0017), what is left of a relationship
-- answer's cost on a large index is three reads that are about ONE symbol and
-- were answered by walking the table for it:
--
--   NameBinding   "how were the references spelled like this name resolved":
--                 an index range over the name, then a fetch of every one of
--                 those rows to read two columns. A hub name has tens of
--                 thousands of references.
--   BoundSites    "which references are bound to this symbol, and from where":
--                 a probe on resolved_symbol_id, then a fetch of each row for
--                 its line, receiver and receiver kind.
--   the dead-code probe "is a reference spelled like this still unbound":
--                 the same range, once per definition scanned.
--
-- Each index below carries every column its read needs, so the read never
-- touches the table. They cost about 7% of the database file and a little write
-- time; they take a second call on a 1.9-million-reference index from 600 to
-- 220 milliseconds.
CREATE INDEX IF NOT EXISTS idx_refs_name_binding
    ON symbol_references (index_run_id, name, resolution, resolved_symbol_id);

-- Partial: only a bound reference is ever asked about by the symbol it is bound
-- to, and those are about 45% of the table.
CREATE INDEX IF NOT EXISTS idx_refs_bound_sites
    ON symbol_references (resolved_symbol_id, file_id, start_line, receiver, receiver_kind)
    WHERE resolution = 'resolved';

-- How many definitions the dead-code scan would examine, stored with the other
-- figures of the run (-1: not computed). It lets a scan limited to one
-- directory read only that directory's definitions and still report the number
-- the whole scan would have examined.
ALTER TABLE run_stats ADD COLUMN unreferenced INTEGER NOT NULL DEFAULT -1;
