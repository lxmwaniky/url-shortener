# url-shortener

A URL shortening service built in Go. Converts long URLs into short, shareable links with click tracking and redirect analytics.

Built as a learning project covering Go, PostgreSQL, Redis, Docker, and GCP deployment.

## Stack

- **Language:** Go
- **Database:** PostgreSQL (persistent storage)
- **Cache:** Redis (fast redirects)
- **Deployment:** GCP Cloud Run + Cloud SQL + Memorystore
- **CI/CD:** GitLab CI

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/shorten` | Shorten a long URL |
| `GET` | `/:code` | Redirect to original URL |
| `GET` | `/stats/:code` | Get URL stats and click count |
| `GET` | `/health` | Health check |

## Local Development

### Prerequisites

- Go 1.21+
- Docker (for running Postgres locally)
- Redis

## Build Roadmap

### Week 1 — Go + PostgreSQL (local)
- [ ] Project setup and Go HTTP server
- [ ] PostgreSQL connection and schema
- [ ] `POST /shorten` endpoint
- [ ] `GET /:code` redirect endpoint
- [ ] Click tracking + `/stats/:code` + `/health`

### Week 2 — Redis + Docker
- [ ] Redis caching layer for redirects
- [ ] Dockerize the Go app
- [ ] Docker Compose — Go + PostgreSQL + Redis

### Week 3 — GCP Deployment
- [ ] Push image to Artifact Registry
- [ ] Deploy to Cloud Run
- [ ] Connect Cloud SQL and Memorystore
- [ ] Live public URL

### Week 4 — Production Hardening
- [ ] Structured logging (`log/slog`)
- [ ] Metrics and observability
- [ ] Proper error handling
- [ ] GitLab CI/CD pipeline


## License

MIT
