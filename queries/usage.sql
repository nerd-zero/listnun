-- name: add-usage-counters
-- Adds each row's count to its (tenant_id, kind, bucket) counter --
-- additive, never overwriting, so concurrent writers and retried flushes
-- can't lose or replace counts. Callers pass one row per key (see
-- internal/usage.Meter, which aggregates before writing): ON CONFLICT
-- can't update the same row twice in one statement.
INSERT INTO usage_counters (tenant_id, kind, bucket, count)
SELECT * FROM UNNEST($1::INTEGER[], $2::TEXT[], $3::TIMESTAMPTZ[], $4::BIGINT[])
ON CONFLICT (tenant_id, kind, bucket) DO UPDATE SET count = usage_counters.count + EXCLUDED.count;
