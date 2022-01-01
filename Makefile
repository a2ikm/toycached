.PHONY: test
test:
	go test ./...

.PHONY: build
build:
	go build

.PHONE: start
start: build
	./toycached
