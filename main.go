package main

import (
	_ "video-processing/docs"
	"video-processing/initiator"
)

// @title           video processing app
// @version         1.0
// @description     Web app built with golang using gin framework for video processing service.
// @termsOfService  http://example.com/terms/

// @contact.name   API Support
// @contact.url    http://www.example.com/support
// @contact.email  support@example.com

// @license.name
// @license.url

// @host      localhost:8080
// @BasePath  /v1

func main() {
	initiator.Init()
}
