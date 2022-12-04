package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"saiP2p/utils"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"gopkg.in/yaml.v2"
)

type Proxy struct {
	Config  *Config
	Storage *utils.Database
}

type Config struct {
	Host          string `yaml:"proxy_host"`
	Port          string `yaml:"proxy_port"`
	ProxyEndpoint string `yaml:"proxy_endpoint"`
	BftHost       string `yaml:"bft_http_host"`
	BftPort       string `yaml:"bft_http_port"`
	StorageURL    string `yaml:"storage_url"`
}

type SyncRequest struct {
	From    int    `json:"block_number_from"`
	To      int    `json:"block_number_to"`
	Address string `json:"address"`
}

func main() {
	config, err := NewConfig("config.yml")
	if err != nil {
		log.Fatalf("Open config file : ", err)
	}

	storage := NewDB()

	proxy := &Proxy{
		Config:  config,
		Storage: storage,
	}

	r := gin.Default()
	r.Use(gin.Recovery())
	r.Use(cors.Default())
	cfg := cors.DefaultConfig()
	cfg.AllowAllOrigins = true

	r.POST(proxy.ProxyEndpoint, config.handler)
	r.GET("/check", proxy.check)
	r.GET("/sync", proxy.sync)

	r.Run(fmt.Sprintf("%s:%s", config.Host, config.Port))
}

func (p *Proxy) handler(c *gin.Context) {
	m := make(map[string]interface{})
	err := c.ShouldBindJSON(&m)
	if err != nil {
		c.Status(500)
		log.Println(err)
		c.Writer.Write([]byte(err.Error()))
		return
	}

	req := jsonRequestType{
		Method: "message",
		Data:   m,
	}
	log.Printf("create request : %+v\n", req)

	b, err := json.Marshal(req)
	if err != nil {
		c.Status(500)
		log.Println(err)
		c.Writer.Write([]byte(err.Error()))
		return
	}

	resp, err := http.Post(fmt.Sprintf("http://%s:%s", p.Config.BftHost, p.Config.BftPort), "application/json", bytes.NewBuffer(b))
	if err != nil {
		c.Status(500)
		log.Println(err)
		c.Writer.Write([]byte(err.Error()))
		return
	}

	defer resp.Body.Close()

	log.Printf("Successfuly broadcasted msg : %+v", m)
}

type jsonRequestType struct {
	Method string      `json:"method"`
	Data   interface{} `json:"data"`
}

func NewConfig(configPath string) (*Config, error) {
	// Create config structure
	config := &Config{}

	// Open config file
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Init new YAML decode
	d := yaml.NewDecoder(file)

	// Start YAML decoding from file
	if err := d.Decode(&config); err != nil {
		return nil, err
	}

	return config, nil
}

func NewDB() utils.Database {
	url, ok := Service.GlobalService.Configuration["storage_url"].(string)
	if !ok {
		log.Fatalf("configuration : invalid storage url provided, url : %s", Service.GlobalService.Configuration["storage_url"])
	}
	email, ok := Service.GlobalService.Configuration["storage_email"].(string)
	if !ok {
		log.Fatalf("configuration : invalid storage email provided, email : %s", Service.GlobalService.Configuration["storage_email"])
	}
	password, ok := Service.GlobalService.Configuration["storage_password"].(string)
	if !ok {
		log.Fatalf("configuration : invalid storage password provided, password : %s", Service.GlobalService.Configuration["storage_email"])
	}

	return &utils.Storage(url, email, password)
}

func (p *Proxy) check(c *gin.Context) {
	c.JSON(200, "check ok")
	return
}

func (p *Proxy) sync(c *gin.Context) {
	syncReq := &SyncRequest{}
	err := c.ShouldBindJSON(syncReq)
	if err != nil {
		c.Status(500)
		log.Println(err)
		c.Writer.Write([]byte(err.Error()))
	}
	criteria := bson.M{}
	err, result := p.Storage.Get("Blockchain")

}
