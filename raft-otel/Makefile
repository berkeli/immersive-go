.PHONY: client
proto_gen:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative service/store/store.proto
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative service/consensus/consensus.proto
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative service/registry/registry.proto

run:
	docker-compose up -d
	make client

client:
	STORE_SERVER="localhost:50052" OTEL_SERVICE_NAME=raft-client go run cmd/client/main.go

destroy:
	docker-compose down
	docker rmi -f $$(docker images -qf "reference=raft-otel*")

