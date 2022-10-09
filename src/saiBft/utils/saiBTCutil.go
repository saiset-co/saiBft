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
func GetBtcKeys(address string) (*models.BtcKeys, error) {
	requestBody := strings.NewReader("method=generateBTC")

	body, err := sendRequest(address, requestBody)
	if err != nil {
		return nil, err
	}

	keys := models.BtcKeys{}

	err = json.Unmarshal(body, &keys)
	if err != nil {
		return nil, err
	}

	return &keys, nil
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
	b := make([]byte, 0)
	switch msg.(type) {
	case *models.BlockConsensusMessage:
		b, err = json.Marshal(msg)
		if err != nil {
			return nil, fmt.Errorf(" marshal blockConsensusMessage : %w", err)
		}
	case *models.ConsensusMessage:
		b, err = json.Marshal(msg)
		if err != nil {
			return nil, fmt.Errorf("marshal ConsensusMessage : %w", err)
		}
	case *models.TransactionMessage:
		b, err = json.Marshal(msg)
		if err != nil {
			return nil, fmt.Errorf("marshal TransactionMessage : %w", err)
		}
	default:
		return nil, errors.New("unknown type of message")
	}
	preparedString := fmt.Sprintf("method=signMessage&p=%s&message=%s", privateKey, string(b))
	requestBody := strings.NewReader(preparedString)

	body, err := sendRequest(address, requestBody)
	if err != nil {
		return nil, fmt.Errorf("sendRequest to saiBTC : %w", err)
	}

	response := models.SignMessageResponse{}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, fmt.Errorf("unmarshal response from saiBTC  : %w\n response body : %s", err, string(body))
	}
	return &response, nil
}
