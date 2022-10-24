FROM prom/prometheus

# Add our custom config
ADD prometheus.yml /etc/prometheus/prometheus.yml