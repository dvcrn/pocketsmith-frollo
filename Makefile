.PHONY: docker-build

docker-build:
	docker buildx build --platform linux/amd64,linux/arm64 -t dvcrn/pocketsmith-frollo:latest . --push
