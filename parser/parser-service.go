package parser

import (
	"encoding/json"
	"ethereum-parser/notifier"
	"ethereum-parser/storage"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	ws "github.com/gorilla/websocket"
)

type ParserService struct {
	notifier notifier.Notifier
	storage  storage.Storage
	apiURL   string
	conn     *ws.Conn
}

func NewParserService(storage storage.Storage, apiURL string, notifier notifier.Notifier) Parser {
	return &ParserService{
		notifier: notifier,
		storage:  storage,
		apiURL:   apiURL,
	}
}

func (p *ParserService) GetCurrentBlock() int {
	return hexToInt(p.storage.GetLastBlock())
}

func (p *ParserService) Subscribe(address string) bool {
	// Generate a user ID for simplicity (can be enhanced)
	userID := fmt.Sprintf("user-%s", address)

	// Add the address to storage
	p.storage.AddUser(address, userID)
	return true
}

func (p *ParserService) GetTransactions(address string) ([]Transaction, error) {
	// Initialize an empty slice to store transactions
	var transactions []Transaction

	// Set the URL for the Ethereum RPC API
	apiURL := "https://ethereum-rpc.publicnode.com"

	// Prepare the JSON-RPC request body
	requestBody := fmt.Sprintf(`{
		"jsonrpc": "2.0",
		"method": "eth_getLogs",
		"params": [{
			"fromBlock": "0x0",   // Starting from block 0 (or adjust as needed)
			"toBlock": "latest",  // Up to the latest block
			"address": "%s"
		}],
		"id": 1
	}`, address)

	// Make the HTTP POST request to the Ethereum node
	resp, err := http.Post(apiURL, "application/json", io.NopCloser(strings.NewReader(requestBody)))
	if err != nil {
		// Return the error if the HTTP request fails
		return nil, fmt.Errorf("error fetching logs: %w", err)
	}
	defer resp.Body.Close()

	// Check if the response status is OK
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-OK HTTP status: %s", resp.Status)
	}

	// Define the structure of the response to decode JSON into
	var result struct {
		Result []struct {
			BlockHash       string `json:"blockHash"`
			BlockNumber     string `json:"blockNumber"`
			TransactionHash string `json:"transactionHash"`
			From            string `json:"from"`
			To              string `json:"to"`
			Value           string `json:"value"`
		} `json:"result"`
	}

	// Decode the JSON response into the result struct
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		// Return the error if decoding fails
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	// Process the transactions for the user address
	for _, log := range result.Result {
		transactions = append(transactions, Transaction{
			Hash:    log.TransactionHash,
			From:    log.From,
			To:      log.To,
			Value:   log.Value,
			BlockNo: hexToInt(log.BlockNumber),
		})
	}

	// Return the transactions for the user address
	return transactions, nil
}

// Helper function to convert hex string to integer (block number)
func hexToInt(hexStr string) int {
	blockNumber, err := strconv.ParseInt(hexStr, 0, 64)
	if err != nil {
		log.Printf("Error converting hex to int: %v", err)
		return 0
	}
	return int(blockNumber)
}

func (p *ParserService) fetchTransactionsFromBlock(blockHash string) ([]Transaction, error) {
	// Prepare the JSON-RPC request body
	requestBody := fmt.Sprintf(`{
		"jsonrpc": "2.0",
		"method": "eth_getBlockByNumber",
		"params": ["0x%x", true],
		"id": 1
	}`, blockHash)

	// Make the HTTP POST request to the Ethereum node
	resp, err := http.Post(fmt.Sprintf("https://%v", p.apiURL), "application/json", io.NopCloser(strings.NewReader(requestBody)))
	if err != nil {
		// Return the error if the HTTP request fails
		return nil, fmt.Errorf("error fetching block: %w", err)
	}
	defer resp.Body.Close()

	// Check if the response status is OK
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-OK HTTP status: %s", resp.Status)
	}

	// Define the structure of the response to decode JSON into
	var result struct {
		Result struct {
			Transactions []struct {
				Hash  string `json:"hash"`
				From  string `json:"from"`
				To    string `json:"to"`
				Value string `json:"value"`
			} `json:"transactions"`
		} `json:"result"`
	}

	// Decode the JSON response into the result struct
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		// Return the error if decoding fails
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	// Filter transactions that match the specified address
	var transactions []Transaction
	for _, tx := range result.Result.Transactions {
		transactions = append(transactions, Transaction{
			Hash:    tx.Hash,
			From:    tx.From,
			To:      tx.To,
			Value:   tx.Value,
			BlockNo: hexToInt(blockHash),
		})
	}

	// Return the filtered transactions
	return transactions, nil
}

