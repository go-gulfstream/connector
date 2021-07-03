TAG_NAME=latest

.PHONY: docker-image
docker-image:
	@docker build -t gulstream/gs-connector:$(TAG_NAME) -f gs-connector.dockerfile .
	@docker push gulstream/gs-connector:$(TAG_NAME)
	@docker image prune --filter label=stage=builder
