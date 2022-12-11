package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v2"
)

func NewConfig(configPath string) (*Config, error) {
	config := &Config{}

	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	d := yaml.NewDecoder(file)

	// Start YAML decoding from file
	if err := d.Decode(&config); err != nil {
		return nil, err
	}

	return config, nil
}

func (p *Proxy) SetLogger() {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	option := zap.AddStacktrace(zap.DPanicLevel)
	logger, err := config.Build(option)
	if err != nil {
		log.Fatal("error creating logger : ", err.Error())
	}
	logger.Debug("Logger started", zap.String("mode", "debug"))
	p.Logger = logger
}

func (p *Proxy) createFileDirectory() error {
	path := "files"
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(path, os.ModePerm)
		if err != nil {
			log.Println(err)
			return err
		}
	}
	return nil
}

// detect address to send response via saiP2p
func (p *Proxy) detectResponseAddress(address string) (string, error) {
	client := http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:%s/Get_connections_list_txt", p.Config.P2pHost, p.Config.P2pPort), nil)
	if err != nil {
		return "", fmt.Errorf("create post request :%w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request request :%w", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body :%w", err)
	}

	defer resp.Body.Close()

	connectedNodes, err := p.ParseAddresses(string(body))
	if err != nil {
		return "", fmt.Errorf("parse response body :%w", err)
	}
	p.Logger.Debug("sync - connected nodes found", zap.Strings("connected nodes", connectedNodes), zap.Int("count", len(connectedNodes)))

	for _, addr := range connectedNodes {
		if strings.HasPrefix(addr, address) {
			return addr, nil
		}
	}
	return "", errConnectedAddrNotFound
}

// parse connected addresses from p2p node
func (p *Proxy) ParseAddresses(body string) (addresses []string, err error) {
	s := strings.TrimPrefix(body, "Connected:\n")
	//log.Println("prefix trimmed : ", s)

	addressesSplitted := strings.Split(s, "In Progress Queue:")
	if len(addressesSplitted) != 2 {
		log.Println("parse error : wrong data ", addressesSplitted)
		return nil, errors.New("addresses was not found")
	}

	//log.Println("suffix trimmed : ", addressesSplitted[0])

	addressesWithNumber := strings.Split(addressesSplitted[0], "\n")

	for _, addressWithNumber := range addressesWithNumber {
		addressSlice := strings.Split(addressWithNumber, "=")
		if len(addressSlice) != 2 {
			log.Println("wrong address : ", addressWithNumber)
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

func (p *Proxy) sendDirectMsg(syncResp *SyncResponse, address string) error {
	data, err := json.Marshal(syncResp)
	if err != nil {
		return fmt.Errorf(" marshal request : %w", err)
	}

	param := url.Values{}
	param.Add("message", string(data))
	param.Add("node", address)

	postRequest, err := http.NewRequest("POST", fmt.Sprintf("http://%s:%s/Send_message_to", p.Config.P2pHost, p.Config.P2pPort), strings.NewReader(param.Encode()))
	if err != nil {
		return fmt.Errorf("create post request : %w", err)
	}

	postRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := http.Client{
		Timeout: 3 * time.Second,
	}

	resp, err := client.Do(postRequest)
	if err != nil {
		return fmt.Errorf("send post request : %w", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("send post request wrong response status code : %d", resp.StatusCode)
	}

	return nil
}

// detect incoming msg type from p2p on receive msg callback
func (p *Proxy) detectMsgType(body io.ReadCloser) (interface{}, error) {
	data, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("read body : %w", err)
	}

	m := make(map[string]interface{})

	err = json.Unmarshal(data, &m)
	if err != nil {
		return nil, fmt.Errorf("unmarshal body : %w", err)
	}

	if m["block_number_to"] != 0 && m["address"] != "" {
		syncReq := SyncRequest{}
		err = json.Unmarshal(data, &syncReq)
		if err != nil {
			return nil, fmt.Errorf("unmarshal body to syncRequest: %w", err)
		}
		return &syncReq, nil
	} else if m["link"] != "" {
		syncResp := SyncResponse{}
		err = json.Unmarshal(data, &syncResp)
		if err != nil {
			return nil, fmt.Errorf("unmarshal body to syncResp: %w", err)
		}
		return &syncResp, nil
	} else {
		p.Logger.Error(errWrongMsgType.Error(), zap.Any("msg", m))
		return nil, errWrongMsgType
	}
}

func (p *Proxy) sendSyncResponseMsg(msg *SyncResponse) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal msg : %w", err)
	}
	req := jsonRequestType{
		Method: "GetMissedBlocksResponse",
		Data:   data,
	}

	sendData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal send data : %w", err)
	}

	client := http.Client{
		Timeout: 3 * time.Second,
	}

	newReq, err := http.NewRequest("POST", fmt.Sprintf("http://%s:%s", p.Config.BftHost, p.Config.BftPort), bytes.NewBuffer(sendData))
	if err != nil {
		p.Logger.Error("handler - create post request", zap.Error(err))
		return fmt.Errorf("create post request : %w", err)
	}

	resp, err := client.Do(newReq)
	if err != nil {
		p.Logger.Error("handler - send post request", zap.Error(err))
		return fmt.Errorf("post msg : %w", err)
	}

	defer resp.Body.Close()

	return nil
}
