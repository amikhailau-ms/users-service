BEGIN;

CREATE TABLE store_items (
  id varchar primary key,
  created_at timestamptz DEFAULT current_timestamp,
  updated_at timestamptz DEFAULT NULL,
  name varchar DEFAULT NULL,
  type int NOT NULL,
  image_id varchar NOT NULL,
  description text DEFAULT NULL,
  coins_price int NOT NULL,
  gems_price int NOT NULL,
  UNIQUE(name),
  UNIQUE(image_id)
);

CREATE TRIGGER store_items_updated_at
  BEFORE UPDATE OR INSERT ON store_items
  FOR EACH ROW
  EXECUTE PROCEDURE set_updated_at();

COMMIT;
