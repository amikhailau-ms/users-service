BEGIN;

CREATE TABLE users (
  id varchar primary key,
  email varchar NOT NULL,
  password varchar NOT NULL,
  created_at timestamptz DEFAULT current_timestamp,
  updated_at timestamptz DEFAULT NULL,
  name varchar DEFAULT NULL,
  coins int DEFAULT 0,
  gems int DEFAULT 0,
  is_admin boolean DEFAULT FALSE,
  UNIQUE(name),
  UNIQUE(email)
);

CREATE TRIGGER users_updated_at
  BEFORE UPDATE OR INSERT ON users
  FOR EACH ROW
  EXECUTE PROCEDURE set_updated_at();

COMMIT;
