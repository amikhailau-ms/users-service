
CREATE TABLE store_items (
  id varchar(30) primary key,
  user_id varchar(30),
  created_at timestamptz DEFAULT current_timestamp,
  updated_at timestamptz DEFAULT NULL,
  name varchar(255) DEFAULT NULL,
  type int NOT NULL,
  image bytea NOT NULL,
  description varchar(400) DEFAULT NULL
  coins_price int NOT NULL,
  gems_price int NOT NULL,
  CONSTRAINT store_items_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TRIGGER store_items_updated_at
  BEFORE UPDATE OR INSERT ON store_items
  FOR EACH ROW
  EXECUTE PROCEDURE set_updated_at();

