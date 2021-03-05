all: test


test:
	go clean -testcache
	go test -race -v ./...


update-dependencies:
	go get -u ./...
	go mod vendor
	go mod tidy


vendor:
	go mod vendor
	go mod tidy


check:
	go fmt ./...
