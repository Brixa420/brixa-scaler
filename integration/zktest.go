package main

import (
	"fmt"
	"os/exec"
)

func main() {
	cmd := exec.Command("bash", "-c", `node ../zk-prover/prove.js '{"txCount":100,"batchHash":"abcdef1234567890abcdef1234567890"}'`)
	out, err := cmd.Output()
	fmt.Println("out:", string(out))
	fmt.Println("err:", err)
}
