all: test


.PHONY: test
test:
	go clean -testcache
	go test -race -v ./...


.PHONY: update-dependencies
update-dependencies:
	go get -u ./...
	go mod vendor
	go mod tidy


.PHONY: vendor
vendor:
	go mod vendor
	go mod tidy


.PHONY: check
check:
	go fmt ./...
