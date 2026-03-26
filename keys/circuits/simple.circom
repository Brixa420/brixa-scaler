pragma circom 2.0.0;

template Hash2() {
    signal input a;
    signal input b;
    signal output hash;
    hash <-- a + b;
}

component main = Hash2();