func (p *ParserService) connect() error {
	var err error
	p.conn, _, err = ws.DefaultDialer.Dial(fmt.Sprintf("wss://%v", p.apiURL), nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %v, %v", err, p.apiURL)
	}
	return nil
}

func (p *ParserService) subscribeToNewHeads() error {
	subscriptionRequest := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_subscribe",
		"params":  []interface{}{"newHeads"},
		"id":      1,
	}

	err := p.conn.WriteJSON(subscriptionRequest)
	if err != nil {
		return fmt.Errorf("failed to subscribe to new heads: %v", err)
	}
	return nil
}

func (p *ParserService) listenForBlocks() {

	for {
		var response map[string]interface{}
		err := p.conn.ReadJSON(&response)
		if err != nil {
			log.Printf("Error reading WebSocket message: %v", err)
			break
		}

		// Handle the block data when received
		if params, ok := response["params"].(map[string]interface{}); ok {
			if result, exists := params["result"].(map[string]interface{}); exists {
				blockHash := result["hash"].(string)

				// Log the block number and hash
				log.Printf("Received new block: (%s)", blockHash)

				lastProcessedBlock := p.storage.GetLastBlock()
				// Step 1: Process transactions of the last block first (if exists)
				if lastProcessedBlock != "" {
					// Fetch all transactions for the last block
					transactions, err := p.fetchTransactionsFromBlock(lastProcessedBlock)
					if err != nil {
						log.Printf("Error fetching transactions for last block %s: %v", lastProcessedBlock, err)
						continue
					}

					// Process all transactions in the last block and clear processedTxs afterward
					for _, tx := range transactions {
						txHash := tx.Hash

						// Skip if already processed
						if processed := p.storage.IsTxProcessed(txHash); processed {
							log.Printf("Transaction %s already processed, skipping.", txHash)
							continue
						}

						// Process the transaction based on user involvement
						fromAddress := tx.From
						toAddress := tx.To

						// Compare 'from' and 'to' addresses with stored users
						if _, exists := p.storage.GetUser(fromAddress); exists {
							log.Printf("Transaction %s involves user %s, notifying.", txHash, fromAddress)
							p.notifier.Notify(txHash, fromAddress)
						} else if _, exists := p.storage.GetUser(toAddress); exists {
							log.Printf("Transaction %s involves user %s, notifying.", txHash, toAddress)
							p.notifier.Notify(txHash, toAddress)
						}

						// Mark the transaction as processed
						p.storage.MarkTxProcessed(txHash)
					}

					// After processing last block's transactions, clear processedTxs for the next block
					log.Println("Finished processing all transactions from last block, clearing processedTxs.")
					p.storage.ClearTxProcessed()
				}

				// Step 2: Now process transactions from the current block
				transactions, err := p.fetchTransactionsFromBlock(blockHash)
				if err != nil {
					log.Printf("Error fetching transactions for block %s: %v", blockHash, err)
					continue
				}

				// Process the transactions from the current block and add them to processedTxs
				for _, tx := range transactions {
					txHash := tx.Hash

					// Process the transaction based on user involvement
					fromAddress := tx.From
					toAddress := tx.To

					// Compare 'from' and 'to' addresses with stored users
					if _, exists := p.storage.GetUser(fromAddress); exists {
						log.Printf("Transaction %s involves user %s, notifying.", txHash, fromAddress)
						p.notifier.Notify(txHash, fromAddress)
					} else if _, exists := p.storage.GetUser(toAddress); exists {
						log.Printf("Transaction %s involves user %s, notifying.", txHash, toAddress)
						p.notifier.Notify(txHash, toAddress)
					}

					// Mark the transaction as processed
					p.storage.MarkTxProcessed(txHash)
				}

				// After processing the current block's transactions, update lastProcessedBlock
				p.storage.UpdateLastBlock(blockHash)
			}
		}
	}
}

func (p *ParserService) Start() {
	// Establish WebSocket connection
	err := p.connect()
	if err != nil {
		log.Fatalf("Error connecting to WebSocket: %v", err)
	}

	// Subscribe to new heads (blocks)
	err = p.subscribeToNewHeads()
	if err != nil {
		log.Fatalf("Error subscribing to new heads: %v", err)
	}

	// Start listening for blocks
	go p.listenForBlocks()
	log.Println("Listening for new blocks")
}
