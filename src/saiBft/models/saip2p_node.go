package models

// representation of connected saiP2p connected nodes
type SaiP2pNode struct {
	Address string `json:"address"`
}

type SyncRequest struct {
	From    int    `json:"block_number_from"`
	To      int    `json:"block_number_to"`
	Address string `json:"address"`
}

type SyncResponse struct {
	Blocks []*BlockConsensusMessage `json:"blocks"`
}
