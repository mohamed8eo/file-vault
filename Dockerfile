# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o file-vault ./cmd/api/main.go

# Production stage
FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/file-vault .
COPY --from=builder /app/.env .

EXPOSE 8080

ENTRYPOINT ["./file-vault"]
