FROM golang:1.19-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY ./setup ./
COPY ./types ./types
COPY ./utils ./utils


RUN go build -o /build
