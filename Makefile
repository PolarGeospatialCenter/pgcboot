.PHONY: test docker

test: 
	AWS_ACCESS_KEY_ID=asdf AWS_SECRET_KEY=asdf AWS_REGION=us-east-2 go test -cover ./pkg/...
	AWS_ACCESS_KEY_ID=asdf AWS_SECRET_KEY=asdf AWS_REGION=us-east-2 go test -cover ./cmd/...

docker:
	docker build -f Dockerfile.distroserver -t polargeospatialcenter/distroserver .
