package handlers

import (
	"fmt"
	"net/http"

	"video-processing/models"
	"video-processing/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type User interface {
	RegisterUser(ctx *gin.Context)
	LoginUser(ctx *gin.Context)
	SearchUsers(ctx *gin.Context)
	GetUser(ctx *gin.Context)
	UpdateUser(ctx *gin.Context)
}
type user struct {
	userService services.UserService
}

func NewUser(us services.UserService) User {
	return &user{
		userService: us,
	}
}

// RegisterUser registers a new user.
// @Summary Register a new user
// @Description Register a new user with the input payload
// @Tags users
// @Accept  json
// @Produce  json
// @Param   user  body    models.UserRegistrationRequest  true  "User payload"
// @Success 201 {object} models.User
// @Failure 400 {object} map[string]string
// @Router /v1/users [post]
func (u *user) RegisterUser(ctx *gin.Context) {
	var urr = models.UserRegistrationRequest{}
	if err := ctx.ShouldBindJSON(&urr); err != nil {
		er := &models.Error{
			Code:    http.StatusBadRequest,
			Message: "failed to bind request data",
			Err:     err,
		}
		ctx.Error(er)
		return
	}
	usr, err := u.userService.Register(ctx, urr)
	if err != nil {
		ctx.Error(err)
		return
	}
	usr.Password = ""
	ctx.JSON(http.StatusCreated, gin.H{
		"ok":    true,
		"data":  usr,
		"error": nil,
	})

}

// LoginUser logs in a user.
// @Summary Login a user
// @Description Login a user with the input payload
// @Tags users
// @Accept  json
// @Produce  json
// @Param   user  body    models.LoginRequest  true  "User payload"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]any
// @Router /v1/users/login [post]
func (u *user) LoginUser(ctx *gin.Context) {
	var lr = models.LoginRequest{}
	if err := ctx.ShouldBindJSON(&lr); err != nil {
		err := &models.Error{
			Code:    http.StatusBadRequest,
			Message: "failed to bind request data",
			Err:     err,
		}
		ctx.Error(err)
		return
	}
	res, err := u.userService.Login(ctx, lr)
	if err != nil {
		ctx.Error(err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"ok":    true,
		"data":  res,
		"error": nil,
	})

}

// SearchUsers searches for users.
// @Summary Search for users
// @Description Search for users with the input payload
// @Tags users
// @Accept  json
// @Produce  json
// @Param   user  body    models.User  true  "User payload"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]any
// @Router /v1/users/search [get]
// @Security BearerAuth
func (u *user) SearchUsers(ctx *gin.Context) {
	keyword := ctx.Query("keyword")
	users, err := u.userService.SearchUsers(ctx, keyword)
	if err != nil {
		ctx.Error(err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"ok":    true,
		"data":  users,
		"error": nil,
	})
}

// GetUser gets a user.
// @Summary Get a user
// @Description Get a user with the input payload
// @Tags users
// @Accept  json
// @Produce  json
// @Param   user  body    models.User  true  "User payload"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]any
// @Router /v1/users [get]
// @Security BearerAuth
func (u *user) GetUser(ctx *gin.Context) {
	uid, ok := ctx.Value("user_id").(uuid.UUID)
	if !ok {
		err := &models.Error{
			Code:    http.StatusUnauthorized,
			Message: "failed to get user_id from context",
			Err:     fmt.Errorf("user_id not found in context"),
		}
		ctx.Error(err)
		return
	}
	user, err := u.userService.GetUser(ctx, uid)
	if err != nil {
		ctx.Error(err)
		return
	}
	user.Password = ""
	ctx.JSON(http.StatusOK, gin.H{
		"ok":    true,
		"data":  user,
		"error": nil,
	})
}

// UpdateUser updates a user.
// @Summary Update a user
// @Description Update a user with the input payload
// @Tags users
// @Accept  json
// @Produce  json
// @Param   user  body    models.UpdateUserRequest  true  "User payload"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]any
// @Router /v1/users [patch]
// @Security BearerAuth
func (u *user) UpdateUser(ctx *gin.Context) {
	uid, ok := ctx.Value("user_id").(uuid.UUID)
	if !ok {
		err := &models.Error{
			Code:    http.StatusUnauthorized,
			Message: "failed to get user_id from context",
			Err:     fmt.Errorf("user_id not found in context"),
		}
		ctx.Error(err)
		return
	}
	var urr = models.UpdateUserRequest{}
	if err := ctx.ShouldBindJSON(&urr); err != nil {
		err := &models.Error{
			Code:    http.StatusBadRequest,
			Message: "failed to bind request data",
			Err:     err,
		}
		ctx.Error(err)
		return
	}
	user, err := u.userService.UpdateUser(ctx, uid, urr)
	if err != nil {
		ctx.Error(err)
		return
	}
	user.Password = ""
	ctx.JSON(http.StatusOK, gin.H{
		"ok":    true,
		"data":  user,
		"error": nil,
	})
}
