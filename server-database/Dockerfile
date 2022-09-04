FROM postgres

ENV POSTGRES_DB=go_server_test_db
ENV POSTGRES_USER=postgres
ENV POSTGRES_PASSWORD=postgres

COPY migrations.sql /docker-entrypoint-initdb.d/

