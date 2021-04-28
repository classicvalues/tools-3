GO_BUILD_ARGS = -mod=readonly -v -ldflags="-s -w"

.PHONY: all
all: algolia checker packages autoupdate kv process-version functions/check-pkg-updates/check-pkg-updates.zip functions/process-version/process-version.zip functions/kv-pump/kv-pump.zip

.PHONY: algolia
algolia:
	go build $(GO_BUILD_ARGS) -o bin/algolia ./cmd/algolia

.PHONY: checker
checker:
	# go build $(GO_BUILD_ARGS) -o bin/checker ./cmd/checker
	touch bin/checker

.PHONY: packages
packages:
	# go build $(GO_BUILD_ARGS) -o bin/packages ./cmd/packages
	touch bin/packages

.PHONY: autoupdate
autoupdate:
	# go build $(GO_BUILD_ARGS) -o bin/autoupdate ./cmd/autoupdate
	touch bin/autoupdate

.PHONY: kv
kv:
	# go build $(GO_BUILD_ARGS) -o bin/kv ./cmd/kv
	touch bin/kv

.PHONY: process-version
process-version:
	go build $(GO_BUILD_ARGS) -o bin/process-version ./cmd/process-version

.PHONY: schema
schema:
	./bin/packages human > schema_human.json
	./bin/packages non-human > schema_non_human.json

.PHONY: clean
clean:
	rm -rfv bin/*
	rm -rfv functions/*/*.zip

.PHONY: test
test: clean checker
	go test -v ./test/...

.PHONY: lint
lint:
	go get -u golang.org/x/lint/golint
	$(GOPATH)/bin/golint ./...

.PHONY: dev
dev: autoupdate
	docker build -t cdnjs-dev -f ./dev/Dockerfile .
	docker run -it cdnjs-dev

functions/process-version/process-version.zip:
	make -C ./functions/process-version process-version.zip

functions/check-pkg-updates/check-pkg-updates.zip:
	make -C ./functions/check-pkg-updates check-pkg-updates.zip

functions/kv-pump/kv-pump.zip:
	make -C ./functions/kv-pump kv-pump.zip
