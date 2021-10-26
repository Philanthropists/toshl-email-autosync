.PHONY: build
build: bin vendor fmt
	go build -o bin cmd/run/run.go
	go build -o bin cmd/aws-lambda/main.go
	cp credentials.json bin/

bin:
	mkdir -p bin

.PHONY: vet
vet:
	go vet ./...

.PHONY: staticcheck
staticcheck: vet
	go install honnef.co/go/tools/cmd/staticcheck@latest
	$(GOPATH)/bin/staticcheck ./...

.PHONY: fmt
fmt: staticcheck
	go fmt ./...

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: vendor
vendor: tidy
	go mod vendor

.PHONY: clean
clean:
	rm -rf bin/*
