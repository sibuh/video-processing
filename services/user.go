package services

import (
	"context"
	"fmt"
	"time"
	"video-processing/database/db"
	"video-processing/models"
	"video-processing/utils"

	"github.com/google/uuid"
)

type UserService interface {
	Register(ctx context.Context, input models.UserRegistrationRequest) (models.User, error)
	Login(ctx context.Context, input models.LoginRequest) (models.LoginResponse, error)
	SearchUsers(ctx context.Context, keyword string) ([]models.User, error)
	GetUser(ctx context.Context, uid uuid.UUID) (models.User, error)
	UpdateUser(ctx context.Context, uid uuid.UUID, input models.UpdateUserRequest) (models.User, error)
}

type user struct {
	db           db.Queries
	tokenManager utils.TokenManager
}

func NewUser(db db.Queries, tm utils.TokenManager) UserService {
	return &user{
		db:           db,
		tokenManager: tm,
	}
}

func (u *user) Register(ctx context.Context, arg models.UserRegistrationRequest) (models.User, error) {
	// validate registration request fields
	if err := arg.Validate(); err != nil {
		return models.User{}, models.NewError(err)
	}
	//Hash password before saving
	hash, err := utils.HashPassword(arg.Password)
	if err != nil {
		return models.User{}, models.NewError(err)
	}
	user, err := u.db.CreateUser(ctx, db.CreateUserParams{
		FirstName:  arg.FirstName,
		MiddleName: arg.MiddleName,
		LastName:   arg.LastName,
		Phone:      arg.Phone,
		Username:   arg.Username,
		Password:   hash,
		Email:      arg.Email,
	})
	if err != nil {
		return models.User{}, models.NewError(err)
	}

	return convertDbUserToModelUser(user), nil
}
func convertDbUserToModelUser(user db.User) models.User {
	return models.User{
		ID:                user.ID,
		Username:          user.Username,
		Email:             user.Email,
		Phone:             user.Phone,
		FirstName:         user.FirstName,
		MiddleName:        user.MiddleName,
		LastName:          user.LastName,
		Password:          user.Password,
		ProfilePictureURL: user.ProfilePictureUrl.String,
		CreatedAt:         user.CreatedAt,
		UpdatedAt:         user.UpdatedAt,
		DeletedAt:         user.DeletedAt.Time,
	}
}

func (u *user) Login(ctx context.Context, arg models.LoginRequest) (models.LoginResponse, error) {
	if err := arg.Validate(); err != nil {
		//create custom error
		return models.LoginResponse{}, models.NewError(err)
	}
	// Example: Query user by username (adjust predicate as needed)
	foundUser, err := u.db.GetUserByEmail(ctx, arg.Email)

	if err != nil {
		return models.LoginResponse{}, models.NewError(err)
	}
	if !utils.CheckPassword(foundUser.Password, arg.Password) {
		return models.LoginResponse{}, models.NewError(fmt.Errorf("invalid email or password"))
	}
	token, err := u.tokenManager.CreateToken(utils.Payload{ID: foundUser.ID, IssuedAt: time.Now()})
	if err != nil {
		return models.LoginResponse{}, models.NewError(err)
	}
	foundUser.Password = ""

	return models.LoginResponse{Token: token, User: convertDbUserToModelUser(foundUser)}, nil
}

func (u *user) SearchUsers(ctx context.Context, keyword string) ([]models.User, error) {
	users, err := u.db.SearchUsers(ctx, keyword)
	if err != nil {
		return nil, models.NewError(err)
	}
	var modelUsers []models.User
	for _, user := range users {
		modelUsers = append(modelUsers, convertDbUserToModelUser(user))
	}
	return modelUsers, nil
}
func (u *user) GetUser(ctx context.Context, uid uuid.UUID) (models.User, error) {
	user, err := u.db.GetUser(ctx, uid)
	if err != nil {
		return models.User{}, models.NewError(err)
	}
	user.Password = ""
	return convertDbUserToModelUser(user), nil
}
func (u *user) UpdateUser(ctx context.Context, uid uuid.UUID, input models.UpdateUserRequest) (models.User, error) {
	user, err := u.db.UpdateUser(ctx, db.UpdateUserParams{
		ID:        uid,
		FirstName: input.FirstName,
		LastName:  input.LastName,
		Username:  input.Username,
		Email:     input.Email,
		Phone:     input.Phone,
	})
	if err != nil {
		return models.User{}, models.NewError(err)
	}
	user.Password = ""
	return convertDbUserToModelUser(user), nil
}
