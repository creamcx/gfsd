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
	docker-compose run --rm app ./astro-sarafan -migrate -config=/app/config/config.yaml

logs:
	docker-compose logs -f