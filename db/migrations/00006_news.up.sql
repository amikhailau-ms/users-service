
CREATE TABLE news (
  id varchar(30) primary key,
  created_at timestamptz DEFAULT current_timestamp,
  updated_at timestamptz DEFAULT NULL,
  description varchar(1000) NOT NULL,
  image bytea NOT NULL
);

CREATE TRIGGER news_updated_at
  BEFORE UPDATE OR INSERT ON news
  FOR EACH ROW
  EXECUTE PROCEDURE set_updated_at();

