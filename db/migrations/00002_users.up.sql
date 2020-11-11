
CREATE TABLE users (
  id varchar(30) primary key,
  email varchar(255) NOT NULL,
  password varchar(40) NOT NULL,
  created_at timestamptz DEFAULT current_timestamp,
  updated_at timestamptz DEFAULT NULL,
  name varchar(255) DEFAULT NULL,
  coins int DEFAULT 0,
  gems int DEFAULT 0,
  is_admin boolean DEFAULT FALSE,
  UNIQUE(name)
);

CREATE TRIGGER users_updated_at
  BEFORE UPDATE OR INSERT ON users
  FOR EACH ROW
  EXECUTE PROCEDURE set_updated_at();

