## Exchange Rate Service

HTTP service to convert amounts between fiat and cryptocurrencies, with Prometheus metrics.

### Requirements
- Go 1.23+
- Docker (optional for containerized run)

### Configuration
- Environment:
  - `EXCHANGERATE_API_KEY` (required) — API key for exchangerate.host.

Create a local `.env` (not committed) with:
```
EXCHANGERATE_API_KEY=your_key_here
```

### Run locally
```
go run ./cmd/api
```

API:
- Convert: `GET /convert?from=USD&to=INR&amount=100` (optional `date=YYYY-MM-DD` for fiat historical within 90 days)
- Metrics: `GET /metrics`

### Run with Docker Compose
```
export EXCHANGERATE_API_KEY=your_key_here
docker compose up --build
```

Services:
- App: `http://localhost:8080`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000`

### Tests
```
go test ./...
```

