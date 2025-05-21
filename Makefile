.PHONY: fieldalign
fieldalign:
	@go tool fieldalignment -fix ./queue/...

.PHONY: tidy
tidy:
	@go mod tidy

.PHONY: check
check:
	@go tool golangci-lint run

.PHONY: fix
fix:
	@go tool golangci-lint run --fix

.PHONY: fmt
fmt:
	@go tool golangci-lint fmt

.PHONY: test
test:
	@go test -covermode=atomic -race -coverprofile=coverage.txt -timeout 5m -json -v ./... 2>&1 | go tool gotestfmt -showteststatus
