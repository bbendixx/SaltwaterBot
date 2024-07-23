package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {

	CreateDatabase()

	r := gin.Default()

	r.Use(cors.Default())

	// Define API endpoint
	r.GET("*any", Handler)

	// Start Gin server
	r.Run(":8080")
	
}
