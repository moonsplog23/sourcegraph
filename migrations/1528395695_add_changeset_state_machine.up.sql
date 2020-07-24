BEGIN;

-- Now we add the 'state' field to changesets.
-- See ./internal/campaigns/types.go for the possible values here:
--   - UNPUBLISHED
--   - PUBLISHING
--   - ERRORED
--   - SYNCED
-- We use UNPUBLISHED was the default value
ALTER TABLE changesets ADD COLUMN IF NOT EXISTS state text DEFAULT 'UNPUBLISHED';

-- Since changesets can now be created in an "unpublished" state, we need to
-- make the following columns nullable:
ALTER TABLE changesets ALTER COLUMN external_id DROP NOT NULL;
ALTER TABLE changesets ALTER COLUMN metadata DROP NOT NULL;
ALTER TABLE changesets ALTER COLUMN external_service_type DROP NOT NULL;

-- But before switching to the new flow every changeset we had has been created
-- on the code host and is synced.
UPDATE changesets SET state = 'SYNCED';

-- EXPERIMENTAL: making workerutil work with changesets
ALTER TABLE changesets ADD COLUMN IF NOT EXISTS worker_state text DEFAULT 'queued';
ALTER TABLE changesets ADD COLUMN IF NOT EXISTS failure_message text;
ALTER TABLE changesets ADD COLUMN IF NOT EXISTS started_at timestamp with time zone;
ALTER TABLE changesets ADD COLUMN IF NOT EXISTS finished_at timestamp with time zone;
ALTER TABLE changesets ADD COLUMN IF NOT EXISTS process_after timestamp with time zone;
ALTER TABLE changesets ADD COLUMN IF NOT EXISTS num_resets integer NOT NULL DEFAULT 0;

COMMIT;
