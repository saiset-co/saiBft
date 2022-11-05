package utils

import (
	"bytes"
	"errors"

	"github.com/iamthe1whoknocks/bft/models"
)

// extract result from saiStorage service (crop raw data)
// {"result":[........] --> [.....]}
func ExtractResult(input []byte) ([]byte, error) {
	_, after, found := bytes.Cut(input, []byte(":"))
	if !found {
		return nil, errors.New("wrong result!")
	}

	result := bytes.TrimSuffix(after, []byte("}"))
	return result, nil

}

// detect message type from saiP2p data input
func DetectMsgTypeFromMap(m map[string]interface{}) (string, error) {
	if _, ok := m["block_number"]; ok {
		return models.ConsensusMsgType, nil
	} else if _, ok := m["block_hash"]; ok {
		return models.BlockConsensusMsgType, nil
	} else if _, ok := m["message"]; ok {
		return models.TransactionMsgType, nil
	} else {
		return "", errors.New("unknown msg type")
	}
}
