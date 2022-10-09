package models

// representation of connected saiP2p connected nodes
type SaiP2pNode struct {
	Address string                   `json:"address"`
	Blocks  []*BlockConsensusMessage // for blocks comparison when missed blocks requested. Temporary value.
}

// request to get missed blocks from connected nodes
type GetBlocksRequest struct {
	LastBlockNumber int `json:"last_block_number"`
}
