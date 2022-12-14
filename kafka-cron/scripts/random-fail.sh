#!/bin/sh
# Randomly fail a job
if [ $((RANDOM % 2)) -eq 0 ]; then
    echo "Job failed"
    sleep 60
    exit 1
else
    echo "Job succeeded"
    sleep 60
    exit 0
fi