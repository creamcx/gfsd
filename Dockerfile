FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o astro-sarafan ./cmd/bot

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

COPY --from=builder /app/astro-sarafan .
COPY config/config.yaml .

# Создаем директорию для PDF
RUN mkdir -p ./data/pdf

EXPOSE 8080

CMD ["./astro-sarafan"]