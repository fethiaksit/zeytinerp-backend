-- 016_seed_product_categories.up.sql
-- Idempotent seed of the approved Zeytin Market product categories.
INSERT INTO categories (name)
VALUES
  ('İçecek'),
  ('Atıştırmalık'),
  ('Süt Ürünleri'),
  ('Kahvaltılık'),
  ('Temel Gıda'),
  ('Bakliyat'),
  ('Unlu Mamuller'),
  ('Şarküteri'),
  ('Dondurma'),
  ('Dondurulmuş Gıda'),
  ('Temizlik'),
  ('Kişisel Bakım'),
  ('Kağıt Ürünleri'),
  ('Ev & Mutfak'),
  ('Tüp')
ON CONFLICT DO NOTHING;
