package saiUtil

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

func Send(url string, data io.Reader, token string) (error, []byte) {
	req, err := http.NewRequest("POST", url, data)

	if err != nil {
		fmt.Println("Database error: ", err)
		return err, []byte("")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Token", token)

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		fmt.Println("Database error: ", err)
		return err, []byte("")
	}

	defer resp.Body.Close()
	_ = time.AfterFunc(5*time.Second, func() {
		resp.Body.Close()
	})
	body, _ := ioutil.ReadAll(resp.Body)
	return nil, body
}
