# Currency Exchange Rate Service

This is a robust, production-ready backend service, written in Go, that provides real-time and historical currency exchange rates for both fiat and major cryptocurrencies. It is designed for high availability, performance, and observability.

## Core Features

-   **✔ Real-time & Historical Rates:** Fetches the latest exchange rates and supports historical lookups for fiat currencies up to 90 days in the past.
-   **✔ Fiat & Crypto Support:** Handles conversions between a fixed set of fiat currencies (USD, EUR, GBP, INR, JPY) and major cryptocurrencies (BTC, ETH, USDT).
-   **✔ High-Performance Caching:** Caches the latest rates in-memory for fast lookups, with a resilient background worker that refreshes rates every hour.
-   **✔ Concurrent & Resilient:** The rate-refreshing mechanism fetches fiat and crypto data concurrently. A failure from one provider does not prevent the other from succeeding, ensuring a partially fresh cache is always preferred over a completely stale one.
-   **✔ Robust Error Handling:** The API distinguishes between client errors (e.g., invalid input, `4xx` status codes) and true server failures (`5xx`), providing clear and accurate feedback.

## Architectural Highlights

-   **Clean Architecture:** The project follows Clean Architecture principles, with a clear separation of concerns into `domain`, `repository`, `provider`, `service`, and `api` layers. This makes the codebase modular, testable, and easy to maintain.
-   **Graceful Shutdown:** The service implements a graceful shutdown mechanism. It listens for OS signals (`SIGINT`, `SIGTERM`), stops accepting new requests, waits for in-flight requests to complete, and cleanly stops all background processes before exiting.
-   **Structured Logging:** All logs are produced in a structured `JSON` format using the standard `log/slog` library, complete with log levels and key-value context. This is essential for effective monitoring and debugging in a production environment.
-   **Full Observability Stack:** Comes with a pre-configured Docker Compose setup to run the application alongside **Prometheus** for metrics collection and **Grafana** for visualization.
-   **Dependency Injection:** Dependencies are explicitly injected throughout the application (e.g., passing providers and repositories into services), which enables strong decoupling and thorough unit testing with mocks.

## Getting Started

### Prerequisites

-   **Docker** and **Docker Compose v2** (which uses the `docker compose` command).
-   A free API key from [exchangerate.host](https://exchangerate.host/).

### Running with Docker Compose (Recommended)

This is the easiest and most complete way to run the project, as it includes the full observability stack.

1.  **Clone the repository:**
    ```sh
    git clone https://github.com/Devashish08/ExchangeRateService.git
    cd ExchangeRateService
    ```

2.  **Create the environment file:**
    Create a file named `.env` in the root of the project. This file is ignored by Git and should contain your API key.
    ```
    # In .env file
    EXCHANGERATE_API_KEY=YOUR_API_KEY_HERE
    ```

3.  **Run the stack:**
    ```sh
    docker compose up --build
    ```

The application will now be running and accessible.

-   **API Service:** `http://localhost:8080`
-   **Prometheus:** `http://localhost:9090`
-   **Grafana:** `http://localhost:3000` (login: `admin`/`admin`)

### Running Locally with Go

1.  **Prerequisites:** Go 1.21+ installed.
2.  Follow steps 1 and 2 from the Docker Compose setup to create the `.env` file.
3.  **Load the environment variable:**
    ```sh
    export $(cat .env | xargs)
    ```
4.  **Install dependencies and run:**
    ```sh
    go mod tidy
    go run ./cmd/api
    ```

## API Endpoint

The service provides one primary endpoint for conversions.

#### `GET /convert`

Performs a currency conversion.

**Query Parameters:**

-   `from` (required): The 3-letter currency code to convert from (e.g., `USD`).
-   `to` (required): The 3-letter currency code to convert to (e.g., `INR`).
-   `amount` (required): The numerical amount to convert.
-   `date` (optional): A date in `YYYY-MM-DD` format for a historical conversion. If omitted, the latest cached rate is used.

### Example API Usage (`curl`)

We use `curl -i` to include the HTTP status code in the output.

**1. Successful Latest Fiat Conversion (200 OK)**
```sh
curl -i "http://localhost:8080/convert?from=USD&to=EUR&amount=100"
```

**2. Successful Latest Crypto Conversion (200 OK)**
```sh
curl -i "http://localhost:8080/convert?from=BTC&to=USD&amount=0.5"
```

**3. Client Error: Invalid Date Format (400 Bad Request)**
```sh
curl -i "http://localhost:8080/convert?from=USD&to=EUR&amount=100&date=2020-13-01"
```

**4. Client Error: Historical Date Out of Range (400 Bad Request)**
```sh
curl -i "http://localhost:8080/convert?from=GBP&to=JPY&amount=500&date=2020-01-01"
```

**5. Client Error: Rate Not Found in Cache (404 Not Found)**
*(This is unlikely to happen after startup but shows the error handling)*
```sh
# (If a rate was missing from the cache for some reason)
# HTTP/1.1 404 Not Found
# {"error":"rate not found for FROM to TO"}
```

## Observability: Prometheus & Grafana

The service exposes detailed metrics for monitoring.

1.  **Access Prometheus:** Navigate to `http://localhost:9090`. You can run queries directly, such as:
    -   `http_requests_total` - See request counts broken down by path, method, and status code.
    -   `provider_requests_total` - See the success/failure rate of calls to external APIs.

2.  **Using Grafana:**
    -   Navigate to `http://localhost:3000` and log in (`admin`/`admin`).
    -   **Add Data Source:**
        -   Go to Configuration (gear icon) > Data Sources > Add data source.
        -   Select **Prometheus**.
        -   Set the **Prometheus server URL** to `http://prometheus:9090`.
        -   Click **Save & Test**.
    -   **Create a Panel:**
        -   Go to Dashboards (grid icon) > New dashboard > Add a new panel.
        -   In the query builder, select your Prometheus data source.
        -   In the "Metrics browser" field, enter `http_requests_total` and run the query to see your data.

## Project Structure

The project follows Clean Architecture principles to ensure separation of concerns.

```
/
├── cmd/api/                  # Main application entrypoint and graceful shutdown logic.
├── internal/
│   ├── api/                  # HTTP handlers, routing, and middleware.
│   ├── domain/               # Core data structures (e.g., Currency, ExchangeRate).
│   ├── metrics/              # Prometheus metrics definitions.
│   ├── provider/             # Clients for third-party APIs (exchangerate.host, coingecko).
│   ├── repository/           # Data storage layer (in-memory cache).
│   └── service/              # Core business logic and orchestration.
├── .env                      # Local environment variables (Git-ignored).
├── docker-compose.yml        # Defines the application, Prometheus, and Grafana services.
├── Dockerfile                # Secure, multi-stage Docker build for the application.
└── README.md                 # This file.
```