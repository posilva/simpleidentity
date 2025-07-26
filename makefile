.PHONY: test run deps lint testi testu fmt check

run: check 
	go run cmd/accounts/main.go 

testu:
	go test -timeout 50000ms -v ./internal/... -covermode=count -coverprofile=cover.out && go tool cover -func=cover.out

testi:
	go test -timeout 50000ms -v --short ./test/...

deps:
	go mod tidy 

lint: fmt
	golangci-lint run


test: testi testu  

fmt: 
	go fmt ./...


check: fmt lint testi testu 
	:
