# Experiment Service
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o experiment-service ./cmd/

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app

COPY --from=builder /app/experiment-service .
EXPOSE 8085

CMD ["./experiment-service"]
