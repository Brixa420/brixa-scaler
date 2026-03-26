# BrixaScaler ZK Setup Guide

## Prerequisites
```bash
# Node.js 18+ required (not Node 25)
nvm use 18

# Install circom and snarkjs
npm install -g circom snarkjs

# Clone circomlib
git clone https://github.com/iden3/circomlib.git ~/.circomlib
```

## Circuit: `keys/circuits/batch_merkle.circom`
```circom
pragma circom 2.0.0;

include "poseidon.circom";
include "switcher.circom";

template MerkleTreeChecker(levels) {
    signal input leaf;
    signal input root;
    signal input pathElements[levels];
    signal input pathIndices[levels];

    signal hash[levels + 1];
    hash[0] <== leaf;

    component switcher[levels];
    component hasher[levels];

    for (var i = 0; i < levels; i++) {
        switcher[i] = Switcher();
        switcher[i].sel <== pathIndices[i];
        switcher[i].L <== hash[i];
        switcher[i].R <== pathElements[i];
        hasher[i] = Poseidon(2);
        hasher[i].inputs[0] <== switcher[i].outL;
        hasher[i].inputs[1] <== switcher[i].outR;
        hash[i+1] <== hasher[i].out;
    }

    root === hash[levels];
}

component main {public [root]} = MerkleTreeChecker(4);
```

## Compile (when circom works)
```bash
cd keys/circuits
circom batch_merkle.circom --r1cs --wasm --sym

# Trusted setup
snarkjs powersoftau new bn128 15 pot15_0000.ptau -v
snarkjs powersoftau contribute pot15_0000.ptau pot15_0001.ptau --name="contrib" -e="entropy"
snarkjs powersoftau prepare phase2 pot15_0001.ptau pot15_final.ptau -v

# Generate keys
snarkjs groth16 setup batch_merkle.r1cs pot15_final.ptau batch_merkle.zkey
snarkjs zkey export verificationkey batch_merkle.zkey verification_key.json
```

## Current Status
- Circuit code: ✅ Documented
- circom compiler: ⚠️ Broken on Node 25 (use Node 18)
- Go ZK mocks: ✅ Working (SHA256-based)

The Go server uses SHA256-based ZK mocks for now.
