# CampusLoop Project Plan

This project starts from Google Cloud's Online Boutique demo, reproduced locally with Kind and Kubernetes. The goal is to keep the microservices learning value, then gradually turn the store into CampusLoop, a campus second-hand marketplace.

## Current Local Status

- Project code is now in `/Users/carol/Documents/Microservices Commerce Platform`.
- The local Kubernetes context is `kind-online-boutique`.
- The storefront is reachable at `http://127.0.0.1:8081` while port-forwarding is running.
- Verified services: 12 pods are running. After rebuilding the frontend image, the browser page title should become `CampusLoop`.

Useful commands:

```bash
kubectl get pods
kubectl port-forward deployment/frontend 8081:8080
```

If the app stops responding, first check `kubectl get pods`, then restart the port-forward command.

## How The Demo Is Organized

The app is intentionally split by business capability:

- `src/frontend`: Go web frontend, page routes, templates, static CSS/icons.
- `src/productcatalogservice`: Go product catalog service. The easiest first product change is `products.json`.
- `src/cartservice`: C# cart service backed by Redis.
- `src/checkoutservice`: Go checkout orchestration service.
- `src/paymentservice`: Node.js mock payment service.
- `src/shippingservice`: Go mock shipping quote service.
- `src/emailservice`: Python mock email confirmation service.
- `src/recommendationservice`: Python recommendation service.
- `src/currencyservice`: Node.js currency conversion service.
- `src/adservice`: Java ad service.
- `src/loadgenerator`: Locust traffic generator for demo load.
- `release/kubernetes-manifests.yaml`: fastest local deployment path using prebuilt images.
- `kubernetes-manifests/`: editable Kubernetes manifests by service.

## First Changes To Make It Ours

Start with low-risk visible changes before changing service contracts:

1. Rename the brand from `Online Boutique` to `CampusLoop`.
2. Replace product data in `src/productcatalogservice/products.json` with campus second-hand items.
3. Update frontend text, colors, and logos in `src/frontend/templates` and `src/frontend/static`.
4. Keep the original cart, checkout, payment, shipping, and recommendation flows working.
5. After the UI/product identity is ours, add one custom feature, such as item condition, seller location, pickup status, inventory status, admin product editing, or order history.

## Learning Map

Use this order when studying the system:

1. `README.md`: high-level architecture and deployment commands.
2. `src/frontend/main.go`: route setup and service connections.
3. `src/frontend/handlers.go`: user-facing flows, including home, product detail, cart, and checkout.
4. `src/productcatalogservice/products.json`: product source data.
5. `src/productcatalogservice/server.go`: product catalog API behavior.
6. `src/checkoutservice/main.go`: how checkout calls cart, payment, shipping, and email services.
7. `kubernetes-manifests/*.yaml`: how each service becomes a Kubernetes deployment/service.

## Suggested Project Direction

Good personal-project framing:

> I reproduced Google's Online Boutique microservices demo locally, then evolved it into CampusLoop, a campus second-hand marketplace. I used Kubernetes to run independent services, studied the service boundaries, customized the product catalog and frontend identity, and planned feature extensions around item condition, pickup flow, and order workflows.

This keeps the project honest: it starts as a reproduction, then becomes stronger as we replace default data, branding, and features with our own decisions.
