# Build stage: compile the app and the migrate CLI
FROM golang:1.27-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/app ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -tags postgres -ldflags="-s -w" -o /out/migrate github.com/golang-migrate/migrate/v4/cmd/migrate

# Runtime stage: distroless, no shell, no package manager.
# Migrations run from inside the app binary before listening (see cmd/api).
FROM gcr.io/distroless/static-debian12

COPY --from=builder /out/app /app
COPY --from=builder /out/migrate /migrate
COPY --from=builder /build/migrations /migrations

EXPOSE 3000

ENV RUN_MIGRATIONS=1
ENTRYPOINT ["/app"]
