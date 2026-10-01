package ui

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func noopHandler(c *gin.Context) {}

func LoadRouter(router gin.IRouter) {
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl", gin.H{})
	})
	router.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.tmpl", gin.H{})
	})
	router.GET("/register", func(c *gin.Context) {
		c.HTML(http.StatusOK, "register.tmpl", gin.H{})
	})
	router.GET("/workers", func(c *gin.Context) {
		c.HTML(http.StatusOK, "workers.tmpl", gin.H{})
	})
	router.GET("/jobs", func(c *gin.Context) {
		c.HTML(http.StatusOK, "jobs.tmpl", gin.H{})
	})
	router.GET("/jobs/create", func(c *gin.Context) {
		c.HTML(http.StatusOK, "create_job.tmpl", gin.H{})
	})
	router.GET("/ssh/:workerId", func(c *gin.Context) {
		c.HTML(http.StatusOK, "ssh.tmpl", gin.H{})
	})
	router.GET("/ssh/:workerId/ws", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	})
}
