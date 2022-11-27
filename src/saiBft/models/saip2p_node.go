package models

// representation of connected saiP2p connected nodes
type SaiP2pNode struct {
	Address string `json:"address"`
}

type SyncRequest struct {
	Number int `json:"block_number"`
}

type SyncResponse struct {
	Addresses []string `json:"addresses"`
}
