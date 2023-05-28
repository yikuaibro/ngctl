
.PHONY: build
build: clean fmt
	go build -o bin/ngctl main.go
	cp bin/ngctl /usr/local/bin/ngctl

.PHONY: clean
clean:
	-rm ./bin/ngctl

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...


ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

CHECK_LINT := $(LOCALBIN)/golangci-lint

.PHONY: lint
lint: check-lint
	GOBIN=$(LOCALBIN) CGO_ENABLED=0 golangci-lint run -v --timeout=5m ./...

.PHONY: check-lint
check-lint: $(CHECK_LINT) ## Download golangci-lint-setup locally if necessary.
$(CHECK_LINT): $(LOCALBIN)
	GOBIN=$(LOCALBIN) CGO_ENABLED=0 go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest