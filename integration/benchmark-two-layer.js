/**
 * Two-Layer BrixaScaler Benchmark
 */

const { poseidon2 } = require('poseidon-lite');
const { execSync } = require('child_process');
const fs = require('fs');

const BN254_FIELD = 21888242871839275222246405745257275088548364400416034343698204186575808495617n;

function mod(n) { return ((n % BN254_FIELD) + BN254_FIELD) % BN254_FIELD; }
function hash(left, right) { return mod(poseidon2([BigInt(left), BigInt(right)], 1)); }

function hashString(s) {
  let h = 0n;
  for (let i = 0; i < s.length; i++) {
    h = (h << 8n) + BigInt(s.charCodeAt(i));
  }
  return mod(h);
}

// Layer 1: Transaction Batching
class TransactionBatcher {
  constructor(batchSize = 16) {
    this.batchSize = batchSize;
    this.pending = [];
  }
  
  addTransaction(tx) {
    this.pending.push({
      from: tx.from || '0x' + Math.random().toString(16).slice(2, 42),
      to: tx.to || '0x' + Math.random().toString(16).slice(2, 42),
      value: tx.value || '0x0',
      data: tx.data || '0x'
    });
  }
  
  buildMerkleTree() {
    const leaves = this.pending.map(tx => {
      const data = tx.from + tx.to + tx.value + tx.data;
      return hashString(data);
    });
    
    // Pad to power of 2
    while (leaves.length < this.batchSize) {
      leaves.push(leaves[leaves.length - 1] || 0n);
    }
    
    let level = leaves;
    while (level.length > 1) {
      const next = [];
      for (let i = 0; i < level.length; i += 2) {
        next.push(hash(level[i], level[i+1] || level[i]));
      }
      level = next;
    }
    
    return { root: level[0], leaves: this.pending };
  }
  
  generateProof(leafIndex) {
    const { root, leaves } = this.buildMerkleTree();
    
    // Rebuild tree to get proof
    let tree = [this.pending.map(tx => hashString(tx.from + tx.to + tx.value + tx.data))];
    while (tree[tree.length-1].length > 1) {
      const next = [];
      const level = tree[tree.length-1];
      for (let i = 0; i < level.length; i += 2) {
        next.push(hash(level[i], level[i+1] || level[i]));
      }
      tree.push(next);
    }
    
    const levels = tree.length - 1;
    let idx = leafIndex;
    const pathElements = [], pathIndices = [];
    
    for (let l = 0; l < levels; l++) {
      const siblingIdx = idx % 2 === 0 ? idx + 1 : idx - 1;
      const sibling = tree[l][siblingIdx] ?? tree[l][idx];
      pathElements.push(sibling.toString());
      pathIndices.push(idx % 2);
      idx = Math.floor(idx / 2);
    }
    
    return {
      leaf: leaves[leafIndex]?.from + leaves[leafIndex]?.to + leaves[leafIndex]?.value + leaves[leafIndex]?.data || '0',
      root: root.toString(),
      pathElements,
      pathIndices
    };
  }
}

// Benchmark
async function runBenchmark() {
  console.log('=== Two-Layer BrixaScaler Benchmark ===\n');
  
  const batcher = new TransactionBatcher(16);
  const iterations = 10;
  
  // Layer 1: Batching throughput
  console.log('📦 Layer 1: Transaction Batching');
  const batchTimes = [];
  for (let i = 0; i < iterations; i++) {
    batcher.pending = [];
    for (let j = 0; j < 16; j++) {
      batcher.addTransaction({ 
        from: '0x' + Math.random().toString(16).slice(2, 42), 
        to: '0x' + Math.random().toString(16).slice(2, 42),
        value: '0x' + Math.floor(Math.random() * 1000).toString(16)
      });
    }
    
    const start = Date.now();
    const { root } = batcher.buildMerkleTree();
    batchTimes.push(Date.now() - start);
  }
  
  const avgBatch = batchTimes.reduce((a,b) => a+b, 0) / batchTimes.length;
  console.log(`   Batch (16 txs): ${avgBatch.toFixed(1)} ms`);
  console.log(`   Throughput: ${(16000/avgBatch).toFixed(0)} tx/sec\n`);
  
  // Layer 2: ZK Proof - use working input from before
  console.log('🔐 Layer 2: ZK Proof Generation');
  
  const proof = batcher.generateProof(0);
  // Fix leaf to be a proper BigInt string
  const leafHash = hashString(proof.leaf).toString();
  
  const zkInput = {
    leaf: leafHash,
    root: proof.root,
    pathElements: proof.pathElements,
    pathIndices: proof.pathIndices.map(String)
  };
  
  fs.writeFileSync('/tmp/zk_input.json', JSON.stringify(zkInput));
  
  const zkTimes = [];
  for (let i = 0; i < iterations; i++) {
    const start = Date.now();
    try {
      execSync('snarkjs g16f /tmp/zk_input.json /tmp/batch_merkle.wasm /tmp/batch_merkle_final_new.zkey /tmp/proof.json /tmp/public.json', { 
        cwd: '/tmp', 
        stdio: 'pipe' 
      });
      zkTimes.push(Date.now() - start);
    } catch(e) {
      console.log('Error:', e.message?.slice(0,50));
    }
  }
  
  const avgZK = zkTimes.reduce((a,b) => a+b, 0) / zkTimes.length;
  console.log(`   Prove time: ${avgZK.toFixed(0)} ms`);
  console.log(`   TPS: ${(1000/avgZK).toFixed(1)} proofs/sec\n`);
  
  // Combined
  console.log('📊 Combined Two-Layer System');
  const totalTime = avgBatch + avgZK;
  console.log(`   Total time per batch: ${totalTime.toFixed(0)} ms`);
  console.log(`   Effective TPS: ${(16000/totalTime).toFixed(0)} tx/sec`);
}

runBenchmark();
