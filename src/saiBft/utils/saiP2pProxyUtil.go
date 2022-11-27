package utils

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/iamthe1whoknocks/bft/models"
)

func GetConnectedNodesAddresses(saiP2pProxyAddress string, lastBlockNumber int) ([]string, error) {
	blockRequest := &models.SyncRequest{
		Number: lastBlockNumber,
	}

	b, err := json.Marshal(blockRequest)
	if err != nil {
		return nil, err
	}

	client := http.Client{
		Timeout: 10 * time.Second,
	}

	proxyReq, err := http.NewRequest("POST", saiP2pProxyAddress, bytes.NewBuffer(b))

	resp, err := client.Do(proxyReq)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	syncRespBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	syncResp := models.SyncResponse{}
	err = json.Unmarshal(syncRespBody, &syncResp)
	if err != nil {
		return nil, err
	}

	return syncResp.Addresses, nil

}
