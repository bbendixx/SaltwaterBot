package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func Handler(c *gin.Context) {

	if strings.HasPrefix(c.Request.URL.Path, "/createMatch") {
		CreateMatch(c)
		return
	}

	if strings.HasPrefix(c.Request.URL.Path, "/createMap") {
		SaveMapStats(c)
		return
	}

	if strings.HasPrefix(c.Request.URL.Path, "/getPlayerStats") {
		GetPlayerStats(c)
		return
	}

	// Return a 404 Not Found error for unknown endpoints
	c.JSON(http.StatusNotFound, gin.H{
		"error": "Endpoint not found",
	})

}

