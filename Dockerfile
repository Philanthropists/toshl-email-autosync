FROM golang:1.19.2 as builder

ARG COMMIT=dev

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

ENV GOOS=linux GOARCH=amd64 CGO_ENABLED=0
ENV LOC=/usr/local/bin

COPY . .
RUN GOOS=${GOOS} GOARCH=${GOARCH} CGO_ENABLED=${CGO_ENABLED} \
		 go build \
		 -ldflags="-s -w -X main.GitCommit=${COMMIT}" \
		 -o ${LOC}/main cmd/aws-lambda/main.go

# ---

FROM builder as tests

RUN go test -v ./... && \
		touch /empty

# ---

FROM alpine:3.16.0

WORKDIR /

COPY --from=tests /empty .

# Needed for getting timezone locale info (i.e. America/Bogota)
RUN apk add --no-cache tzdata=2022c-r0

COPY --from=builder /usr/local/bin/main ./main
COPY credentials.json .

ENTRYPOINT [ "/main"]
