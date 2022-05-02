git-commit := $(shell git rev-list -1 HEAD)
ldflags := "-s -w -X main.GitCommit=${git-commit}"
gcflags := -G=3
flags := -ldflags=${ldflags} -gcflags=${gcflags}

.PHONY: build
build: bin vendor fmt credentials
	go build ${flags} -o bin cmd/run/run.go
	cp credentials.json bin/

.PHONY: build-for-lambda
build-for-lambda: bin clean vendor fmt credentials
	GOOS=linux \
	GOARCH=amd64 \
	CGO_ENABLED=0 \
		 go build ${flags} -o bin/main cmd/aws-lambda/main.go
	cp credentials.json bin/

.PHONY: build-docker
build-docker: docker-lint clean fmt credentials
	docker buildx build \
		--platform linux/amd64 \
		--build-arg COMMIT=${git-commit} \
		-t toshl-sync .

.PHONY: credentials
credentials: credentials.json
credentials.json:
	@[ -z "${TOSHL_SECRETS_LOCATION}" ] && read -p "Where are the secrets?: " TOSHL_SECRETS_LOCATION; \
	read -p "What credentials should I use?: " cred_file; \
	cp "$${TOSHL_SECRETS_LOCATION}/$${cred_file}.json" credentials.json

bin:
	mkdir -p bin

.PHONY: docker-lint
docker-lint:
	docker run --rm -i ghcr.io/hadolint/hadolint < Dockerfile

.PHONY: fmt
fmt: staticcheck
	go fmt ./...

.PHONY: staticcheck
staticcheck: vet
	go install honnef.co/go/tools/cmd/staticcheck@latest
	$(GOPATH)/bin/staticcheck ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: vendor
vendor: tidy
	go mod vendor

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: clean-all
clean-all: clean
	rm -f credentials.json

.PHONY: clean
clean:
	rm -rf bin/*
