package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Host          string `yaml:"proxy_host"`
	Port          string `yaml:"proxy_port"`
	ProxyEndpoint string `yaml:"proxy_endpoint"`
	BftHost       string `yaml:"bft_http_host"`
	BftPort       string `yaml:"bft_http_port"`
}

func main() {
	config, err := NewConfig("config.yml")
	if err != nil {
		log.Fatalf("Open config file : ", err)
	}

	r := gin.Default()
	r.Use(gin.Recovery())
	r.Use(cors.Default())
	cfg := cors.DefaultConfig()
	cfg.AllowAllOrigins = true

	r.POST(config.ProxyEndpoint, config.handler)
	r.GET("/check", config.check)
	r.GET("/sync", config.sync)

	r.Run(fmt.Sprintf("%s:%s", config.Host, config.Port))
}

func (cfg *Config) handler(c *gin.Context) {
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

	_, err = http.Post(fmt.Sprintf("http://%s:%s", cfg.BftHost, cfg.BftPort), "application/json", bytes.NewBuffer(b))
	if err != nil {
		c.Status(500)
		log.Println(err)
		c.Writer.Write([]byte(err.Error()))
		return
	}

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

func (cfg *Config) check(c *gin.Context) {
	c.JSON(200, "check ok")
	return
}

func (cfg *Config) sync(c *gin.Context) {
	c.JSON(200, "check ok")
	return
}
