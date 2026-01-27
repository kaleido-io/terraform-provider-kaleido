#!/bin/bash
# Quick test to see if FireFly resources are registered

echo "Checking if provider recognizes FireFly resources..."
terraform providers schema 2>&1 | grep -i "firefly" || echo "No FireFly resources found in schema"

echo ""
echo "Checking resource types in configuration..."
grep -E 'resource "kaleido_platform_firefly' main.tf

echo ""
echo "If the above shows the resources, they are in your config."
echo "If terraform plan hangs, it's likely trying to connect to the API."
echo "Try running: terraform plan -refresh=false"
