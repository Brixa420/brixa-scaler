template StarkHash() {
    signal input in;
    signal output out;
    
    // Simple hash: square and mod
    out <-- in * in % 2188824287183929392224741280424138523977311400354528079069138580175;
}

component main = StarkHash();
