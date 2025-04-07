.PHONY: build run clean stop db-up

build:
	docker build -t astro-sarafan .

run:
	docker-compose up 

clean:
	docker-compose down
	docker rmi astro-sarafan

stop:
	docker-compose down

db-up:
	docker-compose up -d postgres

db-migrate:
	docker-compose run --rm app bash -c "cd /app && go run cmd/migrate/main.go"

logs:
	docker-compose logs -f

prot:
	protoc -I. -I./api/proto \
      --go_out=. --go_opt=paths=source_relative \
      --go-grpc_out=. --go-grpc_opt=paths=source_relative \
      --grpc-gateway_out=. --grpc-gateway_opt=paths=source_relative \
      internal/pkg/gen_v1/gen.proto