CREATE TABLE IF NOT EXISTS marketplace_users (
 id text PRIMARY KEY,
 email text NOT NULL UNIQUE,
 password_hash text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS marketplace_sessions (
 token_hash text PRIMARY KEY,
 user_id text NOT NULL REFERENCES marketplace_users(id),
 expires_at timestamptz NOT NULL
);
CREATE TABLE IF NOT EXISTS marketplace_listings (
 id text PRIMARY KEY,
 seller_id text NOT NULL REFERENCES marketplace_users(id),
 title text NOT NULL CHECK (char_length(title) BETWEEN 1 AND 120),
 description text NOT NULL CHECK (char_length(description) BETWEEN 1 AND 2000),
 price_cents integer NOT NULL CHECK (price_cents BETWEEN 0 AND 100000000),
 pickup text NOT NULL CHECK (char_length(pickup) BETWEEN 1 AND 200),
 status text NOT NULL DEFAULT 'available' CHECK (status IN ('available','reserved','sold')),
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS marketplace_listings_recent ON marketplace_listings(created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS marketplace_listings_seller ON marketplace_listings(seller_id);
CREATE TABLE IF NOT EXISTS marketplace_reservations (
 id text PRIMARY KEY,
 listing_id text NOT NULL REFERENCES marketplace_listings(id),
 buyer_id text NOT NULL REFERENCES marketplace_users(id),
 idempotency_key text NOT NULL CHECK (char_length(idempotency_key) BETWEEN 1 AND 128),
 status text NOT NULL DEFAULT 'active' CHECK (status IN ('active','cancelled','completed')),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE (buyer_id, idempotency_key)
);
CREATE UNIQUE INDEX IF NOT EXISTS marketplace_one_active_reservation ON marketplace_reservations(listing_id) WHERE status = 'active';
CREATE TABLE IF NOT EXISTS marketplace_reservation_events (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 reservation_id text NOT NULL REFERENCES marketplace_reservations(id),
 actor_id text NOT NULL REFERENCES marketplace_users(id),
 action text NOT NULL CHECK (action IN ('reserved','cancelled','completed')),
 created_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE marketplace_listings ADD COLUMN IF NOT EXISTS metadata jsonb NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE marketplace_users ADD COLUMN IF NOT EXISTS display_name text NOT NULL DEFAULT '';
