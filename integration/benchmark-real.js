/**
 * Real ZK Batch Benchmark
 * Generates unique proofs per batch
 */

const { BatchProcessor } = require('./batch-processor');

const BATCH_SIZES = [100, 500, 1000, 5000, 10000];
const RUNS = 3;

async function generateTxs(count) {
    const txs = [];
    for (let i = 0; i < count; i++) {
        txs.push({
            from: '0x' + (Math.random().toString(16).slice(2, 42)).padStart(40, '0'),
            to: '0x' + (Math.random().toString(16).slice(2, 42)).padStart(40, '0'),
            value: Math.floor(Math.random() * 1000000),
            data: '0x'
        });
    }
    return txs;
}

async function runBenchmark(size, runs) {
    const bp = new BatchProcessor({ protocol: 'groth16' });
    await bp.loadKeys();
    
    const results = [];
    
    for (let run = 0; run < runs; run++) {
        const txs = await generateTxs(size);
        const result = await bp.processBatch(txs);
        results.push(result);
    }
    
    const avgProofTime = results.reduce((a, r) => a + r.proofTime, 0) / results.length;
    const avgTPS = results.reduce((a, r) => a + r.tps, 0) / results.length;
    
    return {
        batchSize: size,
        avgProofTime: Math.round(avgProofTime),
        avgTPS: Math.round(avgTPS),
        verified: results.filter(r => r.verified).length
    };
}

async function main() {
    console.log('💜 BRIXASCALER - REAL ZK BENCHMARK 💜\n');
    console.log('='.repeat(50));
    
    for (const size of BATCH_SIZES) {
        const result = await runBenchmark(size, RUNS);
        console.log(`Batch ${size.toLocaleString()}: ${result.avgProofTime}ms | ${result.avgTPS.toLocaleString()} TPS | Verified: ${result.verified}/${RUNS}`);
    }
    
    console.log('='.repeat(50));
    console.log('\n✅ REAL proofs generated for each batch!');
}

main().catch(console.error);
