package models

// btc keys got from saiBTC
type BtcKeys struct {
	Private string `json:"Private"`
	Public  string `json:"Public"`
	Address string `json:"Address"`
}

// validate signature response from saiBTC
type ValidateSignatureResponse struct {
	Address   string `json:"address"`
	Message   string `json:"message"`
	Signature string `json:"signature"`
}

// sign signature response from saiBTC
type SignMessageResponse struct {
	Message   string `json:"message"`
	Signature string `json:"signature"`
}
