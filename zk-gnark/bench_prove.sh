#!/bin/bash

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║           REAL BENCHMARK - PROVE ONLY (No Verify)            ║"
echo "╚══════════════════════════════════════════════════════════════╝"

for n in 10 50 100 500 1000; do
    leaves=$(seq 1 $n | tr '\n' ',' | sed 's/,$//')
    
    # Get proofs from buildtree
    proofs=$(curl -s -X POST http://localhost:4111/buildtree \
      -H "Content-Type: application/json" \
      -d "{\"leaves\":[$leaves]}")
    
    # Time just prove (no verify)
    start=$(date +%s%N)
    for i in $(seq 1 $n); do
        p=$(echo "$proofs" | jq -c ".proofs[$((i-1))]")
        curl -s -X POST http://localhost:4111/prove \
          -H "Content-Type: application/json" \
          -d "$p" > /dev/null 2>&1
    done
    ms=$((($(date +%s%N) - start) / 1000000))
    
    rate=$((n * 1000 / ms))
    echo "  $n proofs: ${ms}ms = $rate proofs/sec"
done

echo ""
echo "Proof size check (single proof):"
size=$(curl -s -X POST http://localhost:4111/buildtree \
  -H "Content-Type: application/json" \
  -d '{"leaves":["1"]}' | jq -c '.proofs[0]' | curl -s -X POST http://localhost:4111/prove \
  -H "Content-Type: application/json" \
  -d @- | jq '.proofSize')
echo "  Single proof: $size bytes"
