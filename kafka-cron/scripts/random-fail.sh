#!/bin/sh
# Randomly fail a job
rand_time=$((RANDOM % 60))
if [ $((RANDOM % 2)) -eq 0 ]; then
    echo "Sleeping for $(($rand_time+120)) seconds"
    sleep $(($rand_time+120))
    echo "Job failed"
    exit 1
else
    echo "Sleeping for $(($rand_time+120)) seconds"
    sleep $(($rand_time+120))
    echo "Job succeeded"
    exit 0
fi