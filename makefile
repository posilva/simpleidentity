.PHONY: test run deps

run: test 
	go run cmd/accounts/main.go 

test:
	go test -v ./...

deps:
	go mod tidy 
