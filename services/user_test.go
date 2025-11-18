package services_test

import (
	"context"
	"fmt"
	"log"
	"testing"
	"video-processing/database/db"
	"video-processing/initiator"
	"video-processing/models"
	"video-processing/services"

	"video-processing/utils"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/o1egl/paseto"
	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	instance, cleanup := InitTestDB()
	defer cleanup()
	db := db.New(instance.pool)

	// Clean up any existing data
	instance.pool.Exec(context.Background(), "TRUNCATE TABLE users CASCADE")

	u := services.NewUser(*db, instance.tm)
	testCases := []struct {
		name  string
		input models.UserRegistrationRequest
		want  models.User
		error error
	}{
		{
			name: "success",
			input: models.UserRegistrationRequest{
				FirstName:  "Girma",
				MiddleName: "tesfaye",
				LastName:   "ngusu",
				Username:   "gimmy",
				Phone:      "0912345678",
				Email:      "gimmy@gmail.com",
				Password:   "test123",
			},
			want: models.User{
				FirstName:  "Girma",
				MiddleName: "tesfaye",
				LastName:   "ngusu",
				Username:   "gimmy",
				Phone:      "0912345678",
				Email:      "gimmy@gmail.com",
			},
			error: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := u.Register(context.Background(), tc.input)
			require.NoError(t, err)
			require.NotEmpty(t, out.ID)
			require.Equal(t, tc.want.FirstName, out.FirstName)
			require.Equal(t, tc.want.MiddleName, out.MiddleName)
			require.Equal(t, tc.want.LastName, out.LastName)
			require.Equal(t, tc.want.Username, out.Username)
			require.Equal(t, tc.want.Phone, out.Phone)
			require.Equal(t, tc.want.Email, out.Email)

		})
	}
}
func InitTestDB() (struct {
	pool *pgxpool.Pool
	tm   utils.TokenManager
}, func()) {
	v, err := loadConfig("../../config")
	if err != nil {
		log.Fatal(err)
	}
	testDbName := utils.RandomString(10)
	maintenanceDbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		v.Database.User, v.Database.Password,
		v.Database.Host, v.Database.Port,
		"postgres")

	testDbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		v.Database.User, v.Database.Password,
		v.Database.Host, v.Database.Port,
		testDbName)

	ctx := context.Background()

	conn, err := pgx.Connect(ctx, maintenanceDbURL)
	if err != nil {
		log.Fatal(err)
	}

	_, err = conn.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS \"%s\"", testDbName))
	if err != nil {
		log.Fatal(err)
	}

	_, err = conn.Exec(ctx, fmt.Sprintf("CREATE DATABASE \"%s\"", testDbName))
	if err != nil {
		log.Fatal(err)
	}

	err = getMigrations("file://../../database/schema", testDbName, testDbURL)
	if err != nil {
		conn.Close(ctx)
		log.Fatal(err)
	}

	pool, err := initiator.NewPool(ctx, testDbURL)
	if err != nil {
		log.Fatal(err)
	}
	tm := utils.NewTokenManager(v.Token.Key, v.Token.Duration, *paseto.NewV2())
	return struct {
			pool *pgxpool.Pool
			tm   utils.TokenManager
		}{
			pool: pool,
			tm:   tm,
		}, func() {
			_, err = conn.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS \"%s\"", testDbName))
			if err != nil {
				log.Printf("Warning: failed to drop test db %s: %v", testDbName, err)
			}
			conn.Close(ctx)
			pool.Close()
		}
}
func TestLogin(t *testing.T) {
	instance, cleanup := InitTestDB()
	defer cleanup()

	ctx := context.Background()
	db := db.New(instance.pool)

	// Clean up any existing data
	instance.pool.Exec(ctx, "TRUNCATE TABLE users CASCADE")

	u := services.NewUser(*db, instance.tm)

	// Register a user first
	registrationInput := models.UserRegistrationRequest{
		FirstName:  "John",
		MiddleName: "Doe",
		LastName:   "Smith",
		Username:   "johnsmith",
		Phone:      "0911223344",
		Email:      "john@example.com",
		Password:   "password123",
	}
	_, err := u.Register(ctx, registrationInput)
	require.NoError(t, err)

	testCases := []struct {
		name        string
		input       models.LoginRequest
		expectError bool
	}{
		{
			name: "successful login",
			input: models.LoginRequest{
				Email:    "john@example.com",
				Password: "password123",
			},
			expectError: false,
		},
		{
			name: "invalid password",
			input: models.LoginRequest{
				Email:    "john@example.com",
				Password: "wrongpassword",
			},
			expectError: true,
		},
		{
			name: "user not found",
			input: models.LoginRequest{
				Email:    "nonexistent@example.com",
				Password: "password123",
			},
			expectError: true,
		},
		{
			name: "missing email",
			input: models.LoginRequest{
				Email:    "",
				Password: "password123",
			},
			expectError: true,
		},
		{
			name: "missing password",
			input: models.LoginRequest{
				Email:    "john@example.com",
				Password: "",
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := u.Login(ctx, tc.input)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, out.Token)
				require.Equal(t, tc.input.Email, out.User.Email)
				require.Empty(t, out.User.Password) // Password should be cleared
			}
		})
	}

	defer func() {
		instance.pool.Exec(ctx, "TRUNCATE TABLE users")
	}()
}

