package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

func main() {
	abs, _ := filepath.Abs("../zk-prover/prove.js")
	workDir, _ := filepath.Abs(".")
	
	fmt.Println("Script:", abs)
	fmt.Println("WorkDir:", workDir)
	
	cmd := exec.Command("node", abs, `{"txCount":100,"batchHash":"abc"}`)
	cmd.Dir = workDir
	
	out, err := cmd.Output()
	fmt.Println("Output:", string(out))
	fmt.Println("Err:", err)
}
