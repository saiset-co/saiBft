package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"saiP2p/utils"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

func (p *Proxy) Handler(c *gin.Context) {
	m := make(map[string]interface{})
	err := c.ShouldBindJSON(&m)
	if err != nil {
		c.Status(500)
		p.Logger.Error("handler - bind json", zap.Error(err))
		c.JSON(http.StatusInternalServerError, err)
		return
	}

	req := jsonRequestType{
		Method: "message",
		Data:   m,
	}
	//log.Printf("create request : %+v\n", req)

	b, err := json.Marshal(req)
	if err != nil {
		c.Status(500)
		p.Logger.Error("handler - marshal request", zap.Error(err))
		c.JSON(http.StatusInternalServerError, err)
		return
	}

	resp, err := http.Post(fmt.Sprintf("http://%s:%s", p.Config.BftHost, p.Config.BftPort), "application/json", bytes.NewBuffer(b))
	if err != nil {
		c.Status(500)
		p.Logger.Error("handler - send post request", zap.Error(err))
		c.JSON(http.StatusInternalServerError, err)
		return
	}

	defer resp.Body.Close()

	//log.Printf("Successfuly broadcasted msg : %+v", m)
}

// 1.handle direct get block messages from nodes
// 2. handle get block messages response
func (p *Proxy) sync(c *gin.Context) {
	// detect msg type it can be syncRequest  (from bft)
	// or syncResponse (from another p2pProxy)
	msg, err := p.detectMsgType(c.Request.Body)
	if err != nil {
		p.Logger.Error("sync - bind json", zap.Error(err))
		return
	}

	p.Logger.Debug("detected msg", zap.Any("msg", msg))

	switch msg.(type) {
	case *SyncResponse:
		syncRespMsg := msg.(*SyncResponse)
		p.Logger.Debug("sync - got sync response", zap.Any("request", syncRespMsg))
		err := p.sendSyncResponseMsg(syncRespMsg)
		if err != nil {
			p.Logger.Error("sync - sync response msg - send", zap.Error(err))
			return
		}
	case *SyncRequest:
		syncReq := msg.(*SyncRequest)
		p.Logger.Debug("sync - got sync request", zap.Any("request", syncReq))
		respAddress, err := p.detectResponseAddress(syncReq.Address)
		if err != nil {
			p.Logger.Error("sync - detect response address", zap.Error(err))
			return
		}
		p.Logger.Debug("got response address", zap.String("response address", respAddress))

		filterGte := bson.M{"block.number": bson.M{"$lte": syncReq.To, "$gte": syncReq.From}}
		err, result := p.Storage.Get("Blockchain", filterGte, bson.M{}, p.Config.StorageToken)
		if err != nil {
			p.Logger.Error("sync - get blocks from storage", zap.Error(err))
			c.JSON(http.StatusInternalServerError, err)
			return
		}

		//handle error
		if len(result) == 2 {
			p.Logger.Error("sync - get blocks from storage", zap.Error(errNoBlocks))
			err = p.sendSyncResponseMsg(&SyncResponse{
				Link:  "",
				Error: errNoBlocks,
			})
			if err != nil {
				p.Logger.Error("sync - sync response msg - send", zap.Error(err))
				return
			}
			return
		}

		data, err := utils.ExtractResult(result)
		if err != nil {
			p.Logger.Error("sync - extract result", zap.Error(err))
			err = p.sendSyncResponseMsg(&SyncResponse{
				Link:  "",
				Error: err,
			})
			if err != nil {
				p.Logger.Error("sync - sync response msg - send", zap.Error(err))
				return
			}
			return
		}

		//p.Logger.Debug("sync - raw result of get blocks from storage", zap.String("body", string(data)))

		blocks := make([]BlockConsensusMessage, 0)

		err = json.Unmarshal(data, &blocks)
		if err != nil {
			p.Logger.Error("sync - unmarshal result into blocks", zap.Error(err))
			err = p.sendSyncResponseMsg(&SyncResponse{
				Link:  "",
				Error: err,
			})
			if err != nil {
				p.Logger.Error("sync - sync response msg - send", zap.Error(err))
				return
			}
		}

		p.Logger.Debug("got requested blocks", zap.Any("blocks", blocks))

		filename := fmt.Sprint(time.Now().Unix())

		file, err := os.OpenFile("./files/"+filename, os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			p.Logger.Error("sync - open file to save", zap.Error(err))
			err = p.sendSyncResponseMsg(&SyncResponse{
				Link:  "",
				Error: err,
			})
			if err != nil {
				p.Logger.Error("sync - sync response msg - send", zap.Error(err))
				return
			}
		}
		_, err = file.Write(data)
		if err != nil {
			p.Logger.Error("sync - write data to file", zap.Error(err))
			err = p.sendSyncResponseMsg(&SyncResponse{
				Link:  "",
				Error: err,
			})
			if err != nil {
				p.Logger.Error("sync - sync response msg - send", zap.Error(err))
				return
			}
		}

		ip := GetOutboundIP()

		blocksFileLink := fmt.Sprintf("http://%s:%s/files/%s", ip, p.Config.Port, filename)
		resp := &SyncResponse{
			Link:  blocksFileLink,
			Error: nil,
		}

		p.Logger.Debug("response created", zap.Any("response", resp))

		err = p.sendDirectMsg(resp, respAddress)
		if err != nil {
			p.Logger.Error("sync - send direct msg", zap.Error(err))
			err = p.sendSyncResponseMsg(&SyncResponse{
				Link:  "",
				Error: err,
			})
			if err != nil {
				p.Logger.Error("sync - sync response msg - send", zap.Error(err))
				return
			}
		}

	}

}

func (p *Proxy) check(c *gin.Context) {
	c.JSON(200, "check ok")
	return
}
