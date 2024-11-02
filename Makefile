NAME = stremthru
DOCKER_ID = muniftanjim

.PHONY: all clean fmt test build run docker-build docker-push docker-run

all: build image-build

clean:
	rm -rf $(NAME)

fmt:
	go fmt ./...

test:
	go test ./...

build: clean
	go build

run:
	go run .

docker-build:
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		-t $(DOCKER_ID)/$(NAME) .

docker-run:
	docker run --rm -it --name $(NAME) \
		-p 8080:8080 \
		$(DOCKER_ID)/$(NAME)

docker-push:
	docker push $(DOCKER_ID)/$(NAME)