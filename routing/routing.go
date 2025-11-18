package routing

import (
	"net/http"
	"video-processing/handlers"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Handlers struct {
	UserHandler handlers.User
	Middlewares handlers.Middleware
}

func RegisterRoutes(engine *gin.Engine, handlers Handlers) {
	routeMap := []struct {
		method      string
		path        string
		handler     gin.HandlerFunc
		middlewares []gin.HandlerFunc
	}{
		{
			method:      http.MethodGet,
			path:        "/swagger/*any",
			handler:     ginSwagger.WrapHandler(swaggerFiles.Handler),
			middlewares: nil,
		},
		{
			method:      http.MethodPost,
			path:        "/register",
			handler:     handlers.UserHandler.RegisterUser,
			middlewares: nil,
		},
		{
			method:      http.MethodGet,
			path:        "/search",
			handler:     handlers.UserHandler.SearchUsers,
			middlewares: []gin.HandlerFunc{handlers.Middlewares.Authenticate()},
		},
		{
			method:      http.MethodPost,
			path:        "/login",
			handler:     handlers.UserHandler.LoginUser,
			middlewares: nil,
		},
		{
			method:      http.MethodGet,
			path:        "/user",
			handler:     handlers.UserHandler.GetUser,
			middlewares: []gin.HandlerFunc{handlers.Middlewares.Authenticate()},
		},
		{
			method:      http.MethodPatch,
			path:        "/user",
			handler:     handlers.UserHandler.UpdateUser,
			middlewares: []gin.HandlerFunc{handlers.Middlewares.Authenticate()},
		},
	}
	group := engine.Group("v1")
	group.Use(handlers.Middlewares.Cors())
	for _, r := range routeMap {
		group.Handle(r.method, r.path, append(r.middlewares, r.handler)...)
	}
}
