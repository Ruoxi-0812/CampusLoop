# CampusLoop edge frontend

Vercel serves `index.html`, `/edge/*` and `/static/*` without contacting Render.
The homepage initially displays nine explicitly labeled sample catalog items and
replaces them with current PostgreSQL listings after `/api/marketplace/listings`
responds. Search and category filtering work before and after that update.

Other public page routes serve `edge/gateway.html`. The gateway retrieves the
original Go-rendered page through `/_pages/*`, retaining the public URL, session,
forms, and original backend JavaScript. Only verified application HTML is rendered.
Render wake-up HTML is never displayed. Missing pages show an error, while temporary
failures retry for up to about 90 seconds and then offer a manual retry.

API routes still proxy directly to the original backend. Only page/data GETs are
retried. Mutations are not automatically replayed. There is no periodic keep-alive.

Deploy this directory as the existing Vercel project's root (Other framework,
no build command). Preview a deployment before promoting to production.
When backend frontend assets change, copy the corresponding files from
`src/frontend/static` into `static` before deploying the edge frontend.

Validation: simulate a 503 HTML backend response, verify the homepage and filters
remain available, verify a direct login URL displays the branded loading state,
then restore the backend and verify live listings and the original login form.
