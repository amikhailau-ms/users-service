BEGIN;

DROP TRIGGER user_stats_updated_at on user_stats;

DROP TABLE user_stats;

COMMIT;
