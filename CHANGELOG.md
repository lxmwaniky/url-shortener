# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Integrated **Singleflight** pattern (`golang.org/x/sync/singleflight`) in `CachedURLRepository` to eliminate cache stampedes (thundering herd problem) under high concurrent cache misses.
- Introduced customizable Redis connection pool options (`REDIS_POOL_SIZE`, `REDIS_MIN_IDLE_CONNS`) and strict network timeouts (`REDIS_DIAL_TIMEOUT`, `REDIS_READ_TIMEOUT`, `REDIS_WRITE_TIMEOUT`) to prevent connection leaks and starvation.
- Declared decoupled `RedisPingable` interface inside `internal/web/handlers.go` and implemented a composite, zero-allocation `/health` endpoint checking both Postgres and Redis status.
- Added extensive mock-based unit tests for the composite health checker in `internal/web/handlers_test.go`.
- Added dynamic SRE configurations to `.env` and `.example.env` for memory limits, eviction policies, pooling, and connection timeouts.

### Changed
- Refactored `cmd/api/main.go` to connect to Redis, decorate the URL repository with caching, and wire up `RedisRateLimiter` instances to read and write endpoints.
- Upgraded the configuration loader in `internal/config/config.go` and `.example.env` with type-safe Redis parameters (`REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`, `REDIS_DB`).
- Swapped the concrete `IPRateLimiter` in the `RateLimit` middleware with the polymorphic `Limiter` interface.
- Changed Go module path and all internal project imports from `gitlab.com/lxmwaniky/url-shortener` to `github.com/lxmwaniky/url-shortener` to completely decouple the codebase from GitLab.
- Decoupled the API redirection base URI from hardcoded production checks, loading it dynamically via a `BASE_URL` environment configuration parameter with an automatic local fallback based on active ports.
- Configured production Redis memory constraints (`--maxmemory 256mb`) and cache eviction rules (`--maxmemory-policy allkeys-lru`) dynamically under `docker-compose.yml` to prevent OOM termination.

### Removed
- Removed `.gitlab-ci.yml` and all references to GitLab from the project configuration and remote mappings.

### Added
- Established high-performance URL redirection cache using the Decorator Pattern (`CachedURLRepository` in `internal/repository/cached_url.go`), bypassing PostgreSQL on cache hit and fallback-caching on miss.
- Integrated automated cache pre-warming upon URL creation, enabling immediate sub-millisecond retrieval.
- Implemented Bijective Expire Synchronization to match cache key lifetime dynamically with the exact database `ExpiresAt` value.
- Added type-safe Redis client connection initializer in `internal/db/redis.go`.
- Designed a unified, polymorphic `Limiter` interface in `internal/web/limiter.go` to seamlessly support multiple rate-limiting implementations.
- Developed `RedisRateLimiter` in `internal/web/redis_limiter.go` leveraging batched Redis transaction pipelines (`TxPipeline`) for transaction-safe, single-round-trip distributed fixed-window rate limiting.
- Configured Redis container service and persistent volume storage under `docker-compose.yml`.
- Added mock-based and environment-aware integration tests for URL caching and Redis-backed rate limiting in `internal/repository/cached_url_test.go` and `internal/web/redis_limiter_test.go`.
- Created a comprehensive, system design and architecture-focused `README.md` containing simple API endpoints and quick start instructions.
- Database migration `000002_alter_urls_short_code_limit.up.sql` to dynamically increase `short_code` column limit to `255` characters.
- Core `URL` domain model in `internal/models/url.go`.
- Cryptographic `Feistel` cipher service in `internal/service/feistel.go` to scramble database IDs for non-predictability.
- High-performance `Base62Encoder` service in `internal/service/encoder.go` to translate scrambled IDs to URL-safe strings.
- Exhaustive unit tests for both services in `internal/service/feistel_test.go` and `internal/service/encoder_test.go`.
- Database up and down migration scripts under `internal/db/migrations/000001_create_urls_table.{up,down}.sql` with custom sequencing.
- Migration embedding in Go using `go:embed` in `internal/db/migrations/migrations.go`.
- Fail-fast configuration loader in `internal/config/config.go` with strict validation of database settings and the Feistel seed.
- Integrated `godotenv` with a smart parent-directory recursive search helper in `internal/config/config.go` to automatically load `.env` from anywhere within the repository during tests and server execution.
- Unit test suite for the configuration loader in `internal/config/config_test.go`.
- Database connection pooling wrapper with SRE tuning limits in `internal/db/db.go`.
- Transaction-safe embedded database migration runner in `internal/db/migrations_runner.go` to safely apply embedded migrations on startup/flag.
- Full integration tests verifying connectivity, ping health, and migration application in `internal/db/db_test.go`.
- High-performance `URLRepository` interface in `internal/repository/url.go` separating data storage abstraction from business implementation.
- Concrete repository implementation in `internal/repository/postgres_url.go` featuring single-trip sequence allocation writes and optimized direct integer primary key seeks for redirection.
- Deep integration test suite for URLRepository covering duplicate custom alias detection, sequential ID allocation, and lookups in `internal/repository/postgres_url_test.go`.
- Custom in-memory IP-based Token Bucket rate limiter in `internal/web/limiter.go` to enforce custom write throttling thresholds.
- Modular web middlewares for structured logging (`slog`), panic recovery, and IP rate limiting in `internal/web/middleware.go`.
- HTTP router endpoints for shortening (POST /shorten), 302 redirects (GET /{code}), and connection pool health checks (GET /health) in `internal/web/handlers.go`.
- Unit test suite for HTTP handler routing validation in `internal/web/handlers_test.go`.
- Production-ready application entrypoint in `cmd/api/main.go` featuring graceful server shutdown and an automated asynchronous background database cleaner to daily purge expired links.

### Fixed
- Fixed Go configuration build failure by removing unused `"net"` and `"strings"` package imports from `internal/config/config.go`.
- Secured URL shortening against Server-Side Request Forgery (SSRF) by wiring up the `isPrivateIP` validation block inside the `Shorten` web handler to reject internal/private loopback networks.
- Decoupled `Handlers` database client by replacing the concrete `*sql.DB` type with a mockable `DBConnection` interface.
- Resolved compilation issues in `internal/web/handlers_test.go` and added a unit test case validating private IP address rejection inside `TestShortenHandlerValidation`.
- Fixed leaky environment variable cleanup in `internal/config/config_test.go` by replacing manual restoration checks with robust, explicit `os.LookupEnv` and `os.Unsetenv` state management.
- Resolved premature skip triggers in `internal/db/db_test.go` and `internal/repository/postgres_url_test.go` by invoking `config.Load()` first, enabling the recursive `.env` file loader to resolve configuration parameters before asserting their presence.
- Corrected Go `uint64` database lookup overflow in `GetByShortCode` by verifying decrypted sequence IDs are within the PostgreSQL signed `BIGINT` range (`math.MaxInt64`) before querying, preventing 500 errors and ensuring invalid short codes properly return 404.
- Added detailed structured `slog.Error` logs in the Redirect web handler to capture database retrieval failures on internal server errors.
