const crypto = require('crypto');

function sha256(data) {
  return crypto.createHash('sha256').update(data).digest('hex');
}

function buildMerkleRoot(leaves) {
  if (!leaves.length) return sha256('empty');
  let layer = leaves.map(l => sha256(JSON.stringify(l)));
  while (layer.length > 1) {
    const next = [];
    for (let i = 0; i < layer.length; i += 2) {
      const left = layer[i];
      const right = layer[i + 1] || left;
      next.push(sha256(left + right));
    }
    layer = next;
  }
  return layer[0];
}

function processBatch(txCount) {
  const txs = [];
  for (let i = 0; i < txCount; i++) {
    txs.push({ from: `0x${i.toString(16)}`, to: `0x${i}`, value: i, nonce: i, data: 'data' });
  }
  const start = Date.now();
  const root = buildMerkleRoot(txs);
  const elapsed = Date.now() - start;
  return { root, elapsed, tps: txCount / (elapsed / 1000) };
}

async function run() {
  const sizes = [1000, 10000, 100000, 1000000];
  console.log('=== JS Merkle Layer Benchmark ===\n');
  for (const size of sizes) {
    const { root, elapsed, tps } = processBatch(size);
    console.log(`${size.toString().padStart(7)} txs: ${(elapsed/1000).toFixed(2)} ms | TPS: ${tps.toFixed(0)} | root: ${root.slice(0,16)}...`);
  }

  console.log('\n=== Stress Test (parallel) ===');
  for (const batchCount of [10, 50, 100]) {
    const txs = [];
    for (let i = 0; i < 100000; i++) {
      txs.push({ from: `0x${i}`, to: `0x${i}`, value: i, nonce: i, data: 'data' });
    }
    
    const start = Date.now();
    const promises = [];
    for (let b = 0; b < batchCount; b++) {
      promises.push(new Promise(resolve => {
        setImmediate(() => {
          const root = buildMerkleRoot(txs);
          resolve(root);
        });
      }));
    }
    await Promise.all(promises);
    const elapsed = Date.now() - start;
    const total = batchCount * 100000;
    console.log(`${batchCount} × 100K = ${total.toLocaleString()} txs in ${elapsed} ms | TPS: ${(total / (elapsed/1000)).toFixed(0)}`);
  }
}

run();
