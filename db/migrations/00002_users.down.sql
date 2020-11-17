BEGIN;

DROP TRIGGER users_updated_at on users;

DROP TABLE users;

COMMIT;
