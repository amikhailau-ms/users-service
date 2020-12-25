BEGIN;

ALTER TABLE store_items DROP COLUMN on_sale;
ALTER TABLE store_items DROP COLUMN sale_coins_price;
ALTER TABLE store_items DROP COLUMN sale_gems_price;

COMMIT;
