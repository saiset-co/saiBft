package models

import valid "github.com/asaskevich/govalidator"

// btc keys got from saiBTC
type BtcKeys struct {
	Private string `json:"Private" valid:",required"`
	Public  string `json:"Public" valid:",required"`
	Address string `json:"Address" valid:",required"`
}

// Validate block consensus message
func (m *BtcKeys) Validate() error {
	_, err := valid.ValidateStruct(m)
	return err
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
