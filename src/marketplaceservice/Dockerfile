FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /marketplace .
FROM alpine:3.22
RUN apk add --no-cache ca-certificates
COPY --from=build /marketplace /marketplace
USER 65532:65532
EXPOSE 8090
ENTRYPOINT ["/marketplace"]
