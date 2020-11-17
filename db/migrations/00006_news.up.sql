BEGIN;

CREATE TABLE news (
  id varchar primary key,
  created_at timestamptz DEFAULT current_timestamp,
  updated_at timestamptz DEFAULT NULL,
  title varchar NOT NULL,
  description text NOT NULL,
  image_link varchar NOT NULL,
  UNIQUE(title)
);

CREATE TRIGGER news_updated_at
  BEFORE UPDATE OR INSERT ON news
  FOR EACH ROW
  EXECUTE PROCEDURE set_updated_at();

COMMIT;
