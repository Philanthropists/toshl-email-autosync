FROM golang:1.19.3 as builder

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

ENV GOOS=linux GOARCH=amd64 CGO_ENABLED=0
ENV LOC=/usr/local/bin

COPY . .
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

COPY docker_entry.sh .
ADD https://github.com/aws/aws-lambda-runtime-interface-emulator/releases/latest/download/aws-lambda-rie .
RUN chmod 500 aws-lambda-rie

COPY --from=builder /usr/local/bin/main ./main
COPY credentials.json .
RUN echo "${COMMIT}" > ./version

ENTRYPOINT ["/docker_entry.sh"]
