# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: geth suave android ios evm all test clean

GOBIN = ./build/bin
GO ?= latest
GORUN = env GO111MODULE=on go run

geth:
	$(GORUN) build/ci.go install ./cmd/geth
	@echo "Done building."
	@echo "Run \"$(GOBIN)/geth\" to launch geth."

suave:
	$(GORUN) build/ci.go install ./cmd/geth
	cd $(GOBIN) && rm -f suave && ln -s geth suave
	@echo "Done building."
	@echo "Run \"$(GOBIN)/suave\" to launch SUAVE."

	# Move a copy of the binary to GOPATH/bin
	cp $(GOBIN)/suave $(GOPATH)/bin

all:
	$(GORUN) build/ci.go install

test: all
	$(GORUN) build/ci.go test

lint: ## Run linters.
	$(GORUN) build/ci.go lint

clean:
	env GO111MODULE=on go clean -cache
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go install golang.org/x/tools/cmd/stringer@latest
	env GOBIN= go install github.com/fjl/gencodec@latest
	env GOBIN= go install github.com/golang/protobuf/protoc-gen-go@latest
	env GOBIN= go install ./cmd/abigen
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

suavedevtools:
	./suave/scripts/contracts.sh build
	go run ./suave/gen/main.go -write

devnet-up:
	docker-compose -f ./suave/devenv/docker-compose.yml up -d --build

devnet-down:
	docker-compose -f ./suave/devenv/docker-compose.yml down
