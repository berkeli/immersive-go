SHELL=/bin/bash
.PHONY: run test develop

outputs:
	mkdir -p tmp/outputs

run: outputs
	docker build --target run -t run .
	docker run \
		--mount type=bind,source="$$(echo $$HOME)/.aws",target=/root/.aws \
		-e AWS_ROLE_ARN="arn:aws:iam::297880250375:role/GoCourseLambdaUserReadWriteS3"\
		-e AWS_REGION=eu-west-2 -e AWS_PROFILE=cyfplus -e S3_BUCKET="batch-processing-berkeli"\
		--mount type=bind,source="$$(pwd)/outputs",target=/outputs \
		--rm run

test: outputs
	docker build --target test -t test .
	docker run \
		--rm test \

develop: outputs
	docker build --target develop -t develop .
	docker run -it \
		--mount type=bind,source="$$(pwd)",target=/app \
		--mount type=bind,source="$$(echo $$HOME)/.aws",target=/root/.aws \
		-e AWS_ROLE_ARN="arn:aws:iam::297880250375:role/GoCourseLambdaUserReadWriteS3"\
		-e AWS_REGION=eu-west-2 -e AWS_PROFILE=cyfplus -e S3_BUCKET="batch-processing-berkeli"\
		--mount type=bind,source="$$(pwd)/outputs",target=/app/outputs \
		--rm develop

reader_server:
	go run ./cmd/reader_server

downloader:
	go run ./cmd/downloader

converter:
	go run ./cmd/converter

uploader:
	AWS_ROLE_ARN="arn:aws:iam::297880250375:role/GoCourseLambdaUserReadWriteS3" \
	AWS_REGION=eu-west-2 \
	AWS_PROFILE=cyfplus \
	S3_BUCKET="batch-processing-berkeli" \
	go run ./cmd/uploader

reader:
	go run ./cmd/read --input inputs/large-input.csv --addr localhost:50051

protoc: services/reader/service/reader.proto
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		services/reader/service/reader.proto