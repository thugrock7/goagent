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
	find ./instrumentation -type d -print | \
	grep examples/ | \
	xargs -I {} bash -c 'if [ -f "{}/main.go" ] ; then cd {}; go build -o ./build_example main.go ; fi'
	find . -name "build_example" -delete

generate-config: # generates config object for Go
	@echo "Compiling the proto file"
	@# use protoc v3.13 and protoc-gen-go v1.25.0
	@cd config/agent-config; protoc --go_out=paths=source_relative:.. config.proto
	@echo "Generating the loaders"
	@cd config; go run cmd/generator/main.go agent-config/config.proto
	@echo "Done."

.PHONY: fmt
fmt:
	gofmt -w -s ./
