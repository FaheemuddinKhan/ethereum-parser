package handler

import (
	"encoding/json"
	"ethereum-parser/parser"
	"fmt"
	"net/http"
)

type ParserHandler struct {
	parser parser.Parser
}

func NewParserHandler(parser parser.Parser) *ParserHandler {
	return &ParserHandler{parser: parser}
}

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func (h *ParserHandler) GetCurrentBlockHandler(w http.ResponseWriter, r *http.Request) {
	currentBlock := h.parser.GetCurrentBlock()

	h.respond(w, http.StatusOK, Response{
		Success: true,
		Data:    currentBlock,
	})
}

func (h *ParserHandler) SubscribeHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Address string `json:"address"`
	}

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil || payload.Address == "" {
		h.respond(w, http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid payload. Address is required.",
		})
		return
	}

	success := h.parser.Subscribe(payload.Address)
	if success {
		h.respond(w, http.StatusOK, Response{
			Success: true,
			Data:    "Subscription successful",
		})
	} else {
		h.respond(w, http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to subscribe",
		})
	}
}

func (h *ParserHandler) GetTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	address := r.URL.Query().Get("address")
	if address == "" {
		h.respond(w, http.StatusBadRequest, Response{
			Success: false,
			Error:   "Address query parameter is required",
		})
		return
	}

	transactions, err := h.parser.GetTransactions(address)

	if err != nil {
		h.respond(w, http.StatusInternalServerError, Response{
			Success: false,
			Error:   "something went wrong at server side",
		})
	}

	h.respond(w, http.StatusOK, Response{
		Success: true,
		Data:    transactions,
	})
}

func (h *ParserHandler) respond(w http.ResponseWriter, statusCode int, response Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Println("Failed to send response:", err)
	}
}
