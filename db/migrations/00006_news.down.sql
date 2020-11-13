BEGIN;

DROP TRIGGER news_updated_at on news;

DROP TABLE news;

COMMIT;
