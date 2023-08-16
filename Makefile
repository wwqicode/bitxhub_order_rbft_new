
SHELL := /bin/bash
CURRENT_PATH = $(shell pwd)
GO  = GO111MODULE=on go

help: Makefile
	@echo "Choose a command run:"
	@sed -n 's/^##//p' $< | column -t -s ':' | sed -e 's/^/ /'

## make rbft: build plugin (make plugin type= <rbft>)
rbft:
	@mkdir -p build
	$(GO) build --buildmode=plugin -o build/rbft.so rbft/*.go

.PHONY: rbft