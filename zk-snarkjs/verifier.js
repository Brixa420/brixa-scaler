#!/usr/bin/env node
/**
 * External ZK Verifier using snarkjs (JavaScript/WASM)
 * Much faster verification than Go gnark for small circuits
 */

const fs = require('fs');
const path = require('path');
const http = require('http');

const PORT = process.env.PORT || '4112';

// snarkjs Groth16 verification
async function groth16Verify(proofJson, vkJson) {
    const { groth16 } = await import('snarkjs');
    
    const proof = typeof proofJson === 'string' ? JSON.parse(proofJson) : proofJson;
    const vk = typeof vkJson === 'string' ? JSON.parse(vkJson) : vkJson;
    
    const start = Date.now();
    const isValid = await groth16.isValid(vk, proof);
    const verifyMs = Date.now() - start;
    
    return { isValid, verifyMs };
}

// Batch verification using snarkjs
async function batchVerify(proofs, vk) {
    const { groth16 } = await import('snarkjs');
    
    const start = Date.now();
    const results = [];
    
    for (const proof of proofs) {
        const p = typeof proof === 'string' ? JSON.parse(proof) : proof;
        const isValid = await groth16.isValid(vk, p);
        results.push(isValid);
    }
    
    const batchMs = Date.now() - start;
    return { results, batchMs };
}

// Generate ZKey (proving key) from compiled circuit
async function setupCircuit(pkeyPath, zkeyOutPath) {
    const { groth16 } = await import('snarkjs');
    
    console.log('Running Groth16 setup...');
    const ptauFile = 'ptau.json';
    
    // Check for existing ptau or create new
    let ptau;
    if (fs.existsSync(ptauFile)) {
        console.log('Loading existing ptau...');
        ptau = JSON.parse(fs.readFileSync(ptauFile));
    }
    
    // This would need a compiled .wasm + .r1cs from circom
    console.log('Setup requires compiled circuit from circom');
}

// HTTP Server
const server = http.createServer(async (req, res) => {
    res.setHeader('Access-Control-Allow-Origin', '*');
    res.setHeader('Access-Control-Allow-Methods', 'POST, GET, OPTIONS');
    res.setHeader('Access-Control-Allow-Headers', 'Content-Type');
    
    if (req.method === 'OPTIONS') {
        res.writeHead(200);
        res.end();
        return;
    }
    
    if (req.url === '/health') {
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ status: 'ok', backend: 'snarkjs' }));
        return;
    }
    
    if (req.url === '/verify' && req.method === 'POST') {
        let body = '';
        req.on('data', chunk => body += chunk);
        req.on('end', async () => {
            try {
                const { proof, publicSignals } = JSON.parse(body);
                
                // Load VK from file or use embedded
                const vkPath = process.env.VK_PATH || path.join(__dirname, 'vk.json');
                const vk = JSON.parse(fs.readFileSync(vkPath));
                
                const result = await groth16Verify(proof, vk);
                
                res.writeHead(200, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({
                    valid: result.isValid,
                    verifyMs: result.verifyMs
                }));
            } catch (err) {
                res.writeHead(500, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({ error: err.message }));
            }
        });
        return;
    }
    
    if (req.url === '/batch-verify' && req.method === 'POST') {
        let body = '';
        req.on('data', chunk => body += chunk);
        req.on('end', async () => {
            try {
                const { proofs } = JSON.parse(body);
                
                const vkPath = process.env.VK_PATH || path.join(__dirname, 'vk.json');
                const vk = JSON.parse(fs.readFileSync(vkPath));
                
                const result = await batchVerify(proofs, vk);
                
                res.writeHead(200, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({
                    results: result.results,
                    batchMs: result.batchMs,
                    avgMs: result.batchMs / proofs.length
                }));
            } catch (err) {
                res.writeHead(500, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({ error: err.message }));
            }
        });
        return;
    }
    
    if (req.url === '/benchmark' && req.method === 'POST') {
        let body = '';
        req.on('data', chunk => body += chunk);
        req.on('end', async () => {
            try {
                const { iterations } = JSON.parse(body) || { iterations: 100 };
                
                const vkPath = process.env.VK_PATH || path.join(__dirname, 'vk.json');
                const vk = JSON.parse(fs.readFileSync(vkPath));
                
                // Use a sample proof for benchmarking
                const sampleProofPath = path.join(__dirname, 'sample_proof.json');
                if (!fs.existsSync(sampleProofPath)) {
                    res.writeHead(400);
                    res.end(JSON.stringify({ error: 'No sample proof available' }));
                    return;
                }
                
                const sampleProof = JSON.parse(fs.readFileSync(sampleProofPath));
                
                const start = Date.now();
                for (let i = 0; i < iterations; i++) {
                    await groth16Verify(sampleProof, vk);
                }
                const totalMs = Date.now() - start;
                
                res.writeHead(200, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({
                    iterations,
                    totalMs,
                    perVerifyMs: totalMs / iterations,
                    tps: Math.round(iterations * 1000 / totalMs)
                }));
            } catch (err) {
                res.writeHead(500, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({ error: err.message }));
            }
        });
        return;
    }
    
    res.writeHead(404);
    res.end('Not found');
});

server.listen(PORT, () => {
    console.log(`🔐 External ZK Verifier (snarkjs) running on port ${PORT}`);
    console.log('Endpoints:');
    console.log('  POST /verify      - Single proof verification');
    console.log('  POST /batch-verify - Batch verification');
    console.log('  POST /benchmark  - Run benchmark');
    console.log('  GET  /health     - Health check');
});