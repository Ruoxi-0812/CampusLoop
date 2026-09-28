# Free public demo deployment

Run the existing Go frontend and marketplace API as two processes in one Render Free web-service container. PostgreSQL lives in Neon Free. This preserves the original pages without running the inherited commerce stack or Kubernetes online.

## Render and Neon

1. Create/select a Neon Free project and copy its PostgreSQL connection string with TLS enabled. Store it only in Render's secret `DATABASE_URL` environment variable, never in this repository.
2. Deploy `render.yaml` as a Render Blueprint, or create a Docker Web Service with repository-root context and `deploy/Dockerfile`. Select **Free**, not a paid plan. This repository's deployment branch must include the local implementation first.
3. Set `SEED_DEMO_CATALOG=true` and a generated secret `SEED_PASSWORD` to import the original nine catalog entries. Imports preserve existing rows. No local user accounts, sessions, reservations or test listings are uploaded. The sample catalog seller is `catalog-demo@example.test`.
4. The public endpoint serves the existing homepage, login, publishing, product detail and reservation pages. The API process listens only on loopback inside the container.
5. Verify the Render URL before changing Vercel routing. Check `/`, `/_healthz`, and `/api/marketplace/listings`, then test registration and a temporary listing/reservation with dedicated test accounts.

`start.sh` waits for database migration and API readiness, optionally seeds the catalog, then starts the frontend on Render's `PORT`. It forwards termination to both processes and exits if either fails. Health checks use frontend `/_healthz` so monitoring does not continuously query and wake the database.

## Existing Vercel URL

Vercel serves the homepage and static assets immediately, then loads live listings asynchronously from Render. Other pages use a branded connection screen while retrieving the original Go-rendered page through `/_pages/*`. API requests are proxied through `/api/*`. See `vercel-demo/README.md` for routing and asset synchronization details. Deploy a preview and verify it before promoting to production.

## Zero-cost constraints

Use the Free plans and avoid paid upgrades. Render's free instance hours are shared by workspace; using one instance avoids two services drawing from that pool. Free services sleep after inactivity and may take about a minute to wake. Neon Free has storage/compute/transfer limits. With no Render payment method, excess bandwidth suspends services and excess build minutes stop new builds instead of billing. If the account already has billing enabled, review spending controls before deployment. These are limited free tiers, not a guarantee of unlimited free hosting or permanent availability.

Do not use Render's 30-day free PostgreSQL trial for this persistent demo. Do not configure synthetic keep-alive traffic. The original static Vercel version remains available until the new origin is verified.
