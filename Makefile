.DEFAULT_GOAL := build

.PHONY: fmt vet staticcheck build
fmt:
	go fmt ./...

vet: fmt
	go vet ./...

staticcheck: vet
	staticcheck ./...

build: staticcheck
	GOARCH=amd64 GOOS=linux go build 

# consider staticcheck, govulncheck, etc