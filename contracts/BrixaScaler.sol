// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/**
 * @title BrixaScaler L1 Contract
 * @notice Receives batch commitments from the scaling layer
 */
contract BrixaScaler {
    // State
    mapping(bytes32 => bool) public committedBatches;
    mapping(address => bool) public authorizedBatches;
    uint256 public batchCount;
    
    // Events
    event BatchCommitted(bytes32 indexed root, address batcher, uint256 timestamp);
    event BatchSettled(bytes32 indexed root, uint256 batchId);
    
    // Modifiers
    modifier onlyAuthorized() {
        require(authorizedBatches[msg.sender], "Not authorized batcher");
        _;
    }
    
    constructor() {
        authorizedBatches[msg.sender] = true; // Deployer is authorized
    }
    
    /**
     * @notice Commit a batch root from the scaling layer
     * @param root Merkle root of the batch
     */
    function commitBatch(bytes32 root) external onlyAuthorized {
        require(!committedBatches[root], "Batch already committed");
        
        committedBatches[root] = true;
        batchCount++;
        
        emit BatchCommitted(root, msg.sender, block.timestamp);
    }
    
    /**
     * @notice Verify if a batch root has been committed
     * @param root Merkle root to check
     * @return bool True if committed
     */
    function isBatchCommitted(bytes32 root) external view returns (bool) {
        return committedBatches[root];
    }
    
    /**
     * @notice Authorize a new batcher address
     * @param batcher Address to authorize
     */
    function authorizeBatcher(address batcher) external {
        require(msg.sender == address(this), "Only self");
        authorizedBatches[batcher] = true;
    }
}
