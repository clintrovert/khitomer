.PHONY: proto
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/*.proto

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: build
build:
	go build ./cmd/leader
	go build ./cmd/worker

