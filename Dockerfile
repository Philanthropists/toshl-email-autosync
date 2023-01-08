FROM golang:1.19.4 as builder

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

ENV GOOS=linux GOARCH=amd64 CGO_ENABLED=0
ENV LOC=/usr/local/bin

COPY cmd/aws-lambda ./cmd/aws-lambda
COPY internal ./internal
COPY pkg ./pkg

RUN GOOS=${GOOS} GOARCH=${GOARCH} CGO_ENABLED=${CGO_ENABLED} \
	go build \
	-ldflags="-s -w" \
	-o ${LOC}/main cmd/aws-lambda/run.go

# ---

FROM builder as tests

RUN go test -v ./... && \
	touch /empty

# ---

FROM alpine:3.17.0

ARG COMMIT=dev

WORKDIR /

COPY --from=tests /empty .

# Needed for getting timezone locale info (i.e. America/Bogota)
RUN apk add --no-cache tzdata=2022f-r1

COPY docker_entry.sh credentials.json ./
ADD https://github.com/aws/aws-lambda-runtime-interface-emulator/releases/latest/download/aws-lambda-rie .

RUN chmod 755 aws-lambda-rie docker_entry.sh credentials.json

COPY --from=builder /usr/local/bin/main ./main
RUN echo "${COMMIT}" > ./version

ENTRYPOINT ["/docker_entry.sh"]
