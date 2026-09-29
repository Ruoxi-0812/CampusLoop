# CampusLoop static frontend

Vercel serves complete pages immediately. Only JSON data is loaded asynchronously
from the Go API; navigation never downloads and replaces a second HTML document.
The existing Go templates generate the account, publishing and dashboard pages:

```sh
go run scripts/edge-export/main.go
cp src/frontend/static/js/marketplace.js vercel-demo/static/js/marketplace.js
```

Run from the repository root after changing templates. Product layout lives in
`scripts/edge-export/product-body.html`; product information is updated in place.
Samples or previously viewed details can appear before live data arrives; reservation
availability always comes from the API. Homepage sample cards are replaced by live listings.

GET requests retry during backend startup, preserving authorization headers.
Mutations wait for a read-only health check and are sent once, without automatic
replay. No periodic keep-alive is configured. API authentication and PostgreSQL
remain on the existing backend. Deploy this directory, preview, then promote.
