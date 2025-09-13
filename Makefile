# Disable all the default make stuff
MAKEFLAGS += --no-builtin-rules
.SUFFIXES:

## Display a list of the documented make targets
.PHONY: help
help:
	@echo Documented Make targets:
	@perl -e 'undef $$/; while (<>) { while ($$_ =~ /## (.*?)(?:\n# .*)*\n.PHONY:\s+(\S+).*/mg) { printf "\033[36m%-30s\033[0m %s\n", $$2, $$1 } }' $(MAKEFILE_LIST) | sort

.PHONY: .FORCE
.FORCE:

build:
	go build ./cmd/score-cloudrun/

test:
	go vet ./...
	go test ./... -cover -race

test-app: build
	./score-cloudrun --version
	./score-cloudrun init
	cat score.yaml
	./score-cloudrun generate score.yaml
	cat manifests.yaml

build-container:
	docker build -t score-cloudrun:local .

test-container: build-container
	docker run --rm score-cloudrun:local --version
	docker run --rm -v .:/score-cloudrun score-cloudrun:local init
	cat score.yaml
	docker run --rm -v .:/score-cloudrun score-cloudrun:local generate score.yaml
	cat manifests.yaml