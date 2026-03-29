const { 
  Field, 
  Poseidon, 
  MerkleTree, 
  Bool,
  ZkProgram
} = require('o1js');

const MerkleProgram = ZkProgram({
  name: 'MerkleProof',
  methods: {
    verify: {
      // Public: root
      // Private: leaf, 4 path siblings, 4 isLeft flags
      privateInputs: [Field, Field, Field, Field, Field, Field, Bool, Bool, Bool, Bool],
      
      method(root, leaf, path0, path1, path2, path3, isLeft0, isLeft1, isLeft2, isLeft3) {
        // Level 0
        let hash = isLeft0.equals(Bool(true))
          ? Poseidon.hash([leaf, path0])
          : Poseidon.hash([path0, leaf]);
          
        // Level 1
        hash = isLeft1.equals(Bool(true))
          ? Poseidon.hash([hash, path1])
          : Poseidon.hash([path1, hash]);
          
        // Level 2  
        hash = isLeft2.equals(Bool(true))
          ? Poseidon.hash([hash, path2])
          : Poseidon.hash([path2, hash]);
          
        // Level 3 (root)
        hash = isLeft3.equals(Bool(true))
          ? Poseidon.hash([hash, path3])
          : Poseidon.hash([path3, hash]);
          
        root.assertEquals(hash);
      }
    }
  }
});

async function setup() {
  console.log('╔══════════════════════════════════════════════════════════════╗');
  console.log('║         o1js MERKLE - Fixed with isLeft Flags             ║');
  console.log('╚══════════════════════════════════════════════════════════════╝');
  console.log('Compiling...');
  
  await MerkleProgram.compile();
  console.log('✅ Compiled!');
  
  // Test
  const tree = new MerkleTree(5);
  for (let i = 0; i < 16; i++) tree.setLeaf(BigInt(i), new Field(i + 1));
  const root = tree.getRoot();
  const witness = tree.getWitness(5n);
  const leaf = tree.getLeaf(5n);
  
  console.log('\nGenerating proof...');
  const proof = await MerkleProgram.verify(
    root, leaf,
    witness[0].sibling, witness[1].sibling, witness[2].sibling, witness[3].sibling,
    new Bool(witness[0].isLeft), new Bool(witness[1].isLeft), 
    new Bool(witness[2].isLeft), new Bool(witness[3].isLeft)
  );
  
  console.log('Proof generated!');
  
  // Verify
  const valid = await MerkleProgram.verify(proof, { root });
  console.log('Verification:', valid ? '✅ SUCCESS' : '❌ FAILED');
  
  startServer(tree);
}

function startServer(tree) {
  const http = require('http');
  const server = http.createServer(async (req, res) => {
    res.setHeader('Content-Type', 'application/json');
    
    if (req.method === 'POST' && req.url === '/buildtree') {
      let body = '';
      req.on('data', c => body += c);
      req.on('end', () => {
        const { leaves } = JSON.parse(body);
        const t = new MerkleTree(5);
        for (let i = 0; i < Math.min(leaves.length, 16); i++) {
          t.setLeaf(BigInt(i), new Field(leaves[i]));
        }
        const root = t.getRoot();
        const proofs = [];
        for (let i = 0; i < Math.min(leaves.length, 16); i++) {
          const w = t.getWitness(BigInt(i));
          proofs.push({ 
            leaf: leaves[i], 
            root: root.toString(), 
            path: w.map(x => x.sibling.toString()),
            isLeft: w.map(x => x.isLeft)
          });
        }
        res.end(JSON.stringify({ root: root.toString(), proofs }));
      });
      return;
    }
    
    if (req.method === 'POST' && req.url === '/prove') {
      let body = '';
      req.on('data', c => body += c);
      req.on('end', async () => {
        const { leaf, root, path, isLeft } = JSON.parse(body);
        try {
          const start = Date.now();
          const proof = await MerkleProgram.verify(
            new Field(root), new Field(leaf),
            new Field(path[0] || 0), new Field(path[1] || 0),
            new Field(path[2] || 0), new Field(path[3] || 0),
            new Bool(isLeft[0]), new Bool(isLeft[1]), new Bool(isLeft[2]), new Bool(isLeft[3])
          );
          const proveMs = Date.now() - start;
          
          const vStart = Date.now();
          const valid = await MerkleProgram.verify(proof, { root: new Field(root) });
          const verifyMs = Date.now() - vStart;
          
          res.end(JSON.stringify({ 
            proveMs, verifyMs, 
            proofSize: JSON.stringify(proof).length, 
            valid 
          }));
        } catch (e) {
          res.end(JSON.stringify({ error: e.message }));
        }
      });
      return;
    }
    res.end(JSON.stringify({ error: 'Unknown endpoint' }));
  });
  
  server.listen(4112, () => console.log('\n🚀 o1js ready on port 4112'));
}

setup();
