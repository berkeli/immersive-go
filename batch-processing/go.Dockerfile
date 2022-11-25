FROM golang:1.19-buster AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY cmd ./cmd
COPY services ./services
COPY utils ./utils

RUN go build -o /app/build ./...

FROM gcr.io/distroless/base-debian10 AS app

COPY --from=build /app/build /app/build

FROM app AS reader_server

ENTRYPOINT ["/app/build", "reader_server"]

FROM app AS downloader
RUN mkdir -p /outputs
ENTRYPOINT ["/app/build", "downloader"]

FROM app AS converter
RUN mkdir -p /outputs
ENTRYPOINT ["/app/build", "converter"]

FROM app AS uploader
RUN mkdir -p /outputs
ENTRYPOINT ["/app/build", "uploader"]