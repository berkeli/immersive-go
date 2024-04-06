FROM golang:1.19-buster AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY *.go ./

RUN go build -o /app/build

FROM gcr.io/distroless/base-debian10

COPY --from=build /app/build /app/build

EXPOSE 80

ENTRYPOINT ["/app/build"]