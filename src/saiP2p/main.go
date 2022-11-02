package main

import (
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.Use(gin.Recovery())
	cfg := cors.DefaultConfig()
	cfg.AllowAllOrigins = true

	r.POST("send", handler)

	r.Run("127.0.0.1:8071")
}

func handler(c *gin.Context) {
	m := make(map[string]interface{})
	err := c.ShouldBindJSON(&m)
	if err != nil {
		c.Status(500)
		log.Println(err)
		c.Writer.Write([]byte(err.Error()))
		return
	}
	c.JSON(200, "OK")
	log.Printf("Successfuly broadcasted msg : %+v", m)
}
