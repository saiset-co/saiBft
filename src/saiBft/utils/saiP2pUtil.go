package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/iamthe1whoknocks/bft/models"
)

// send direct get block message to connected nodes
func SendDirectGetBlockMsg(address string, blockNumber int) ([]*models.BlockConsensusMessage, error) {
	getBlocksRequest := &models.GetMissedBlocksRequest{
		LastBlockNumber: blockNumber,
	}

	data, err := json.Marshal(getBlocksRequest)
	if err != nil {
		return nil, fmt.Errorf("chain - sendDirectGetBlockMsg - marshal request : %w", err)
	}

	param := url.Values{}
	param.Add("message", string(data))

	postRequest, err := http.NewRequest("post", address, strings.NewReader(param.Encode()))
	if err != nil {
		return nil, fmt.Errorf("chain - sendDirectGetBlockMsg - create post request : %w", err)
	}

	postRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(postRequest)
	if err != nil {
		return nil, fmt.Errorf("chain - sendDirectGetBlockMsg - send post request : %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("chain - sendDirectGetBlockMsg - send post request wrong response status code : %d", resp.StatusCode)
	}

	blocks := make([]*models.BlockConsensusMessage, 0)

	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("chain - sendDirectGetBlockMsg - send post request - read body from response : %w", err)
	}

	defer resp.Body.Close()

	err = json.Unmarshal(respData, &blocks)
	if err != nil {
		return nil, fmt.Errorf("chain - sendDirectGetBlockMsg - send post request - unmarshal response body : %w", err)
	}
	return blocks, nil

}
