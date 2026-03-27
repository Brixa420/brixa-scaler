package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// StateStore handles persistent state using LevelDB
type StateStore struct {
	db   *leveldb.DB
	mu   sync.RWMutex
	cfg  *PersistenceConfig
}

// PersistenceConfig holds persistence settings
type PersistenceConfig struct {
	Enabled     bool   `yaml:"enabled"`
	Backend     string `yaml:"backend"`
	Path        string `yaml:"path"`
	SyncWrites  bool   `yaml:"sync_writes"`
	Compression bool   `yaml:"compression"`
}

// StateRoot represents a verified state root
type StateRoot struct {
	BlockHeight uint64    `json:"block_height"`
	Timestamp   int64     `json:"timestamp"`
	BatchID     string    `json:"batch_id"`
	MerkleRoot  string    `json:"merkle_root"`
	TxCount     uint64    `json:"tx_count"`
}

// Batch represents a batch of transactions
type Batch struct {
	BatchID     string   `json:"batch_id"`
	TxHashes    []string `json:"tx_hashes"`
	MerkleRoot  string   `json:"merkle_root"`
	Proof       []byte   `json:"proof"`
	Timestamp   int64    `json:"timestamp"`
	Settled     bool     `json:"settled"`
	SettlementTx string  `json:"settlement_tx,omitempty"`
}

// TxRecord represents a transaction in the store
type TxRecord struct {
	Hash          string `json:"hash"`
	BatchID       string `json:"batch_id"`
	Position      uint64 `json:"position"`
	InclusionProof []byte `json:"inclusion_proof"`
	Timestamp     int64  `json:"timestamp"`
	Status        string `json:"status"` // pending, batched, settled
}

// NewStateStore creates a new LevelDB state store
func NewStateStore(cfg *PersistenceConfig) (*StateStore, error) {
	if !cfg.Enabled {
		return &StateStore{cfg: cfg}, nil
	}

	dbOpts := &opt.Options{
		NoSync: !cfg.SyncWrites,
	}
	if cfg.Compression {
		dbOpts.Compression = opt.SnappyCompression
	}

	db, err := leveldb.OpenFile(cfg.Path, dbOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to open LevelDB: %w", err)
	}

	return &StateStore{
		db:  db,
		cfg: cfg,
	}, nil
}

// Close closes the state store
func (s *StateStore) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

// PutStateRoot stores a state root
func (s *StateStore) PutStateRoot(root *StateRoot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := []byte(fmt.Sprintf("sr:%d", root.BlockHeight))
	value, err := json.Marshal(root)
	if err != nil {
		return err
	}
	return s.db.Put(key, value, nil)
}

// GetStateRoot retrieves a state root by block height
func (s *StateStore) GetStateRoot(height uint64) (*StateRoot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := []byte(fmt.Sprintf("sr:%d", height))
	value, err := s.db.Get(key, nil)
	if err != nil {
		return nil, err
	}

	var root StateRoot
	if err := json.Unmarshal(value, &root); err != nil {
		return nil, err
	}
	return &root, nil
}

// GetLatestStateRoot returns the most recent state root
func (s *StateStore) GetLatestStateRoot() (*StateRoot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	iter := s.db.NewIterator(util.BytesPrefix([]byte("sr:")), nil)
	defer iter.Release()

	if !iter.Last() {
		return nil, fmt.Errorf("no state roots found")
	}

	var root StateRoot
	if err := json.Unmarshal(iter.Value(), &root); err != nil {
		return nil, err
	}
	return &root, nil
}

// PutBatch stores a batch
func (s *StateStore) PutBatch(batch *Batch) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := []byte(fmt.Sprintf("batch:%s", batch.BatchID))
	value, err := json.Marshal(batch)
	if err != nil {
		return err
	}
	return s.db.Put(key, value, nil)
}

// GetBatch retrieves a batch by ID
func (s *StateStore) GetBatch(id string) (*Batch, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := []byte(fmt.Sprintf("batch:%s", id))
	value, err := s.db.Get(key, nil)
	if err != nil {
		return nil, err
	}

	var batch Batch
	if err := json.Unmarshal(value, &batch); err != nil {
		return nil, err
	}
	return &batch, nil
}

// MarkBatchSettled marks a batch as settled
func (s *StateStore) MarkBatchSettled(batchID, settlementTx string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	batch, err := s.GetBatch(batchID)
	if err != nil {
		return err
	}
	batch.Settled = true
	batch.SettlementTx = settlementTx

	value, err := json.Marshal(batch)
	if err != nil {
		return err
	}
	key := []byte(fmt.Sprintf("batch:%s", batchID))
	return s.db.Put(key, value, nil)
}

// PutTx stores a transaction record
func (s *StateStore) PutTx(tx *TxRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := []byte(fmt.Sprintf("tx:%s", tx.Hash))
	value, err := json.Marshal(tx)
	if err != nil {
		return err
	}
	return s.db.Put(key, value, nil)
}

// GetTx retrieves a transaction by hash
func (s *StateStore) GetTx(hash string) (*TxRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := []byte(fmt.Sprintf("tx:%s", hash))
	value, err := s.db.Get(key, nil)
	if err != nil {
		return nil, err
	}

	var tx TxRecord
	if err := json.Unmarshal(value, &tx); err != nil {
		return nil, err
	}
	return &tx, nil
}

// PutPendingTx stores a pending transaction
func (s *StateStore) PutPendingTx(txHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := []byte(fmt.Sprintf("pending:%s", txHash))
	ts := make([]byte, 8)
	binary.BigEndian.PutUint64(ts, uint64(time.Now().UnixNano()))
	return s.db.Put(key, ts, nil)
}

// GetPendingTxs returns all pending transaction hashes
func (s *StateStore) GetPendingTxs() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var txs []string
	iter := s.db.NewIterator(util.BytesPrefix([]byte("pending:")), nil)
	defer iter.Release()

	for iter.Next() {
		txHash := string(iter.Key()[8:]) // Remove "pending:" prefix
		txs = append(txs, txHash)
	}
	return txs, nil
}

// RemovePendingTx removes a transaction from pending queue
func (s *StateStore) RemovePendingTx(txHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := []byte(fmt.Sprintf("pending:%s", txHash))
	return s.db.Delete(key, nil)
}

// GetStats returns store statistics
func (s *StateStore) GetStats() (map[string]uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string] uint64)
	
	// Count state roots
	iter := s.db.NewIterator(util.BytesPrefix([]byte("sr:")), nil)
	for iter.Next() {
		stats["state_roots"]++
	}
	iter.Release()

	// Count batches
	iter = s.db.NewIterator(util.BytesPrefix([]byte("batch:")), nil)
	for iter.Next() {
		stats["batches"]++
	}
	iter.Release()

	// Count transactions
	iter = s.db.NewIterator(util.BytesPrefix([]byte("tx:")), nil)
	for iter.Next() {
		stats["transactions"]++
	}
	iter.Release()

	// Count pending
	iter = s.db.NewIterator(util.BytesPrefix([]byte("pending:")), nil)
	for iter.Next() {
		stats["pending"]++
	}
	iter.Release()

	return stats, nil
}

// IsEnabled returns whether persistence is enabled
func (s *StateStore) IsEnabled() bool {
	return s.cfg != nil && s.cfg.Enabled
}