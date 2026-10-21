.PHONY: build run test integration-test migrate clean proto deps

PROTO_INCLUDE ?= /usr/local/include

build:
	go build -o bin/life-core .

run:
	go run .

test:
	go test ./...

integration-test:
	go test -tags=integration ./router -run TestTaskListHTTP -v

migrate:
	go run ./cmd/migrate

clean:
	rm -rf bin/

proto:
	protoc proto/*.proto \
		-I . -I $(PROTO_INCLUDE) \
		--go_out=. --go_opt=paths=source_relative \
		--myhttp_out=. --myhttp_opt=paths=source_relative

deps:
	go mod tidy
