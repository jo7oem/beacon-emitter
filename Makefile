DEV_BIN:=dev_tools/bin
MAKEFILE_DIR := $(shell cd $(dir $(lastword $(MAKEFILE_LIST)))&&pwd )
ABS_DEV_BIN := $(MAKEFILE_DIR)/$(DEV_BIN)

.PHONY: build
build:
	go build -o hatsukari ./...

.PHONY: test
test:
	go test ./...

.PHONY: fmt
fmt: $(DEV_BIN)/golangci-lint  $(DEV_BIN)/wsl
	$(DEV_BIN)/wsl --fix ./... >/dev/null 2>&1 || true
	$(DEV_BIN)/golangci-lint run --fix --config=.golangci.yml

.PHONY: lint
lint: $(DEV_BIN)/golangci-lint
	$(DEV_BIN)/golangci-lint run --config=.golangci.yml

.PHONY: setup
setup: $(DEV_BIN)/golangci-lint

$(DEV_BIN)/golangci-lint:
	mkdir -p $(@D)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(@D) v1.55.2


$(DEV_BIN)/wsl:
	mkdir -p $(@D)
	GOBIN=$(ABS_DEV_BIN) go install github.com/bombsimon/wsl/v4/cmd...@master

.PHONY: clean
clean:
	rm -rf dev_tools/bin


