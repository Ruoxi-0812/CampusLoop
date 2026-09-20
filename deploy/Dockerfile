FROM golang:1.25-alpine AS marketplace-build
WORKDIR /build/marketplace
COPY src/marketplaceservice/go.mod src/marketplaceservice/go.sum ./
RUN go mod download
COPY src/marketplaceservice/ ./
RUN CGO_ENABLED=0 go build -trimpath -o /out/marketplace . && CGO_ENABLED=0 go build -trimpath -o /out/seed ./cmd/seed

FROM golang:1.25-alpine AS frontend-build
WORKDIR /build/frontend
COPY src/frontend/go.mod src/frontend/go.sum ./
RUN go mod download
COPY src/frontend/ ./
RUN CGO_ENABLED=0 go build -trimpath -o /out/frontend .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates bash tini
WORKDIR /app/frontend
COPY --from=marketplace-build /out/marketplace /out/seed /app/
COPY --from=frontend-build /out/frontend /app/frontend/server
COPY src/frontend/templates ./templates
COPY src/frontend/static ./static
COPY src/productcatalogservice/products.json /app/products.json
COPY --chmod=755 deploy/start.sh /app/start.sh
USER 65532:65532
ENV PORT=10000
EXPOSE 10000
ENTRYPOINT ["/sbin/tini", "--", "/app/start.sh"]
