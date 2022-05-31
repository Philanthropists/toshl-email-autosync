git-commit := $(shell git rev-list -1 HEAD)
ldflags := "-s -w -X main.GitCommit=${git-commit}"
gcflags := -G=3
flags := -ldflags=${ldflags} -gcflags=${gcflags}

.PHONY: build
build: bin clean vendor fmt credentials
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

registry.json:
	@[ -z "${TOSHL_SECRETS_LOCATION}" ] && read -p "Where are the secrets?: " TOSHL_SECRETS_LOCATION; \
	read -p "What registry should I use?: " registry; \
	cp "$${TOSHL_SECRETS_LOCATION}/$${registry}_registry.json" registry.json

.PHONY: upload-docker
upload-docker: registry.json
	awsprofile="$$(cat registry.json | jq -r .awsprofile)"; \
	registry="$$(cat registry.json | jq -r .registry)"; \
	aws --profile $${awsprofile} ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin $${registry}; \
	docker tag toshl-sync:latest $${registry}/private-ecr:latest; \
	docker push $${registry}/private-ecr:latest; \
	rm -f registry.json

.PHONY: credentials
credentials: credentials.json
credentials.json:
	@[ -z "${TOSHL_SECRETS_LOCATION}" ] && read -p "Where are the secrets?: " TOSHL_SECRETS_LOCATION; \
	read -p "What credentials should I use?: " cred_file; \
	cp "$${TOSHL_SECRETS_LOCATION}/$${cred_file}_creds.json" credentials.json

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
	rm -f registry.json

.PHONY: clean
clean:
	rm -rf bin/*
