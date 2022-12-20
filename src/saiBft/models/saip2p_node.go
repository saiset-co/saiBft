package models

const (
	SyncRequestType  = "sync_request"
	SyncResponseType = "sync_response"
)

// representation of connected saiP2p connected nodes
type SaiP2pNode struct {
	Address string `json:"address"`
}

type SyncRequest struct {
	Type    string `json:"type"`
	From    int    `json:"block_number_from"`
	To      int    `json:"block_number_to"`
	Address string `json:"address"`
}

type SyncResponse struct {
	Type  string `json:"type"`
	Error error  `json:"error"`
	Link  string `json:"link"`
}
