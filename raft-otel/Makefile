include $(PWD)/.env
.PHONY: client
proto_gen:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative service/store/store.proto
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative service/consensus/consensus.proto
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative service/registry/registry.proto

run:
	docker-compose up -d
	make client

add:
	docker-compose --profile add up

client:
	STORE_SERVER="localhost:8080" OTEL_SERVICE_NAME=raft-client HONEYCOMB_API_KEY=$(HONEYCOMB_API_KEY) go run cmd/client/main.go

destroy:
	docker-compose down
	docker rmi -f $$(docker images -qf "reference=raft-otel*")

