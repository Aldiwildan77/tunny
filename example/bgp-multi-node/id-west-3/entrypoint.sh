#!/bin/bash
set -e

echo "================================="
echo " Starting provider node"
echo "================================="

echo "[node] IP addresses:"
ip -brief addr

echo "[node] Routes:"
ip route

echo "[node] Starting FRR..."

# Start FRR daemons
/usr/lib/frr/frrinit.sh start

sleep 2

echo "[node] FRR status:"
vtysh -c "show ip bgp summary" || true

echo "[node] Starting Tunny provider..."

exec tunny provider -c /etc/tunny/config.yaml