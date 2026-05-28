# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

COPY go.mod ./
COPY go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api main.go database.go middleware.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /build/api .

EXPOSE 8080

CMD ["./api"]
