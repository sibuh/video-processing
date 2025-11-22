package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"video-processing/models"
	"video-processing/utils"

	"log/slog"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
)

type Middleware interface {
	Authenticate() gin.HandlerFunc
	Cors() gin.HandlerFunc
	// BeforeWsConnection() gin.HandlerFunc
	ErrorMiddleware() gin.HandlerFunc
}
type middleware struct {
	tm       utils.TokenManager
	enforcer *casbin.Enforcer
	logger   *slog.Logger
}

func NewMiddleware(tm utils.TokenManager, enforcer *casbin.Enforcer, logger *slog.Logger) Middleware {
	return &middleware{
		tm:       tm,
		enforcer: enforcer,
		logger:   logger,
	}
}

func (m *middleware) Authenticate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token := ctx.Request.Header.Get("Authorization")
		if token == "" {
			err := &models.Error{
				Code:        http.StatusUnauthorized,
				Message:     "access denied",
				Description: "access token not found",
				Err:         fmt.Errorf("access token not found"),
			}
			ctx.Error(err)
			ctx.Abort()
			return
		}
		tokenParts := strings.Split(token, " ")
		if tokenParts[0] != "Bearer" {
			er := &models.Error{
				Code:        http.StatusUnauthorized,
				Message:     "access denied",
				Description: "token is not of Bearer type",
				Params:      fmt.Sprintf("token: %s", token),
				Err:         fmt.Errorf("invalid access token: token is not of Bearer type, got: %s", tokenParts[0]),
			}
			ctx.Error(er)
			ctx.Abort()
			return
		}
		if len(tokenParts) != 2 {
			er := &models.Error{
				Code:        http.StatusUnauthorized,
				Message:     "access denied",
				Description: "token format is invalid: expected 'Bearer <token>'",
				Params:      fmt.Sprintf("token: %s", token),
				Err:         fmt.Errorf("invalid access token: expected 'Bearer <token>', got %d parts: %s", len(tokenParts), token),
			}
			ctx.Error(er)
			ctx.Abort()
			return
		}
		payload, err := m.tm.VerifyToken(tokenParts[1])
		if err != nil {
			ctx.Error(err)
			ctx.Abort()
			return
		}

		ctx.Set("user_id", payload.ID)
		ctx.Next()
	}
}

func (m *middleware) Cors() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Header("Access-Control-Allow-Origin", "*")
		ctx.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		ctx.Header("Access-Control-Allow-Headers", "*")
		if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}
		ctx.Next()
	}
}

// ErrorHandlerMiddleware is a Gin middleware to catch and handle custom errors.
func (m *middleware) ErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next() // Process the request

		// Check if any error was attached to the context
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				var Err models.Error
				if errors.As(err.Err, &Err) {
					m.logger.Error(fmt.Sprintf("Code: %d, Message: %s, Description: %s, Params: %s, Err: %v", Err.Code, Err.Message, Err.Description, Err.Params, Err.Err))
					// Send a structured JSON response to the client
					c.JSON(Err.Code, gin.H{
						"ok":    false,
						"data":  nil,
						"error": Err,
					})
					c.Abort() // Abort further handlers if we've sent a response
					return
				} else {
					// This is a general unexpected error
					m.logger.Error(fmt.Sprintf("Code: %d, Message: %s, Description: %s, Params: %s, Err: %v", Err.Code, Err.Message, Err.Description, Err.Params, Err.Err))
					c.JSON(http.StatusInternalServerError, gin.H{
						"ok":    false,
						"data":  nil,
						"error": errors.New("internal server error"),
					})
					c.Abort()
					return
				}
			}
		}
	}
}

func (m *middleware) Authorize() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user_id, exists := ctx.Get("user_id")
		if !exists {
			err := &models.Error{
				Code:    http.StatusUnauthorized,
				Message: "access denied",
				Err:     fmt.Errorf("user id not found"),
			}
			ctx.Error(err)
			ctx.Abort()
			return
		}
		obj := ctx.Request.URL.Path
		act := ctx.Request.Method
		dom := KnowDomain(obj)
		result, err := m.enforcer.Enforce(user_id, dom, obj, act)
		if err != nil {
			err := &models.Error{
				Code:    http.StatusUnauthorized,
				Message: "access denied",
				Err:     fmt.Errorf("access denied"),
			}
			ctx.Error(err)
			ctx.Abort()
			return
		}
		if !result {
			err := &models.Error{
				Code:    http.StatusUnauthorized,
				Message: "access denied",
				Err:     fmt.Errorf("access denied"),
			}
			ctx.Error(err)
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}
func KnowDomain(path string) string {
	// TODO: Implement domain logic based on the path
	return "default"
}
