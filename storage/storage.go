package storage

type Storage interface {
	AddUser(userID, address string)
	GetUsers() map[string]string
	MarkTxProcessed(txHash string)
	IsTxProcessed(txHash string) bool
	ClearTxProcessed()
	UpdateLastBlock(blockHash string)
	GetLastBlock() string
	GetUser(userID string) (string, bool)
}
