#!/usr/bin/env node
const http = require('http');
const snarkjs = require('snarkjs');
const fs = require('fs');
const path = require('path');

const KEYS_DIR = path.join(__dirname, '..', 'keys');
const PORT = process.argv[2] || 3111;

const { zkey, wasm, vk } = {
    zkey: path.join(KEYS_DIR, 'simple_0001.zkey'),
    wasm: path.join(KEYS_DIR, 'simple_js/simple.wasm'),
    vk: path.join(KEYS_DIR, 'simple_vk.json')
};

async function prove(data) {
    const { txCount, batchHash } = data;
    const BN = BigInt(21888242871839275222246405745257275088548364400416034343698204186575808495617);
    const a = BigInt('0x' + batchHash.slice(0, 16)) % BN;
    const b = BigInt(txCount);
    
    const { proof, publicSignals } = await snarkjs.groth16.fullProve({ a: a.toString(), b: b.toString() }, wasm, zkey);
    const vkData = JSON.parse(fs.readFileSync(vk, 'utf8'));
    const valid = await snarkjs.groth16.verify(vkData, publicSignals, proof);
    
    return { proveMs: Date.now(), valid };
}

const server = http.createServer(async (req, res) => {
    if (req.method === 'POST' && req.url === '/prove') {
        let body = '';
        req.on('data', d => body += d);
        req.on('end', async () => {
            try {
                const result = await prove(JSON.parse(body));
                res.writeHead(200, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify(result));
            } catch (e) {
                res.writeHead(500);
                res.end(JSON.stringify({ error: e.message }));
            }
        });
    } else {
        res.writeHead(404);
        res.end();
    }
});

server.listen(PORT, () => console.log(`ZK Prover running on port ${PORT}`));
