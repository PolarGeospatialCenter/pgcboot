.PHONY: test deps docker

test: deps
	go test -cover ./pkg/...
	go test -cover ./cmd/...

vendor: Gopkg.lock Gopkg.toml
	dep ensure -vendor-only

deps: vendor

docker:
	docker build -f Dockerfile.distroserver -t polargeospatialcenter/distroserver .
