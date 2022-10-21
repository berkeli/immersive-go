FROM golang:1.19-buster AS build

WORKDIR /app

COPY static static

COPY cmd/static-server cmd/static-server
COPY go* .

RUN go mod download

RUN go build -o /app/static-server ./cmd/static-server

FROM gcr.io/distroless/base-debian10

COPY --from=build /app/static-server /app/static-server
COPY assets /app/assets

ENTRYPOINT ["/app/static-server", "-path", "/app/assets"]
