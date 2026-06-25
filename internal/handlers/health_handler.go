package handlers

import "github.com/gin-gonic/gin"

func Health(c *gin.Context) {
	ok(c, gin.H{"status": "ok"})
}
