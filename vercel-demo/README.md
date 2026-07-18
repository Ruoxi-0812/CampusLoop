# CampusLoop Vercel Demo

This is a lightweight static demo of CampusLoop for portfolio and resume use.
It mirrors the main marketplace flow without requiring the full GKE microservices stack to stay online.

## What It Shows

- Browse Northeastern student listings
- Search and category filters
- Product detail pages
- Login/create account state with localStorage
- Seller quick-message flow and messages page
- Post item and my listings pages
- Cart and campus pickup checkout mock flow

## Local Preview

From this folder:

```bash
python3 -m http.server 5173
```

Then open:

```text
http://127.0.0.1:5173
```

## Vercel

Deploy this folder as the Vercel project root. No build command is required.

Suggested settings:

- Framework Preset: Other
- Build Command: leave empty
- Output Directory: leave empty
- Install Command: leave empty

The production version can live on Vercel, while the full microservices version remains deployable on GKE.
