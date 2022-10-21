FROM golang:1.19-buster AS build

WORKDIR /app

COPY api api
COPY cmd/api-server cmd/api-server
COPY go* .

RUN go mod download

RUN go build -o /app/api-server ./cmd/api-server

FROM gcr.io/distroless/base-debian10

COPY --from=build /app/api-server /app/api-server

ENTRYPOINT ["/app/api-server"]
