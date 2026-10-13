# Kliktaut

URL shortener with click analytics, built as a backend portfolio project. Create a
short link, share it, and see who clicked it: device, browser, OS, referrer, and
which links get traffic.

Go 1.27, Fiber v3, PostgreSQL 17, Redis 8.

## What it does

- Short links, either random (7-char base62) or with a custom alias
- 302 redirects with click tracking that does not block the response
- Analytics per link: total clicks, unique visitors, device/browser breakdown, top referrers, 7-day series
- Overview stats across all links
- Soft delete and link expiration
- API key auth, Redis rate limiting, and a Redis cache on the redirect path
- Interactive API docs (Scalar) generated from the OpenAPI spec

## Requirements

- Go 1.27+
- Docker and Docker Compose

## Quick start

```bash
git clone <your-repo-url> kliktaut-api
cd kliktaut-api

# 1. environment
cp .env.example .env
# edit .env: set a real API_KEY (the default is a placeholder)

# 2. databases (ports 5433 and 6381, see docker-compose.yml)
docker compose up -d

# 3. schema
make migrate

# 4. run
make run
```

The API listens on `http://localhost:3000`. Health check:

```bash
curl localhost:3000/health
# {"database":"ok","redis":"ok","status":"healthy","uptime":"3s"}
```

Optional: `make seed` fills the database with demo links and clicks so the
analytics pages have data. It is safe to run more than once.

## API documentation

Open http://localhost:3000/docs for the Scalar UI. You can send requests from
there: click **Auth**, paste your `API_KEY`, and try any endpoint. The raw spec
lives at `/openapi.yaml`.

## Using the API

Every endpoint under `/api/v1` needs the `X-API-Key` header. Redirects and
`/health` are public.

```bash
KEY=$(grep '^API_KEY=' .env | cut -d= -f2)

# create a link with a custom alias
curl -X POST localhost:3000/api/v1/links \
  -H "X-API-Key: $KEY" -H 'Content-Type: application/json' \
  -d '{"original_url":"https://github.com/you/repo","custom_alias":"my-repo","title":"My repo"}'

# open http://localhost:3000/my-repo in a browser, then check analytics
curl -H "X-API-Key: $KEY" "localhost:3000/api/v1/links?search=my-repo"
curl -H "X-API-Key: $KEY" localhost:3000/api/v1/stats
```

### Endpoints

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| GET | `/{shortCode}` | no | Redirect and record the click |
| GET | `/health` | no | Health check (pings Postgres and Redis) |
| GET | `/docs` | no | Scalar API reference |
| POST | `/api/v1/links` | yes | Create a link |
| GET | `/api/v1/links` | yes | List links (`page`, `per_page`, `search`) |
| GET | `/api/v1/links/{id}` | yes | Link detail |
| PATCH | `/api/v1/links/{id}` | yes | Update `original_url`, `title`, `expires_at`, `is_active` |
| DELETE | `/api/v1/links/{id}` | yes | Soft delete (sets `is_active=false`) |
| GET | `/api/v1/links/{id}/analytics` | yes | Analytics summary |
| GET | `/api/v1/links/{id}/clicks` | yes | Raw click rows |
| GET | `/api/v1/stats` | yes | Overview statistics |

### Error format

```json
{ "success": false, "error": { "code": "LINK_NOT_FOUND", "message": "Link not found" } }
```

| Status | Code | When |
|--------|------|------|
| 400 | `INVALID_REQUEST` | Bad body, invalid URL, malformed ID |
| 401 | `UNAUTHORIZED` | Missing or wrong API key |
| 404 | `LINK_NOT_FOUND` | Unknown short code or link ID |
| 409 | `ALIAS_TAKEN` | Custom alias already used or reserved |
| 410 | `LINK_UNAVAILABLE` | Link deactivated or expired |
| 429 | `RATE_LIMITED` | Rate limit exceeded |
| 500 | `INTERNAL_ERROR` | Unexpected server error |

Rate-limited responses carry `X-RateLimit-Limit`, `X-RateLimit-Remaining`, and
`X-RateLimit-Reset` (epoch seconds).

