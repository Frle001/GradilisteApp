# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-s -w" \
    -o api .

# Runtime stage
FROM alpine:3.21

RUN apk --no-cache add ca-certificates tzdata && \
    adduser -D -u 1001 appuser

WORKDIR /app

COPY --from=builder /build/api .

RUN chown appuser:appuser /app/api

USER appuser

EXPOSE 8080

CMD ["./api"]