func TestGetUser(t *testing.T) {
	instance, cleanup := InitTestDB()
	defer cleanup()
	db := db.New(instance.pool)
	ctx := context.Background()
	// Clean up any existing data
	instance.pool.Exec(ctx, "TRUNCATE TABLE users CASCADE")

	u := services.NewUser(*db, instance.tm)

	// Register a user first
	registrationInput := models.UserRegistrationRequest{
		FirstName:  "Alice",
		MiddleName: "Marie",
		LastName:   "Johnson",
		Username:   "alicejohnson",
		Phone:      "0922334455",
		Email:      "alice@example.com",
		Password:   "alice123",
	}
	registeredUser, err := u.Register(context.Background(), registrationInput)
	require.NoError(t, err)

	testCases := []struct {
		name        string
		userID      uuid.UUID
		expectError bool
	}{
		{
			name:        "get existing user",
			userID:      registeredUser.ID,
			expectError: false,
		},
		{
			name:        "get non-existent user",
			userID:      uuid.New(),
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := u.GetUser(ctx, tc.userID)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.userID, out.ID)
				require.Equal(t, registrationInput.FirstName, out.FirstName)
				require.Equal(t, registrationInput.Email, out.Email)
				require.Empty(t, out.Password) // Password should be cleared
			}
		})
	}

}

func TestUpdateUser(t *testing.T) {
	instance, cleanup := InitTestDB()
	defer cleanup()
	db := db.New(instance.pool)
	ctx := context.Background()
	// Clean up any existing data
	instance.pool.Exec(ctx, "TRUNCATE TABLE users CASCADE")

	u := services.NewUser(*db, instance.tm)

	// Register a user first
	registrationInput := models.UserRegistrationRequest{
		FirstName:  "Bob",
		MiddleName: "James",
		LastName:   "Williams",
		Username:   "bobwilliams",
		Phone:      "0933445566",
		Email:      "bob@example.com",
		Password:   "bob123",
	}
	registeredUser, err := u.Register(ctx, registrationInput)
	require.NoError(t, err)

	testCases := []struct {
		name        string
		userID      uuid.UUID
		input       models.UpdateUserRequest
		expectError bool
	}{
		{
			name:   "update user successfully",
			userID: registeredUser.ID,
			input: models.UpdateUserRequest{
				FirstName: "Bobby",
				LastName:  "Williams-Jr",
				Phone:     "0944556677",
				Username:  "bobbyjr",
				Email:     "bobby@example.com",
			},
			expectError: false,
		},
		{
			name:   "update non-existent user",
			userID: uuid.New(),
			input: models.UpdateUserRequest{
				FirstName: "Test",
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := u.UpdateUser(ctx, tc.userID, tc.input)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.userID, out.ID)
				require.Equal(t, tc.input.FirstName, out.FirstName)
				require.Equal(t, tc.input.LastName, out.LastName)
				require.Equal(t, tc.input.Phone, out.Phone)
				require.Equal(t, tc.input.Username, out.Username)
				require.Equal(t, tc.input.Email, out.Email)
				require.Empty(t, out.Password) // Password should be cleared
			}
		})
	}

}

func TestSearchUsers(t *testing.T) {
	instance, cleanup := InitTestDB()
	defer cleanup()
	ctx := context.Background()

	db := db.New(instance.pool)

	// Clean up any existing data
	instance.pool.Exec(ctx, "TRUNCATE TABLE users CASCADE")

	u := services.NewUser(*db, instance.tm)

	// Register multiple users
	users := []models.UserRegistrationRequest{
		{
			FirstName:  "Charlie",
			MiddleName: "Brown",
			LastName:   "Davis",
			Username:   "charliedavis",
			Phone:      "0955667788",
			Email:      "charlie@example.com",
			Password:   "charlie123",
		},
		{
			FirstName:  "David",
			MiddleName: "Lee",
			LastName:   "Martinez",
			Username:   "davidmartinez",
			Phone:      "0966778899",
			Email:      "david@example.com",
			Password:   "david123",
		},
		{
			FirstName:  "Eva",
			MiddleName: "Grace",
			LastName:   "Taylor",
			Username:   "evataylor",
			Phone:      "0977889900",
			Email:      "eva@example.com",
			Password:   "eva123",
		},
	}

	for _, userInput := range users {
		_, err := u.Register(ctx, userInput)
		require.NoError(t, err)
	}

	testCases := []struct {
		name            string
		keyword         string
		expectedMinSize int
	}{
		{
			name:            "search by first name",
			keyword:         "Charlie",
			expectedMinSize: 1,
		},
		{
			name:            "search by email",
			keyword:         "david@example.com",
			expectedMinSize: 1,
		},
		{
			name:            "search by username",
			keyword:         "evataylor",
			expectedMinSize: 1,
		},
		{
			name:            "search with no results",
			keyword:         "nonexistentuser",
			expectedMinSize: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := u.SearchUsers(ctx, tc.keyword)
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(results), tc.expectedMinSize)
		})
	}
}
