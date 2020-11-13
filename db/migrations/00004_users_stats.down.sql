BEGIN;

DROP TRIGGER users_stats_updated_at on users_stats;

DROP TABLE users_stats;

COMMIT;
