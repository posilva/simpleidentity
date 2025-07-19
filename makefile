.PHONY: test run 

run: test 
	go run cmd/accounts/main.go 

test:
	go test -v ./...
