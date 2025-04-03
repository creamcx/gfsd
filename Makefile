.PHONY: build run clean

build:
	docker build -t astro-sarafan .

run:
	docker run -it \
		-v $(shell pwd)/config/config.yaml:/app/config.yaml \
		astro-sarafan

clean:
	docker rmi astro-sarafan

stop:
	docker stop $$(docker ps -q -f ancestor=astro-sarafan) || true

