# Inference Service
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o inference-service ./cmd/

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app

COPY --from=builder /app/inference-service .
EXPOSE 8084

CMD ["./inference-service"]
