NAME = stremthru
DOCKER_ID = muniftanjim

.PHONY: all clean fmt test build run docker-build docker-push docker-run

all: build docker-build

clean:
	rm -rf $(NAME)

fmt:
	go fmt ./...

test:
	STREMTHRU_ENV=test go test -v ./...

build: clean
	go build --tags "fts5"

run:
	go run --tags "fts5" .

docker-build:
	docker buildx build \
		--file Dockerfile \
		--platform linux/amd64,linux/arm64 \
		-t $(DOCKER_ID)/$(NAME):latest .

docker-run:
	docker run --rm -it --name $(NAME) \
		-p 7860:7860 \
		$(DOCKER_ID)/$(NAME)

docker-push:
	docker push $(DOCKER_ID)/$(NAME)
