FROM golang:1.19-buster AS build

WORKDIR /app

COPY static static
COPY api api
COPY cmd/api-server cmd/api-server
COPY cmd/static-server cmd/static-server
COPY go* .

RUN go mod download

RUN go build -o /app/static-server ./cmd/static-server
RUN go build -o /app/api-server ./cmd/api-server

FROM ubuntu:latest

COPY --from=build /app/static-server /app/static-server
COPY --from=build /app/api-server /app/api-server
COPY assets /app/assets
COPY config /app/config

WORKDIR /app

RUN apt-get update -y
RUN DEBIAN_FRONTEND=noninteractive apt-get install postgresql -y
RUN sed '/^# Database administrative=.*/a local   all             root                                md5' /etc/postgresql/14/main/pg_hba.conf
RUN apt-get install nginx -y

ADD migrations.sql /var/lib/postgresql/migrations.sql

EXPOSE 8080

CMD service postgresql start; \
    runuser -l postgres -c "psql -c\"CREATE USER root WITH PASSWORD 'password';\""; \
    runuser -l postgres -c "psql -c\"CREATE DATABASE go_server_db;\""; \
    psql -d go_server_db -f /var/lib/postgresql/migrations.sql; \
    DATABASE_URL=postgres://root:password@127.0.0.1:5432/go_server_db ./api-server & \
    ./static-server --path assets & \
    nginx -c `pwd`/config/nginx.conf
