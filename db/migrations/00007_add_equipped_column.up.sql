BEGIN;

ALTER TABLE users_store_items ADD COLUMN equipped boolean DEFAULT FALSE;

COMMIT;
