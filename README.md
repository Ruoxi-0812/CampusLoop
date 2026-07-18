# CampusLoop

CampusLoop is a student-to-student second-hand marketplace for Northeastern University students. It helps students browse used textbooks, dorm essentials, furniture, electronics, clothing, and daily supplies, then coordinate pickup around campus.

Live demo: [https://neuloop.vercel.app](https://neuloop.vercel.app)

## Highlights

- Campus-focused resale marketplace for student listings
- Polished homepage with searchable listings, category filters, and product cards
- Product detail pages with price, condition, status, pickup area, handoff time, and seller prompts
- Simple account flow with login, create-account, avatar state, messages, and my-listings pages
- Post-item flow for sellers with photo, title, price, category, campus, dorm area, pickup notes, contact method, and availability
- Static Vercel demo for resume viewing, plus Kubernetes manifests for the full microservices version

## Tech Stack

- **Frontend:** Go templates, HTML, CSS, JavaScript
- **Services:** Go, Node.js, Python, Java, C#
- **Communication:** gRPC and Protocol Buffers
- **Data/cache:** Redis cart store and JSON-backed product catalog
- **Platform:** Docker, Kubernetes, Kind, Skaffold, Vercel

## Architecture

CampusLoop uses a microservices commerce architecture. The frontend coordinates user-facing flows while separate services handle product catalog, cart, checkout, recommendation, ads, currency, payment, shipping, and email behavior.

| Service | Language | Role |
| --- | --- | --- |
| [frontend](src/frontend) | Go | Serves the CampusLoop web UI and marketplace flows |
| [productcatalogservice](src/productcatalogservice) | Go | Provides campus resale product data |
| [cartservice](src/cartservice) | C# | Stores cart data through Redis |
| [checkoutservice](src/checkoutservice) | Go | Coordinates checkout, payment, shipping, and email confirmation |
| [currencyservice](src/currencyservice) | Node.js | Converts listing prices between currencies |
| [paymentservice](src/paymentservice) | Node.js | Provides mock payment behavior for checkout |
| [shippingservice](src/shippingservice) | Go | Provides mock shipping quotes and tracking |
| [emailservice](src/emailservice) | Python | Sends mock order confirmation emails |
| [recommendationservice](src/recommendationservice) | Python | Recommends related products |
| [adservice](src/adservice) | Java | Provides promotional messages |
| [loadgenerator](src/loadgenerator) | Python/Locust | Generates demo traffic for testing |

Service definitions live in [protos/demo.proto](protos/demo.proto). Kubernetes manifests live in [release/kubernetes-manifests.yaml](release/kubernetes-manifests.yaml).

## Project Structure

- [src/frontend/templates](src/frontend/templates): shared layout, homepage, product, account, messages, listings, and post-item pages
- [src/frontend/static/styles/styles.css](src/frontend/static/styles/styles.css): CampusLoop visual design
- [src/frontend/static/icons](src/frontend/static/icons): CampusLoop logo and UI icons
- [src/frontend/static/img/products/campusloop](src/frontend/static/img/products/campusloop): product images
- [src/productcatalogservice/products.json](src/productcatalogservice/products.json): marketplace item catalog
- [vercel-demo](vercel-demo): static resume-ready website deployed on Vercel

## Local Development

Run the full microservices version with a local Kubernetes cluster:

```sh
kind create cluster --name campusloop
kubectl apply -f release/kubernetes-manifests.yaml
kubectl get pods
kubectl port-forward deployment/frontend 8081:8080
```

Then open:

```text
http://127.0.0.1:8081
```

## Vercel Demo

The resume demo is exported as a static site in [vercel-demo](vercel-demo). In Vercel, set the project root directory to:

```text
vercel-demo
```

Then deploy the project to production.

## Attribution

This project adapts the open-source [GoogleCloudPlatform/microservices-demo](https://github.com/GoogleCloudPlatform/microservices-demo) architecture and reworks it into a Northeastern-focused campus marketplace experience. The original project is licensed under the Apache License 2.0.
