FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . . 
RUN go build -v -o orderservice . 

FROM alpine:latest
WORKDIR /app 
# Create non-root user for security
RUN addgroup -S appuser \
 && adduser -S -G appuser -H -s /sbin/nologin appuser

COPY --from=builder --chown=appuser:appuser /app/orderservice /app/orderservice 


USER appuser

EXPOSE 8081

ENTRYPOINT [ "/app/orderservice" ]

