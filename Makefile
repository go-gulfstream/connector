.PHONY: docker-image
docker-image:
	@docker build -t gulstream/gs-connector:latest -f gs-connector.dockerfile .
	@docker push gulstream/gs-connector:latest
	@docker image prune --filter label=stage=builder
