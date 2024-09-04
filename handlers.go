package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func Handler(c *gin.Context) {

	if strings.HasPrefix(c.Request.URL.Path, "/createMatch") {
		response := CreateMatch(c)
		c.JSON(http.StatusOK, gin.H{"message": response})
		return
	}

	if strings.HasPrefix(c.Request.URL.Path, "/uploadMap") {
		response := UploadMap(c)
		c.JSON(http.StatusOK, gin.H{"message": response})
		return
	}

	if strings.HasPrefix(c.Request.URL.Path, "/pStats") {
		response := PStats(c)
		c.JSON(http.StatusOK, gin.H{"message": response})
		return
	}

	if strings.HasPrefix(c.Request.URL.Path, "/hStats") {
		response := PStatsHero(c)
		c.JSON(http.StatusOK, gin.H{"message": response})
		return
	}

	if strings.HasPrefix(c.Request.URL.Path, "/tStats") {
		response := TStats(c)
		c.JSON(http.StatusOK, gin.H{"message": response})
		return
	}

	if strings.HasPrefix(c.Request.URL.Path, "/tmStats") {
		response := TStatsMap(c)
		c.JSON(http.StatusOK, gin.H{"message": response})
		return
	}

	if strings.HasPrefix(c.Request.URL.Path, "/compareStats") {
		response := CompareStats(c)
		c.JSON(http.StatusOK, gin.H{"message": response})
		return
	}

	if strings.HasPrefix(c.Request.URL.Path, "/updateLeaderboards") {
		response := UpdateLeaderboards()
		c.JSON(http.StatusOK, gin.H{"message": response})
		return
	}

	// Return a 404 Not Found error for unknown endpoints
	c.JSON(http.StatusNotFound, gin.H{
		"error": "Endpoint not found",
	})

}
