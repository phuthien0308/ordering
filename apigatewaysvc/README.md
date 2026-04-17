# API Gateway Service (`apigatewaysvc`)

The API Gateway is the single public entry point for all client traffic. It translates HTTP/REST/JSON from external clients into gRPC calls to internal backend services, enforcing policies (auth, rate limiting, timeouts) at the boundary so individual services don't have to.

---

## Feature Roadmap

### 🔴 Phase 1 — Core (Get it Working)

| Feature | Description |
|---|---|
| **Request Routing** | Maps incoming HTTP paths to the correct upstream gRPC service and method |
| **Protocol Translation** | Parses incoming JSON body → builds proto request → calls gRPC → serializes proto response back to JSON |
| **Error Handling** | Translates gRPC status codes to HTTP status codes and wraps all responses in a consistent JSON envelope |

gRPC → HTTP status code mapping:
```
codes.OK               → 200
codes.NotFound         → 404
codes.InvalidArgument  → 400
codes.AlreadyExists    → 409
codes.Unauthenticated  → 401
codes.PermissionDenied → 403
codes.Internal         → 500
codes.Unavailable      → 503
```

---

### 🟡 Phase 2 — Production Safe

| Feature | Description |
|---|---|
| **Authentication (JWT)** | Validates JWT tokens on every request before forwarding. Single enforcement point — services trust all traffic arriving from the gateway |
| **Rate Limiting** | Limits requests per IP or per user ID (e.g. 100 req/min per IP, 1000 req/min per authenticated user) |
| **Request/Response Logging** | Logs every request with method, path, duration, status code, and user ID |
| **Timeouts & Deadline Propagation** | Sets a maximum request duration. Returns `504 Gateway Timeout` if upstream does not respond in time instead of hanging the client indefinitely |
| **CORS** | Adds `Access-Control-Allow-*` headers for browser-based clients making cross-origin requests |

---

### 🟢 Phase 3 — Resilience

| Feature | Description |
|---|---|
| **Circuit Breaker** | If a downstream service fails repeatedly, the circuit opens and returns `503` immediately instead of forwarding failing requests. Resets after a cooldown period |
| **Load Balancing** | Distributes requests across multiple instances of a service using round-robin or least-connections |
| **Request Validation** | Validates request bodies before they reach the service. Returns `400 Bad Request` early with a descriptive message |

---

### 🔵 Phase 4 — Observability

| Feature | Description |
|---|---|
| **Metrics (Prometheus)** | Exposes request counts, error rates, and latency histograms. Scraped by Prometheus for dashboards and alerting |
| **Distributed Tracing** | Propagates OpenTelemetry trace headers so a single request can be traced across `gateway → service → database` |
| **Response Caching** | Caches read-heavy responses (e.g. product search) with a TTL to reduce load on downstream services |

---

## TLS Strategy

TLS works at two boundaries in this architecture and they serve different purposes:

### 1. Client → Gateway (TLS Termination)
The gateway is the public-facing endpoint and **terminates TLS from external clients**. All traffic from browsers and mobile apps arrives as HTTPS. The gateway decrypts it, processes the request, and forwards it internally.

```
Internet Client
  ─── HTTPS (TLS 1.3) ──→  apigatewaysvc (:443)
                                    │
                            TLS terminated here
                                    │
                            internal network
```

In Go, this is done with:
```go
server := &http.Server{
    TLSConfig: &tls.Config{
        MinVersion: tls.VersionTLS13,
    },
}
server.ListenAndServeTLS("server.crt", "server.key")
```

### 2. Gateway → Services (mTLS — Mutual TLS)
Internal traffic between the gateway and backend gRPC services should use **mTLS (mutual TLS)**. This means both sides present certificates to authenticate each other — not just the server proving its identity to the client. This prevents a rogue process inside your network from impersonating a legitimate service.

```
apigatewaysvc ──── mTLS ────→ productservice (:5000)
apigatewaysvc ──── mTLS ────→ accountservice (:50052)
```

This is already partially in place — `accountservice` loads `server.crt` and `server.key`. The gateway needs a client certificate to connect to it:
```go
creds, err := credentials.NewClientTLSFromFile("ca.crt", "")
conn, err := grpc.NewClient("localhost:50052", grpc.WithTransportCredentials(creds))
```

> **For local development**, using `insecure.NewCredentials()` is acceptable (as `productservice` currently does). Introduce mTLS before deploying to any shared or production environment.

---

## Routing Table

| Method | Path | Service | RPC |
|---|---|---|---|
| `POST` | `/v1/products` | productservice | `CreateProduct` |
| `PUT` | `/v1/products/:sku` | productservice | `UpdateProduct` |
| `DELETE` | `/v1/products/:sku` | productservice | `DeleteProduct` |
| `GET` | `/v1/products/search` | productservice | `SearchProducts` |
| `POST` | `/v1/accounts` | accountservice | `CreateAccount` |
| `POST` | `/v1/orders` | orderservice | `PlaceOrder` |

---

## Architecture

```
Client (HTTP/JSON)
        │
        ▼
┌──────────────────────────┐
│      apigatewaysvc       │
│                          │
│  ┌──────────────────┐    │
│  │    Middleware     │    │ ← Auth, Rate Limit, Logging, CORS, Timeout
│  └────────┬─────────┘    │
│           │              │
│  ┌────────▼─────────┐    │
│  │     Router        │    │ ← maps path → handler
│  └────────┬─────────┘    │
│           │              │
│  ┌────────▼─────────┐    │
│  │  gRPC Clients    │    │ ← one per downstream service
│  └──────────────────┘    │
└──────────┬───────────────┘
           │  gRPC (mTLS in prod)
    ┌──────┴───────┐
    ▼              ▼
productservice  accountservice  orderservice ...
```
