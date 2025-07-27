.PHONY: test run deps lint testi testu fmt check cover local-cover setup 

run: check 
	go run cmd/accounts/main.go 

cover:
	go tool cover -func=cover.out ./internal/...

local-cover:
	gocovsh coverage.out

testu:
	go test -cover -timeout 50000ms -v ./internal/... -covermode=count -coverprofile=coverage.out 

testi:
	go test -cover -timeout 50000ms -v --short ./test/... -covermode=count -coverprofile=coverage.out 


deps:
	go mod tidy 

lint: fmt
	golangci-lint run


test: testi testu 

fmt: 
	go fmt ./...

setup:
	go install github.com/orlangure/gocovsh@latest


check: lint test  

ci: check cover
