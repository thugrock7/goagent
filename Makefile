.DEFAULT_GOAL := test

.PHONY: test
test: test-unit test-docker

.PHONY: test-unit
test-unit:
	@go test -count=1 -v -race -cover ./...

.PHONY: docker-test
test-docker:
	@./tests/docker/test.sh

.PHONY: bench
bench:
	go test -v -run - -bench . -benchmem ./...

.PHONY: lint
lint:
	@echo "Running linters..."
	@golangci-lint run ./... && echo "Done."

.PHONY: deps
deps:
	@go get -v -t -d ./...

.PHONY: ci-deps
deps-ci:
	@go get github.com/golangci/golangci-lint/cmd/golangci-lint

check-examples:
	go build -o ./examples/http_client instrumentation/opentelemetry/net/hyperhttp/examples/client/main.go && rm ./examples/http_client
	go build -o ./examples/http_server instrumentation/opentelemetry/net/hyperhttp/examples/server/main.go && rm ./examples/http_server
	go build -o ./examples/grpc_client instrumentation/opentelemetry/google.golang.org/hypergrpc/examples/client/main.go && rm ./examples/grpc_client
	go build -o ./examples/grpc_server instrumentation/opentelemetry/google.golang.org/hypergrpc/examples/server/main.go && rm ./examples/grpc_server
	go build -o ./examples/http_client instrumentation/opencensus/net/hyperhttp/examples/client/main.go && rm ./examples/http_client
	go build -o ./examples/http_server instrumentation/opencensus/net/hyperhttp/examples/server/main.go && rm ./examples/http_server
	go build -o ./examples/grpc_client instrumentation/opencensus/google.golang.org/hypergrpc/examples/client/main.go && rm ./examples/grpc_client
	go build -o ./examples/grpc_server instrumentation/opencensus/google.golang.org/hypergrpc/examples/server/main.go && rm ./examples/grpc_server

generate-config: # generates config object for Go
	# if agent-config module isn't present we initialize submodules.
	[ -d "./config/agent-config" ] || git submodule update --init --recursive
	@cd config; go run cmd/generator/main.go agent-config/config.proto
