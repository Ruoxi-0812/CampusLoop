# Marketplace workflow verification

Implemented in `src/marketplaceservice`, with the original Go frontend pages rendering persistent data and proxying API requests on the same origin.

## Automated verification

The final service suite passed with `go test -race -count=1 -v ./...` against PostgreSQL 16 in the local Docker development database. Frontend tests also passed with `go test ./...`. The new GitHub Actions job provisions PostgreSQL and runs both suites; it has been configured locally, not yet executed on GitHub.

- **Contention:** 100 distinct authenticated buyers race to reserve one listing, repeated on 10 listings. Requests alternate across two HTTP server instances with separate application objects and PostgreSQL pools. All 10 rounds produced exactly one creation and 99 expected HTTP 409 conflicts: 1,000 attempts, 10 winners, 990 conflicts, no duplicate active reservations. Counts and timestamp are saved in `marketplace-concurrency.json`.
- **Retries:** 30 concurrent requests using the same buyer/key returned one reservation ID: one 201 and 29 replay responses (200).
- **Database defense:** a direct attempt to insert a second active reservation was rejected by the partial unique index.
- **Permissions and validation:** unauthenticated requests, self-reservation, unauthorized cancellation/completion, malformed prices, missing fields and spoofed seller IDs were rejected.
- **Lifecycle:** cancellation releases an item; another buyer can reserve it; replaying an old cancellation does not release the new reservation; seller completion makes it sold and prevents another reservation.
- **Competing transitions:** concurrent buyer cancellation and seller completion ran for 10 rounds, accepting only one transition each time and preserving listing/reservation/audit consistency.
- **Atomic rollback:** an injected failure on audit insertion rolled back the reservation and listing update together; retry after removing the failure succeeded.
- **Persistence:** another service pool reads committed states and stored idempotency results.

The two test servers run in one Go test process; this proves coordination through independently pooled database transactions, not a multi-host production deployment. The 990 conflict responses are expected business outcomes, not infrastructure errors. This is a correctness test, not a throughput benchmark or evidence of a percentage reduction from an earlier reservation implementation.

## Browser and container verification

Using two independent browser tabs and synthetic local accounts:

1. Signed in as the seller and published “Calculus textbook · Demo listing”.
2. Signed in as the buyer in a separate tab, observed the shared listing and reserved it.
3. Restarted the actual Marketplace Docker container.
4. Reloaded the buyer page: session and reservation remained valid; the item remained reserved.
5. Refreshed the seller view and confirmed the handoff.
6. Both views showed the reservation completed and the item sold.

No real payment, email or external transaction occurred. The PostgreSQL volume and local demo service are left running for review at http://127.0.0.1:8081/.

## Resume wording

```latex
\resumeItem{Built a Go/PostgreSQL listing and reservation service with transactional row locking, idempotent APIs, and database uniqueness constraints; validated single-winner reservations under 100-buyer contention across 1,000 test requests.}
```

This bullet describes the implemented business logic and tested contention scale. Pair it with the separate catalog-index performance bullet if space permits.

## Original-page integration verification

The independent prototype storefront was removed. The existing home, product, login, post-item and my-listings templates and purple CampusLoop styling now implement the workflow. `/marketplace` redirects to the original home page. The original nine catalog products were imported without overwriting data, preserving IDs and images.

Browser verification on the original pages: signed in through `/login`, published “Desk lamp · Original page test” using the full existing `/post-item` form, reserved it from its original-style detail page in an independent buyer session, cancelled through `/my-listings`, reserved again, and confirmed handoff as the seller. The final page showed `sold` and `completed`. Both frontend tests and real-PostgreSQL integration tests with the race detector passed. New tests also cover original template rendering, additional listing fields, rejection of unsafe images, and uploaded-image persistence/serving.

The local frontend is available at port 8081; this work has not been published to the Vercel static deployment.
