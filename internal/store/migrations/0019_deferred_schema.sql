-- Schema objects set aside while an index is first loaded.
--
-- A first index writes every symbol and reference row once, in file order,
-- into tables that hold nothing else. Keeping eight secondary indexes and the
-- run_stats triggers current row by row costs more than the rest of the sink
-- put together: each index is written at a random place, and every first touch
-- of a page inside the per-file savepoint copies that page into the statement
-- journal. Building the same indexes from the finished table is one sort each.
--
-- So `repocortex index` drops those objects before the first load of an empty
-- index and creates them again before it publishes the run. Each dropped
-- object's CREATE statement is stored here, in the same transaction as the
-- drop, so the two can never disagree: a row in this table IS an object the
-- schema is missing. The statement text is copied from sqlite_master, which
-- means the objects come back exactly as the migrations made them and this
-- table needs no second copy of any of them.
--
-- An interrupted run leaves its rows behind. The next `repocortex index`
-- restores them before doing anything else, and `repocortex status` and
-- `repocortex doctor` report them until it has.
CREATE TABLE deferred_schema (
    name          TEXT PRIMARY KEY,
    kind          TEXT NOT NULL CHECK (kind IN ('index','trigger')),
    sql           TEXT NOT NULL,
    deferred_unix INTEGER NOT NULL DEFAULT 0
);
