start_prom:
	docker build -t prometheus -f prometheus.Dockerfile .
	docker run --rm\
		--name prometheus \
    	-p 9090:9090 \
    	prometheus