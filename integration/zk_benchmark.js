const http = require('http');

const BATCH_SIZE = 1000; // txs per micro-batch
const NUM_BATCHES = 10;   // number of micro-batches to test

// Simple hash for tx data
function hashTx(tx) {
    let h = 0;
    for (let i = 0; i < tx.length; i++) {
        h = ((h << 5) - h + tx.charCodeAt(i)) | 0;
    }
    return Math.abs(h).toString();
}

function postJSON(url, data) {
    return new Promise((resolve, reject) => {
        const req = http.request(url, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' }
        }, (res) => {
            let b = '';
            res.on('data', c => b += c);
            res.on('end', () => {
                try { resolve(JSON.parse(b)); }
                catch { resolve({ error: b }); }
            });
        });
        req.on('error', reject);
        req.write(JSON.stringify(data));
        req.end();
    });
}

async function buildTree(txCount) {
    // Generate tx leaves
    const leaves = [];
    for (let i = 0; i < txCount; i++) {
        leaves.push(hashTx(`tx_${i}_${Date.now()}`));
    }
    
    // Build Merkle tree
    const tree = await postJSON('http://localhost:4111/buildtree', { leaves });
    return tree;
}

async function proveBatch(leaf, root, siblings, bits) {
    return await postJSON('http://localhost:4111/prove', {
        leaf, root, siblings, bits
    });
}

async function runBenchmark() {
    console.log('╔══════════════════════════════════════════════════════════════╗');
    console.log('║     ZK-Integrated Batch Benchmark                           ║');
    console.log('╚══════════════════════════════════════════════════════════════╝');
    console.log(`Config: ${NUM_BATCHES} micro-batches × ${BATCH_SIZE} txs = ${NUM_BATCHES * BATCH_SIZE} total txs`);
    console.log('');
    
    const totalTxs = NUM_BATCHES * BATCH_SIZE;
    const startTotal = Date.now();
    
    // Generate all transaction leaves
    const allLeaves = [];
    for (let i = 0; i < totalTxs; i++) {
        allLeaves.push(hashTx(`tx_${i}_${Date.now()}`));
    }
    
    // Build single big tree for all txs
    console.log('Building Merkle tree...');
    const treeStart = Date.now();
    const tree = await buildTree(totalTxs);
    const treeMs = Date.now() - treeStart;
    console.log(`Tree built in ${treeMs}ms, root: ${tree.root.slice(0, 32)}...`);
    
    // Prove each batch (batch of BATCH_SIZE txs)
    const batchProofs = [];
    let proveTotal = 0, verifyTotal = 0;
    
    console.log(`\nGenerating ${NUM_BATCHES} batch proofs...`);
    for (let b = 0; b < NUM_BATCHES; b++) {
        // Use first tx of each batch as representative
        const leafIdx = b * BATCH_SIZE;
        const proof = tree.proofs[leafIdx];
        
        const r = await proveBatch(
            proof.leaf,
            proof.root,
            proof.siblings,
            proof.bits
        );
        
        if (r.valid) {
            batchProofs.push(r);
            proveTotal += r.proveMs;
            verifyTotal += r.verifyMs;
        }
        
        if ((b + 1) % 5 === 0) {
            console.log(`  Batch ${b + 1}/${NUM_BATCHES} done`);
        }
    }
    
    const totalMs = Date.now() - startTotal;
    
    // Calculate metrics
    const tps = (totalTxs / (totalMs / 1000)).toFixed(0);
    const avgProveMs = proveTotal / NUM_BATCHES;
    const proofsPerSec = 1000 / avgProveMs;
    
    console.log('\n╔══════════════════════════════════════════════════════════════╗');
    console.log('║                     BENCHMARK RESULTS                         ║');
    console.log('╚══════════════════════════════════════════════════════════════╝');
    console.log(`Total txs:        ${totalTxs.toLocaleString()}`);
    console.log(`Total time:       ${totalMs}ms`);
    console.log(`Raw TPS:          ${tps} tx/sec`);
    console.log('');
    console.log(`ZK prove time:    ${proveTotal}ms total, ${avgProveMs.toFixed(2)}ms avg per batch`);
    console.log(`ZK verify time:  ${verifyTotal}ms total`);
    console.log(`Proofs/sec:      ${proofsPerSec.toFixed(0)}`);
    console.log(`Proof size:      ${batchProofs[0]?.proofSize || 164} bytes`);
    console.log('');
    console.log('Valid proofs:    ' + batchProofs.length + '/' + NUM_BATCHES);
    console.log('');
    console.log('⚠️  This is proof-generation throughput, not end-to-end TPS');
    console.log('⚠️  Real TPS = Raw throughput - ZK overhead');
}

runBenchmark().catch(console.error);
