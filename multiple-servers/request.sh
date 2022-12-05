#!/bin/bash

for i in {1..1000}; do curl -s localhost:8082; sleep 1; done