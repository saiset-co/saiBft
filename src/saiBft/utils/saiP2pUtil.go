package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/iamthe1whoknocks/bft/models"
)

// send direct get block message to connected nodes
func SendDirectGetBlockMsg(node string, req *models.SyncRequest, saiP2pAddress string) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("chain - sendDirectGetBlockMsg - marshal request : %w", err)
	}

	param := url.Values{}
	param.Add("message", string(data))
	param.Add("node", node)

	postRequest, err := http.NewRequest("POST", saiP2pAddress+"/Send_message_to", strings.NewReader(param.Encode()))
	if err != nil {
		return fmt.Errorf("chain - sendDirectGetBlockMsg - create post request : %w", err)
	}

	postRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(postRequest)
	if err != nil {
		return fmt.Errorf("chain - sendDirectGetBlockMsg - send post request : %w", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("chain - sendDirectGetBlockMsg - send post request wrong response status code : %d", resp.StatusCode)
	}

	return nil

}

func GetConnectedNodesAddresses(saiP2Paddress string, blacklist []string) ([]string, error) {
	client := http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("POST", fmt.Sprintf(saiP2Paddress+"/Get_connections_list_txt"), nil)
	if err != nil {
		return nil, fmt.Errorf("chain - handleBlockCandidate - GetBlockchainMissedBlocks - create post request :%w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chain - handleBlockCandidate - GetBlockchainMissedBlocks - do request request :%w", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("chain - handleBlockCandidate - GetBlockchainMissedBlocks - read response body :%w", err)
	}

	defer resp.Body.Close()

	connectedNodes, err := ParseAddresses(string(body))
	if err != nil {
		return nil, fmt.Errorf("chain - handleBlockCandidate - GetBlockchainMissedBlocks - parse response body :%w", err)
	}

	filteredAddresses := make([]string, 0)

	for _, blNode := range blacklist {
		for _, address := range connectedNodes {
			if blNode != address {
				filteredAddresses = append(filteredAddresses, address)
			}
		}
	}
	return filteredAddresses, nil
}

// parse connected addresses from p2p node
func ParseAddresses(body string) (addresses []string, err error) {
	s := strings.TrimPrefix(body, "Connected:\n")

	addressesSplitted := strings.Split(s, "In Progress Queue:")
	if len(addressesSplitted) != 2 {
		log.Println("parse error : wrong data ", addressesSplitted)
		return nil, errors.New("addresses was not found")
	}

	addressesWithNumber := strings.Split(addressesSplitted[0], "\n")

	for _, addressWithNumber := range addressesWithNumber {
		addressSlice := strings.Split(addressWithNumber, "=")
		if len(addressSlice) != 2 {
			continue
		} else {
			addresses = append(addresses, addressSlice[1])
		}
	}
	if len(addresses) == 0 {
		return nil, errors.New("addresses was not found")
	}
	return addresses, nil

}
