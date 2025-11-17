package main

import (
	_ "backend/docs"
	"backend/initiator"
)

// @title           security management app
// @version         1.0
// @description     API for security management service.
// @termsOfService  http://example.com/terms/

// @contact.name   API Support
// @contact.url    http://www.example.com/support
// @contact.email  support@example.com

// @license.name
// @license.url

// @host      sema.com
// @BasePath  /v1

func main() {
	initiator.Init()
}
