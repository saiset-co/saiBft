package models

// representation of connected saiP2p connected nodes
type SaiP2pNode struct {
	Address string `json:"address"`
}

// request to get missed blocks from connected nodes
type GetMissedBlocksRequest struct {
	LastBlockNumber int `json:"number"`
}
