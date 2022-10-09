package models

import (
	valid "github.com/asaskevich/govalidator"
)

const (
	BlockConsensusMsgType = "BlockConsensus"
	ConsensusMsgType      = "сonsensus"
	TransactionMsgType    = "message"
)

// main message, which comes from saiP2P service
type P2pMsg struct {
	Signature string      `json:"signature"`
	Data      interface{} `json:"data"` // can be different type of messages here
}

// Consensus message
type ConsensusMessage struct {
	Type          string   `json:"type" valid:",required"`
	SenderAddress string   `json:"sender_address" valid:",required"`
	Block         int      `json:"block" valid:",required"`
	Round         int      `json:"round" valid:",required"`
	Messages      []string `json:"messages" valid:",required"`
	Signature     string   `json:"signature" valid:",required"`
}

// Validate consensus message
func (m *ConsensusMessage) Validate() error {
	_, err := valid.ValidateStruct(m)
	return err
}

// BlockConsensus message
type BlockConsensusMessage struct {
	Type              string `json:"type" valid:",required"`
	BlockNumber       int    `json:"block" valid:",required"`
	BlockHash         string `json:"block_hash" valid:",required"`
	PreviousBlockHash string `json:"prev_block_hash" valid:",required"`
	SenderAddress     string `json:"sender_address" valid:",required"`
	SenderSignature   string `json:"sender_signature" valid:",required"`
	Votes             int    `json:"votes" valid:",required"` // additional field, which was not added by Valeriy
}

// Validate block consensus message
func (m *BlockConsensusMessage) Validate() error {
	_, err := valid.ValidateStruct(m)
	return err
}

// Transaction message
type TransactionMessage struct {
	MessageHash string `json:"message_hash" valid:",required"`
	Tx          *Tx    `json:"message" valid:",required"`
	Votes       int    `json:"votes"` // additional field, which was not added by Valeriy
}

//
type Tx struct {
	Type            string `json:"type" valid:",required"`
	Block           int    `json:"block" valid:",required"`
	VM              string `json:"vm" valid:",required"`
	SenderAddress   string `json:"sender_address" valid:",required"`
	Message         string `json:"message" valid:",required"`
	SenderSignature string `json:"sender_signature" valid:",required"`
}

// Validate transaction message
func (m *TransactionMessage) Validate() error {
	_, err := valid.ValidateStruct(m)
	return err
}
