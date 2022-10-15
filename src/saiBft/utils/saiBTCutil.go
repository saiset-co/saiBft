package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/iamthe1whoknocks/bft/models"
)

// send and handle response from saiBTC service
func sendRequest(address string, requestBody io.Reader) ([]byte, error) {
	keysRequest, err := http.NewRequest("POST", address, requestBody)
	if err != nil {
		return nil, err
	}
	keysRequest.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(keysRequest)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// get btc keys
func GetBtcKeys(address string) (*models.BtcKeys, []byte, error) {
	requestBody := strings.NewReader("method=generateBTC")

	body, err := sendRequest(address, requestBody)
	if err != nil {
		return nil, nil, err
	}

	keys := models.BtcKeys{}

	err = json.Unmarshal(body, &keys)
	if err != nil {
		return nil, nil, err
	}

	return &keys, body, nil
}

// validate message signature
func ValidateSignature(msg interface{}, address, SenderAddress, signature string) (err error) {
	b := make([]byte, 0)
	switch msg.(type) {
	case *models.BlockConsensusMessage:
		b, err = json.Marshal(msg)
		if err != nil {
			return fmt.Errorf(" marshal blockConsensusMessage : %w", err)
		}
	case *models.ConsensusMessage:
		b, err = json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("marshal ConsensusMessage : %w", err)
		}
	case *models.TransactionMessage:
		b, err = json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("marshal TransactionMessage : %w", err)
		}
	default:
		return errors.New("unknown type of message")
	}
	preparedString := fmt.Sprintf("method=validateSignature&a=%s&signature=%s&message=%s", SenderAddress, signature, string(b))
	requestBody := strings.NewReader(preparedString)

	body, err := sendRequest(address, requestBody)
	if err != nil {
		return fmt.Errorf("sendRequest to saiBTC : %w", err)
	}

	resp := models.ValidateSignatureResponse{}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return fmt.Errorf("unmarshal response from saiBTC  : %w", err)
	}
	if resp.Signature == "valid" {
		return nil
	}
	return errors.New("Signature is not valid")
}

// saiBTC sign message method
func SignMessage(msg interface{}, address, privateKey string) (resp *models.SignMessageResponse, err error) {
	var preparedString string
	switch msg.(type) {
	case *models.BlockConsensusMessage:
		BCMsg := msg.(*models.BlockConsensusMessage)
		data, err := json.Marshal(BCMsg)
		if err != nil {
			return nil, err
		}
		preparedString = fmt.Sprintf("method=signMessage&p=%s&message=%s", privateKey, string(data))
	case *models.ConsensusMessage:
		cMsg := msg.(*models.ConsensusMessage)
		data, err := json.Marshal(cMsg)
		if err != nil {
			return nil, err
		}
		preparedString = fmt.Sprintf("method=signMessage&p=%s&message=%s", privateKey, string(data))
	case *models.TransactionMessage:
		TxMsg := msg.(*models.TransactionMessage)
		data, err := json.Marshal(TxMsg)
		if err != nil {
			return nil, err
		}
		preparedString = fmt.Sprintf("method=signMessage&p=%s&message=%s", privateKey, string(data))
	default:
		return nil, errors.New("unknown type of message")
	}
	requestBody := strings.NewReader(preparedString)

	body, err := sendRequest(address, requestBody)
	if err != nil {
		return nil, fmt.Errorf("sendRequest to saiBTC : %w", err)
	}

	response := models.SignMessageResponse{}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("unmarshal response from saiBTC  : %w\n response body : %s", err, string(body))
	}

	return &response, nil
}
