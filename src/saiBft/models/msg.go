package models

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	valid "github.com/asaskevich/govalidator"
)

const (
	BlockConsensusMsgType = "blockConsensus"
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
	BlockNumber   int      `json:"block_number" valid:",required"`
	Round         int      `json:"round" valid:",required"`
	Messages      []string `json:"messages"`
	Signature     string   `json:"signature" valid:",required"`
	Hash          string   `json:"hash" valid:",required"`
}

// Validate consensus message
func (m *ConsensusMessage) Validate() error {
	_, err := valid.ValidateStruct(m)
	return err
}

// Hashing consensus message
func (m *ConsensusMessage) GetHash() (string, error) {
	b, err := json.Marshal(&ConsensusMessage{
		Type:          m.Type,
		SenderAddress: m.SenderAddress,
		BlockNumber:   m.BlockNumber,
		Round:         m.Round,
		Messages:      m.Messages,
	})
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:]), nil
}

// BlockConsensus message
type BlockConsensusMessage struct {
	Type       string   `json:"type" valid:",required"`
	BlockHash  string   `json:"block_hash" valid:",required"`
	Votes      int      `json:"votes"` // additional field, which was not added by Valeriy
	Block      *Block   `json:"block" valid:",required"`
	Count      int      `json:"-"` // extended value for consensus while get missed blocks from p2p services
	Signatures []string `json:"voted_signatures"`
}

type Block struct {
	Number            int            `json:"number" valid:",required"`
	PreviousBlockHash string         `json:"prev_block_hash" valid:",required"`
	SenderAddress     string         `json:"sender_address" valid:",required"`
	SenderSignature   string         `json:"sender_signature,omitempty" valid:",required"`
	BlockHash         string         `json:"block_hash"`
	Messages          map[string]*Tx `json:"messages"`
}

// Validate block consensus message
func (m *BlockConsensusMessage) Validate() error {
	_, err := valid.ValidateStruct(m)
	return err
}

// Hashing block  message
func (m *Block) GetHash() (string, error) {
	b, err := json.Marshal(&Block{
		Number:            m.Number,
		PreviousBlockHash: m.PreviousBlockHash,
		Messages:          m.Messages,
		SenderAddress:     m.SenderAddress,
	})
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:]), nil
}

// Transaction message
type TransactionMessage struct {
	Type        string      `json:"type" valid:",required"`
	MessageHash string      `json:"message_hash" valid:",required"`
	Tx          *Tx         `json:"message" valid:",required"`
	Votes       int         `json:"votes"` // additional field, which was not added by Valeriy
	VmProcessed bool        `json:"vm_processed"`
	VmResult    bool        `json:"vm_result"`
	VmResponse  interface{} `json:"vm_response"`
	BlockHash   string      `json:"block_hash"`
}

// transaction struct
type Tx struct {
	SenderAddress   string `json:"sender_address" valid:",required"`
	Message         string `json:"message" valid:",required"`
	SenderSignature string `json:"sender_signature" valid:",required"`
	MessageHash     string `json:"message_hash" valid:",required"`
}

// tx message struct
type TxMessage struct {
	Method string   `json:"method" valid:",required"`
	Params []string `json:"params" valid:",required"`
}

// Validate transaction message
func (m *TransactionMessage) Validate() error {
	_, err := valid.ValidateStruct(m)
	return err
}

// Hashing block  message
func (m *Tx) GetHash() (string, error) {
	b, err := json.Marshal(&Tx{
		SenderAddress: m.SenderAddress,
		Message:       m.Message,
	})
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:]), nil
}

type GetBlockMsg struct {
	BCMessage        *BlockConsensusMessage `json:"block_consensus"`
	EqualHashesCount int
}
