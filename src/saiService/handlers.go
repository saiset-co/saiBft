package saiService

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"

	"golang.org/x/net/websocket"
)

type Handler map[string]HandlerElement

type HandlerElement struct {
	Name        string // name to execute, can be path
	Description string
	Function    func(interface{}, string) (interface{}, error)
}

type jsonRequestType struct {
	Method string      `json:"method"`
	Data   interface{} `json:"data"`
}

type cliRequestType struct {
	Data string `json:"data"`
}

type j map[string]interface{}

func (s *Service) handleSocketConnections(conn net.Conn) {
	for {
		var message jsonRequestType
		socketMessage, _ := bufio.NewReader(conn).ReadString('\n')

		if socketMessage != "" {
			_ = json.Unmarshal([]byte(socketMessage), &message)

			if message.Method == "" {
				err := j{"Status": "NOK", "Error": "Wrong message format"}
				errBody, _ := json.Marshal(err)
				log.Println(err)
				conn.Write(append(errBody, eos...))
				continue
			}

			result, resultErr := s.processPath(message.Method, message.Data, mode)

			if resultErr != nil {
				err := j{"Status": "NOK", "Error": resultErr.Error()}
				errBody, _ := json.Marshal(err)
				log.Println(err)
				conn.Write(append(errBody, eos...))
				continue
			}

			body, marshalErr := json.Marshal(result)

			if marshalErr != nil {
				err := j{"Status": "NOK", "Error": marshalErr.Error()}
				errBody, _ := json.Marshal(err)
				log.Println(err)
				conn.Write(append(errBody, eos...))
				continue
			}

			conn.Write(append(body, eos...))
		}
	}
}

// handle cli command
func (s *Service) handleCliCommand(path string, data []byte, mode string) ([]byte, error) {

	var message cliRequestType
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data provided")
	}

	err := json.Unmarshal(data, &message)
	if err != nil {
		return nil, err
	}

	result, err := s.processPath(path, message.Data, mode)
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (s *Service) handleWSConnections(conn *websocket.Conn) {
	for {
		var message jsonRequestType
		if rErr := websocket.JSON.Receive(conn, &message); rErr != nil {
			err := j{"Status": "NOK", "Error": "Wrong message format"}
			log.Println(err)
			websocket.JSON.Send(conn, err)
			continue
		}

		if message.Method == "" {
			err := j{"Status": "NOK", "Error": "Wrong message format"}
			log.Println(err)
			websocket.JSON.Send(conn, err)
			continue
		}

		result, resultErr := s.processPath(message.Method, message.Data, mode)

		if resultErr != nil {
			err := j{"Status": "NOK", "Error": resultErr.Error()}
			log.Println(err)
			websocket.JSON.Send(conn, err)
			continue
		}

		sErr := websocket.JSON.Send(conn, result)

		if sErr != nil {
			err := j{"Status": "NOK", "Error": sErr.Error()}
			log.Println(err)
			websocket.JSON.Send(conn, err)
		}
	}
}

func (s *Service) handleHttpConnections(resp http.ResponseWriter, req *http.Request) {
	var message jsonRequestType

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		e := j{"Status": "NOK", "Error": err.Error()}
		errBody, _ := json.Marshal(e)
		log.Println(e)
		resp.Write(errBody)
		return
	}

	decoderErr := json.Unmarshal(body, &message)
	if decoderErr != nil {
		err := j{"Status": "NOK", "Error": decoderErr.Error()}
		errBody, _ := json.Marshal(err)
		log.Println(err)
		resp.Write(errBody)
		return
	}

	if message.Method == "" {
		err := j{"Status": "NOK", "Error": "Wrong message format"}
		errBody, _ := json.Marshal(err)
		log.Println(err)
		resp.Write(errBody)
		return
	}

	result, resultErr := s.processPath(message.Method, message.Data, mode)

	if resultErr != nil {
		err := j{"Status": "NOK", "Error": resultErr.Error()}
		errBody, _ := json.Marshal(err)
		log.Println(err)
		resp.Write(errBody)
		return
	}

	body, marshalErr := json.Marshal(result)

	if marshalErr != nil {
		err := j{"Status": "NOK", "Error": marshalErr.Error()}
		errBody, _ := json.Marshal(err)
		log.Println(err)
		resp.Write(errBody)
		return
	}

	resp.Write(body)
}

func (s *Service) processPath(path string, data interface{}, mode string) (interface{}, error) {
	h, ok := s.Handlers[path]

	if !ok {
		return nil, errors.New("no handler")
	}

	//todo: Rutina na process

	response, err := h.Function(data, mode)
	if err != nil {
		return nil, err
	}
	return response, nil
}
