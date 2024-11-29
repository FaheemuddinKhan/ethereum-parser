package parser

type Parser interface {
	// Last parsed block
	GetCurrentBlock() int

	// Add address to observer
	Subscribe(address string) bool

	// List of inbound or outbound transactions for an address
	GetTransactions(address string) ([]Transaction, error)

	Start()
}

type Transaction struct {
	Hash    string
	From    string
	To      string
	Value   string
	BlockNo int
}
