FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . . 
RUN go build -v -o config . 

FROM alpine:latest
WORKDIR /app 
# Create non-root user for security
RUN addgroup -S appuser \
 && adduser -S -G appuser -H -s /sbin/nologin appuser

COPY --from=builder --chown=appuser:appuser /app/config /app/config 
COPY --from=ghcr.io/grpc-ecosystem/grpc-health-probe:v0.4.38 /ko-app/grpc-health-probe /bin/grpc_health_probe

USER appuser

EXPOSE 50051

ENTRYPOINT [ "/app/config" ]

