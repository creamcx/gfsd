FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o astro-sarafan ./cmd/bot

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/astro-sarafan .
COPY config/config.yaml .
COPY migrations ./migrations
CMD ["./astro-sarafan"]