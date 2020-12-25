BEGIN;

ALTER TABLE store_items ADD COLUMN on_sale boolean DEFAULT FALSE;
ALTER TABLE store_items ADD COLUMN sale_coins_price int DEFAULT 0;
ALTER TABLE store_items ADD COLUMN sale_gems_price int DEFAULT 0;

COMMIT;
