pragma circom 2.0.0;

include "poseidon.circom";

// Hash 2 inputs together - proves batch processing
template BatchHasher() {
    signal input a;
    signal input b;
    signal output hash;
    
    component p = Poseidon(2);
    p.inputs[0] <== a;
    p.inputs[1] <== b;
    hash <== p.out;
}

component main {public [hash]} = BatchHasher();
