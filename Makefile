.DEFAULT_GOAL := help

help:   ## Print help text.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

lint:   ## Check code using various linters and static checkers.
	gofmt -d .
	go vet -v . || exit 1
	golint -set_exit_status . || exit 1
	errcheck -ignoretests || exit 1
	misspell -locale uk .

install: ## Install or update development dependencies.
	@go get -u github.com/stretchr/testify/assert
	@go get -u github.com/kisielk/errcheck
	@go get -u github.com/client9/misspell/cmd/misspell
	@go get -u github.com/golang/lint/golint


test:   ## Run unit tests and print test coverage.
	@touch .coverage.out
	@go test -coverprofile .coverage.out && go tool cover -func=.coverage.out

.PHONY: help lint test
