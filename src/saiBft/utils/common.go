package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

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

type IP struct {
	Query string `json:"query"`
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

func GetOutboundIP() string {
	req, err := http.Get("http://ip-api.com/json/")
	if err != nil {
		return err.Error()
	}
	defer req.Body.Close()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err.Error()
	}

	var ip IP
	json.Unmarshal(body, &ip)

	return ip.Query
}
