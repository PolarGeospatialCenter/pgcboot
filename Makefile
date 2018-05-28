.PHONY: test deps docker

test: deps
	AWS_ACCESS_KEY_ID=asdf AWS_SECRET_KEY=asdf AWS_REGION=us-east-2 go test -cover ./pkg/...
	AWS_ACCESS_KEY_ID=asdf AWS_SECRET_KEY=asdf AWS_REGION=us-east-2 go test -cover ./cmd/...

vendor: Gopkg.lock Gopkg.toml
	dep ensure -vendor-only

deps: vendor

docker:
	docker build -f Dockerfile.distroserver -t polargeospatialcenter/distroserver .
