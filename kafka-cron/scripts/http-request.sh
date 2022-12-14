#!/bin/sh
# Make an HTTP request to a service, and check status code

wget --spider -S "http://fake-service:8080" 2>&1 | grep "HTTP/" | awk '{print $2}' | grep -q "200"

if [ $? -eq 0 ]; then
    echo "Job succeeded"
    sleep 60
    exit 0
else
    echo "Job failed"
    sleep 60
    exit 1
fi

