# CampusLoop

CampusLoop is a campus second-hand marketplace built on a Kubernetes microservices architecture. Students can browse second-hand textbooks, dorm essentials, electronics, clothing, and study gear, then add items to a cart and move through a checkout flow.

This project started from Google Cloud's Online Boutique microservices demo and has been customized into a student-focused marketplace with new branding, product data, product images, logo, and a polished frontend experience.

## Features

- CampusLoop branding with custom logo and wordmark
- Campus resale product catalog with second-hand student items
- Marketplace-style homepage with hero CTA, category chips, listing cards, price pills, and condition/pickup badges
- Product detail pages with listing information such as condition, pickup location, and verified student seller status
- Cart, checkout, recommendation, currency, payment, shipping, email, and ad services running as independent microservices
- Kubernetes manifests for local or cloud deployment

## Tech Stack

- **Frontend:** Go templates, HTML, CSS
- **Backend services:** Go, Node.js, Python, Java, C#
- **Service communication:** gRPC and Protocol Buffers
- **Data/cache:** Redis cart store and JSON-backed product catalog
- **Platform:** Docker, Kubernetes, Kind, Skaffold

## Architecture

CampusLoop keeps the microservices structure of the original demo while changing the product and user experience layer into a campus marketplace.

| Service | Language | Role |
| --- | --- | --- |
| [frontend](src/frontend) | Go | Serves the CampusLoop web UI and coordinates user-facing flows |
| [productcatalogservice](src/productcatalogservice) | Go | Provides campus marketplace products from `products.json` |
| [cartservice](src/cartservice) | C# | Stores and retrieves cart data through Redis |
| [checkoutservice](src/checkoutservice) | Go | Orchestrates checkout, payment, shipping, and email confirmation |
| [currencyservice](src/currencyservice) | Node.js | Converts listing prices between currencies |
| [paymentservice](src/paymentservice) | Node.js | Mock payment service for checkout |
| [shippingservice](src/shippingservice) | Go | Mock shipping quote and tracking service |
| [emailservice](src/emailservice) | Python | Sends mock order confirmation emails |
| [recommendationservice](src/recommendationservice) | Python | Recommends related products |
| [adservice](src/adservice) | Java | Provides contextual promotional messages |
| [loadgenerator](src/loadgenerator) | Python/Locust | Generates demo traffic for testing |

The service definitions live in [`protos/demo.proto`](protos/demo.proto), and Kubernetes manifests live in [`release/kubernetes-manifests.yaml`](release/kubernetes-manifests.yaml).

## Local Development

The project has been tested locally with Docker, Kind, and Kubernetes.

1. Create or use a local Kubernetes cluster.

   ```sh
   kind create cluster --name online-boutique
   ```

2. Deploy the services.

   ```sh
   kubectl apply -f release/kubernetes-manifests.yaml
   ```

3. Wait for pods to become ready.

   ```sh
   kubectl get pods
   ```

4. Open the frontend locally.

   ```sh
   kubectl port-forward deployment/frontend 8081:8080
   ```

   Then visit:

   ```text
   http://127.0.0.1:8081
   ```

## Development Notes

Useful files for customizing CampusLoop:

- [`src/frontend/templates/home.html`](src/frontend/templates/home.html): homepage layout
- [`src/frontend/templates/header.html`](src/frontend/templates/header.html): logo, navigation, and shared header
- [`src/frontend/templates/product.html`](src/frontend/templates/product.html): product detail page
- [`src/frontend/static/styles/styles.css`](src/frontend/static/styles/styles.css): CampusLoop visual design
- [`src/productcatalogservice/products.json`](src/productcatalogservice/products.json): product catalog data
- [`src/frontend/static/img/products/campusloop`](src/frontend/static/img/products/campusloop): custom product images

## Project Direction

Planned improvements:

- Make category filters interactive
- Add a "Sell an item" listing form
- Add campus pickup preferences and item condition fields
- Add project-specific screenshots and architecture documentation
- Prepare a cloud deployment path on GKE Autopilot

## Attribution

CampusLoop is based on [GoogleCloudPlatform/microservices-demo](https://github.com/GoogleCloudPlatform/microservices-demo). The original project demonstrates a cloud-first ecommerce application composed of independent microservices. This repository adapts that architecture into a campus second-hand marketplace concept.
