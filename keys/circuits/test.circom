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
