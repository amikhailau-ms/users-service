
CREATE TABLE users_store_items (
  id serial primary key,
  user_id varchar(30),
  store_item_id varchar(30),
  created_at timestamptz DEFAULT current_timestamp,
  updated_at timestamptz DEFAULT NULL,
  CONSTRAINT users_store_items_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
  CONSTRAINT users_store_items_store_item_id FOREIGN KEY(store_item_id) REFERENCES store_items(id) ON DELETE CASCADE
);

CREATE TRIGGER users_store_items_updated_at
  BEFORE UPDATE OR INSERT ON users_store_items
  FOR EACH ROW
  EXECUTE PROCEDURE set_updated_at();

