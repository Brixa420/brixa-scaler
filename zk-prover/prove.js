#!/usr/bin/env node
const snarkjs = require('snarkjs');
const fs = require('fs');
const path = require('path');

const KEYS_DIR = path.join(__dirname, '..', 'keys');

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
    
    const input = { a: a.toString(), b: b.toString() };
    const start = Date.now();
    
    const { proof, publicSignals } = await snarkjs.groth16.fullProve(input, wasm, zkey);
    const proveMs = Date.now() - start;
    
    const vkData = JSON.parse(fs.readFileSync(vk, 'utf8'));
    const valid = await snarkjs.groth16.verify(vkData, publicSignals, proof);
    
    return { proveMs, valid };
}

// CLI args only
const args = process.argv.slice(2);
if (args.length > 0) {
    prove(JSON.parse(args[0])).then(r => console.log(JSON.stringify(r))).catch(e => console.error(e.message));
}
