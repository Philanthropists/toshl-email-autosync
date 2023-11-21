git-commit := $(shell git rev-list -1 HEAD)
ldflags := "-s -w"
gcflags := ""
flags := -ldflags=${ldflags} -gcflags=${gcflags}

docker-build-push: docker-build docker-push

.PHONY: build
build: bin clean fmt credentials test
	go build -o bin/run cmd/cli/run.go
	go build -o bin/mail cmd/mail/run.go
	go build -o bin/dynamodb cmd/dynamodb/run.go
	go build -o bin/twilio cmd/twilio/run.go
	go build -o bin/archive cmd/archive/run.go
	go build -o bin/toshl cmd/toshl/toshl.go
	cp credentials.json bin/

.PHONY: build-for-lambda
build-for-lambda: bin clean fmt credentials test
	GOOS=linux \
	GOARCH=amd64 \
	CGO_ENABLED=0 \
		 go build ${flags} -o bin/main cmd/aws-lambda/run.go
	cp credentials.json bin/

.PHONY: build-docker
docker-build: docker-lint clean fmt credentials
	docker buildx build \
		--platform linux/amd64 \
		--build-arg COMMIT=${git-commit} \
		-t toshl-sync .; \
	make clean

.PHONY: test
test:
	go test -cover -coverprofile cover.out ./...

.PHONY: coverage
coverage: test
	go tool cover -html cover.out

.subject:
	@read -p "What subject should I use?: " subject; \
	echo $${subject} > .subject

registry.json: .subject
	@[ -z "${TOSHL_SECRETS_LOCATION}" ] && read -p "Where are the secrets?: " TOSHL_SECRETS_LOCATION; \
	registry=$$(cat .subject); \
	cp "$${TOSHL_SECRETS_LOCATION}/$${registry}_registry.json" registry.json

.PHONY: upload-docker
docker-push: registry.json
	awsprofile="$$(cat registry.json | jq -r .awsprofile)"; \
	registry="$$(cat registry.json | jq -r .registry)"; \
	aws --profile $${awsprofile} ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin $${registry}; \
	docker tag toshl-sync:latest $${registry}/private-ecr:latest; \
	docker push $${registry}/private-ecr:latest; \
	rm -f registry.json

.PHONY: credentials
credentials: credentials.json
credentials.json: .subject
	@[ -z "${TOSHL_SECRETS_LOCATION}" ] && read -p "Where are the secrets?: " TOSHL_SECRETS_LOCATION; \
	cred_file=$$(cat .subject); \
	cp "$${TOSHL_SECRETS_LOCATION}/$${cred_file}_creds.json" credentials.json

bin:
	mkdir -p bin

.PHONY: docker-lint
docker-lint:
	docker run --rm -i -v "$(PWD)/.hadolint.yaml:/.config/hadolint.yaml:ro" ghcr.io/hadolint/hadolint < Dockerfile

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

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: clean-all
clean-all: clean
	rm -f .subject

.PHONY: clean
clean:
	rm -rf bin/*
	rm -f credentials.json
	rm -f registry.json