## Performance

The redirect path serves from Redis and records clicks in a background
goroutine, so the response never waits on the database.

Load test (`make loadtest`, 5000 requests, 50 concurrent, cache hit):

| Metric | Value |
|--------|-------|
| p50 | 2.24 ms |
| p95 | 4.95 ms |
| p99 | 6.61 ms |
| Throughput | ~19,000 req/s |

Definition of done was p50 < 50 ms on a cache hit. Measured locally on the
development machine, without counting the async click write.

Run it yourself:

```bash
make run                  # in one terminal
make seed
make loadtest             # in another
```

The load test needs a high redirect rate limit, since the default is 300/min per
IP. Start the server with a raised limit for the run:

```bash
RATE_LIMIT_REDIRECT_MAX=1000000 make run
```

## Architecture

```
Client
  |
Fiber (middleware: logger, CORS, API key, rate limit)
  |
Handlers -> Services -> Repositories
  |                         |
  |                      Postgres (links, clicks)
  |
Redis (redirect cache, rate limit counters)
```

Redirect flow:

1. `GET /{shortCode}`
2. Look up `link:{shortCode}` in Redis. On a hit, skip the database.
3. On a miss, query Postgres, then cache the result. Links that are inactive or
   expired are never cached.
4. Record the click in a goroutine (insert row plus atomic counter update).
5. Return 302 with the target URL.

`PATCH` and `DELETE` delete the cache key so a changed or deactivated link stops
redirecting right away instead of waiting for the TTL.

## Project layout

```
cmd/api/            entry point, route wiring, migrations on boot (container)
cmd/seed/           demo data seeder
internal/cache/     Redis link cache
internal/config/    env loading and validation
internal/database/  Postgres and Redis connections
internal/docs/      embedded OpenAPI spec
internal/dto/       request and response shapes
internal/handlers/  HTTP handlers
internal/middleware/ API key, rate limiting
internal/models/    GORM models
internal/repositories/ data access
internal/services/  business logic (shortener, redirect rules, analytics)
internal/utils/     response envelope, validation, user-agent parsing
migrations/         SQL up/down files
scripts/loadtest/   redirect load test
```

## Deployment

The Dockerfile builds a static binary and runs it on distroless. Migrations run
from the app binary on boot when `RUN_MIGRATIONS=1` (set in the image), so a
fresh deploy gets the schema without a separate step.

```bash
docker build -t kliktaut .
docker run -p 3000:3000 \
  -e DB_HOST=... -e DB_PASSWORD=... -e DB_NAME=kliktaut \
  -e REDIS_HOST=... -e API_KEY=... \
  kliktaut
```

### Render

`render.yaml` defines a web service plus Postgres and Redis. Point Render at your
repository, set `API_KEY` as a secret, and deploy.

Note on the free tier: the web service spins down after about 15 minutes idle,
so the first request can take 30 to 60 seconds. That cold start shows up in
latency numbers, which is worth mentioning if you share the demo link.

### Production checklist

- Set a strong `API_KEY` (32+ random characters)
- Set `APP_ENV=production`
- Point `BASE_URL` at the public URL
- Set `CORS_ORIGINS` to your allowed origins (comma-separated, no wildcard)
- Enable SSL on the database connection
- Do not commit `.env`

## Known limits

- `unique_visitors` counts distinct IP addresses, so visitors behind a shared
  NAT count as one.
- Rate limiting uses a fixed window, which allows a burst up to twice the limit
  around a window boundary.
- If Redis is down, rate limiting fails open and redirects fall back to Postgres.
  Availability was chosen over strictness at this scale.
- `clicks_by_device` includes the `bot` category while `total_clicks` excludes
  it. The two numbers differ on purpose.

## Development

```bash
make run            # run the API
make migrate        # apply migrations
make migrate-down   # roll back one migration
make seed           # insert demo data
make loadtest       # redirect load test
```

Tests and checks:

```bash
go test ./...
go vet ./...
go build ./...
```

CI runs vet, golangci-lint, `go test -race`, and a build on every push.

## License

MIT
