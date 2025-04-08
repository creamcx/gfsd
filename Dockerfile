FROM golang:1.23-alpine AS builder

WORKDIR /app

# Копируем файлы модуля Go для кэширования зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем код проекта
COPY . .

# Собираем приложение
RUN go build -o astro-sarafan ./cmd/bot

FROM alpine:latest

# Добавляем CA-сертификаты для HTTPS-запросов
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Копируем бинарный файл из предыдущего образа
COPY --from=builder /app/astro-sarafan .

# Создаем директорию для конфигурации
RUN mkdir -p /app/config

# Копируем конфигурационный файл
COPY config/config.yaml /app/config/

# Устанавливаем права на запуск
RUN chmod +x /app/astro-sarafan

# Устанавливаем точку входа
ENTRYPOINT ["./astro-sarafan"]

# По умолчанию без параметров
CMD []