server:
	go run prober_server/main.go

client:
	go run prober_client/main.go --tries 14 --endpoint https://www.google.com/ --endpoint https://duckduckgo.com/ --timeout 3s

monitoring:
	docker build -t prometheus -f prometheus.Dockerfile .
	docker run --name prometheus \
    	-p 9090:9090 \
    	prometheus