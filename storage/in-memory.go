package storage

import (
	"sync"
)

type InMemoryStorage struct {
	mu           sync.RWMutex
	users        map[string]string
	processedTxs map[string]bool
	lastBlock    string
}

func NewInMemoryStorage() Storage {
	return &InMemoryStorage{
		users:        make(map[string]string),
		processedTxs: make(map[string]bool),
		lastBlock:    "",
	}
}

func (ms *InMemoryStorage) AddUser(address, userID string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.users[address] = userID
}

func (ms *InMemoryStorage) GetUsers() map[string]string {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	// Return a copy to avoid external modification
	usersCopy := make(map[string]string)
	for k, v := range ms.users {
		usersCopy[k] = v
	}
	return usersCopy
}

// Mark a transaction as processed
func (ms *InMemoryStorage) MarkTxProcessed(txHash string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.processedTxs[txHash] = true
}

// Check if a transaction is already processed
func (ms *InMemoryStorage) IsTxProcessed(txHash string) bool {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return ms.processedTxs[txHash]
}

// Update the last processed block number
func (ms *InMemoryStorage) UpdateLastBlock(blockHash string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.lastBlock = blockHash
}

func (ms *InMemoryStorage) GetLastBlock() string {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return ms.lastBlock
}

func (ms *InMemoryStorage) GetUser(userID string) (string, bool) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	address, exists := ms.users[userID]
	return address, exists
}

func (ms *InMemoryStorage) ClearTxProcessed() {
	ms.processedTxs = make(map[string]bool)
}
