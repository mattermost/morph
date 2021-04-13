all: test


.PHONY: test
test:
	go clean -testcache
	make test-drivers
	make test-rest

.PHONY: test-rest
test-rest:
	go clean -testcache
	go test -race -v --tags=!drivers,sources ./...

.PHONY: test-drivers
test-drivers:
	go clean -testcache
	go test -race -v --tags=drivers,!sources ./...

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
