# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- Changed Go module path and all internal project imports from `gitlab.com/lxmwaniky/url-shortener` to `github.com/lxmwaniky/url-shortener` to completely decouple the codebase from GitLab.
- Decoupled the API redirection base URI from hardcoded production checks, loading it dynamically via a `BASE_URL` environment configuration parameter with an automatic local fallback based on active ports.

### Removed
- Removed `.gitlab-ci.yml` and all references to GitLab from the project configuration and remote mappings.
- Removed all inline, block, and documentation comments from Go source files (`.go`) across the entire repository to enforce a strictly self-documenting code design.

### Added
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
