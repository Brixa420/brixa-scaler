package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

func main() {
	batchHash := "abcdef1234567890abcdef1234567890"
	data := fmt.Sprintf(`{"txCount":1000,"batchHash":"%s"}`, batchHash[:32])
	
	fmt.Println("Calling:", data)
	
	cmd := exec.Command("node", "zk-prover/prove.js", data)
	output, err := cmd.Output()
	
	fmt.Println("Output:", string(output))
	fmt.Println("Err:", err)
	
	var result map[string]interface{}
	json.Unmarshal(output, &result)
	
	fmt.Println("Parsed:", result)
}
