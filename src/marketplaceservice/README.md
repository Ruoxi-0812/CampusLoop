# Persistent campus marketplace

A Go HTTP service backed by PostgreSQL implements the resale workflow through the original CampusLoop pages (`/`, `/product/{id}`, `/post-item`, `/my-listings`, `/login`): register/sign in, publish an item, browse shared listings, reserve an item, cancel a reservation, and confirm a handoff. It is separate from the inherited mock payment/checkout flow and the static Vercel demo. Accounts here use server-verified sessions, not the demo's localStorage identity.

## Run locally

From the repository root:

```sh
docker compose -f src/marketplaceservice/compose.yaml up -d --build
```

Open http://127.0.0.1:8081/. The compose frontend uses the existing Go templates and CSS, not a second storefront. Use two browser sessions/tabs to sign in as a seller and a buyer. The service stores a session token in sessionStorage per tab; listings, accounts, reservations and request keys are stored in PostgreSQL. No email is sent and email ownership is not verified.

The compose file binds service and database ports to localhost. Its credentials are local development fixtures. The named `marketplace-data` volume persists across container recreation. Stop with `docker compose -f src/marketplaceservice/compose.yaml stop`; do not delete the volume if you want to keep listings.

For Go development, start only `db`, then in this directory:

```sh
DATABASE_URL='postgres://campusloop:campusloop-local-only@127.0.0.1:55432/campusloop?sslmode=disable' go run .
```

`DATABASE_URL` is required; `LISTEN_ADDR` defaults to `127.0.0.1:8090`. Startup applies the embedded bootstrap schema in a transaction protected by a PostgreSQL advisory lock, permitting concurrent service starts. The form integration adds metadata and display-name columns with explicit additive `ALTER TABLE` statements. Future schema changes should use versioned migrations; `IF NOT EXISTS` alone is not an upgrade mechanism.

## Existing Go frontend integration

Build/run the existing frontend with `MARKETPLACE_SERVICE_URL=http://127.0.0.1:8090` (or the service's internal origin when containerized). The original home, product, login, post-item and my-listings pages render persistent marketplace data, and `/api/marketplace/*` is proxied on the same origin, including `BASE_URL` prefixes. `/marketplace` redirects to the original home page. The frontend does not trust or translate demo accounts into authenticated users. Without `MARKETPLACE_SERVICE_URL`, the original demo mode is unchanged. `MARKETPLACE_ONLY=true` runs these existing frontend pages without requiring the legacy checkout microservices; compose enables this mode. The Vercel static deployment does not execute this service.

## Transaction design

- Money is integer cents, with validation in both the HTTP API and database.
- All state changes lock the listing row first with `SELECT ... FOR UPDATE`, then check current state and actor permissions. Different listings can proceed independently.
- A partial unique index permits at most one `active` reservation per listing, including direct database writes. There is no process-local reservation mutex.
- Reservation creation, listing state and audit event commit together. A failure rolls back all three writes.
- `Idempotency-Key` is required to reserve. Keys are scoped to the authenticated buyer and persisted with the reservation. Same buyer/key/listing returns the same reservation; reusing a key for a different listing returns 409. A retry after cancellation/completion returns that terminal reservation and cannot resurrect it.
- A buyer can cancel their active reservation, returning the item to `available`. Only the seller can confirm handoff, moving the item to `sold`. Repeated identical transitions are no-ops; incompatible transitions return 409. Replaying an old cancellation never releases a newer reservation.
- Audit events record actor and transition in the transaction. Passwords use bcrypt; random bearer session tokens are stored as SHA-256 digests with 24-hour expiry and logout revocation.

State transitions:

```text
available -- buyer reserves --> reserved -- seller confirms --> sold
                                  |
                            buyer cancels
                                  |
                              available
```

Reservations are `active -> cancelled` or `active -> completed`; terminal reservations remain for audit and idempotency. The seller cannot reserve their own item.

## API

All write bodies are JSON. Authenticated endpoints require `Authorization: Bearer <token>`; there are no cookie credentials or permissive CORS rules.

| Method | Path under `/api/marketplace` | Behavior |
|---|---|---|
| POST | `/auth/register` | `{email,password,name?}`; create account and session |
| POST | `/auth/login` | `{email,password}`; create session |
| POST | `/logout` | Revoke current token |
| GET | `/me` | Current user ID |
| GET | `/listings` | Latest 100 shared listings |
| POST | `/listings` | `{title,description,price_cents,pickup,metadata?}`; seller comes from session |
| POST | `/listings/{id}/reserve` | Require `Idempotency-Key`; 201 on creation, 200 on replay, 409 on conflict |
| GET | `/reservations` | Latest 100 reservations involving the current buyer or seller |
| POST | `/reservations/{id}/cancel` | Buyer-only cancellation |
| POST | `/reservations/{id}/complete` | Seller-only completion |

`GET /healthz` checks database connectivity. No arbitrary seller/buyer identity is accepted in write payloads.

## Verification

Start the local database, then from this directory:

```sh
TEST_DATABASE_URL='postgres://campusloop:campusloop-local-only@127.0.0.1:55432/campusloop?sslmode=disable' \
MARKETPLACE_TEST_REPORT=../../docs/performance/marketplace-concurrency.json \
go test -race -count=1 -v ./...
```

Tests create an isolated random PostgreSQL schema and drop only that schema on cleanup. Use a development/test database and an account permitted to create schemas. Tests skip if `TEST_DATABASE_URL` is absent; CI always provisions PostgreSQL and sets it.

Coverage includes 100 different buyers contending on one listing over two HTTP servers with independent connection pools (10 rounds / 1,000 requests), 30 concurrent retries with one idempotency key, direct unique-index enforcement, authorization, cancellation/re-reservation, competing cancellation/completion, persisted state through a new service pool, and injected audit-write failure to demonstrate transaction rollback. Two test servers share the test process; they have separate application objects and DB pools. This is not a distributed production deployment or a load-capacity claim.

## Deliberate limits

This implements reservation and in-person handoff, not payment settlement or real-time messaging. Reservations do not auto-expire; cancellation is explicit. Listings/reservations are capped to the latest 100 in the UI; pagination, account recovery, email verification, moderation, session cleanup and comprehensive abuse protection remain future work. Production use needs HTTPS, managed credentials/database backups, and a reviewed authentication/deployment setup. Local compose is the verified deployment; Kubernetes deployment of this new service is not included.

## Original catalog and form fields

Existing title, description, price, category, photo, city, campus, neighborhood, pickup, contact and handoff fields persist through the original form. Uploaded PNG/JPEG/GIF/WebP images are limited to 2 MB and served from an image endpoint. New listings start available; reservation and handoff control later status changes.

To import the original nine demo products without overwriting existing data, run from this directory after starting compose:

```sh
DATABASE_URL='postgres://campusloop:campusloop-local-only@127.0.0.1:55432/campusloop?sslmode=disable' \
SEED_PASSWORD='choose-a-local-demo-password' go run ./cmd/seed
```

The demo seller is `catalog-demo@example.test`; original product IDs and image paths are preserved. The independent prototype storefront has been removed. Backend port 8090 serves APIs; its old `/marketplace` URL redirects to the original frontend when `FRONTEND_URL` is set.
