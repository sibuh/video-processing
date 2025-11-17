package initiator

import (
	"backend/database/db"
	"backend/handlers"
	"backend/routing"
	"backend/services"
	"backend/utils"
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/o1egl/paseto"
)

func Init() {
	logger := NewLogger()
	v, err := LoadConfig("./config")
	if err != nil {
		log.Fatal(err)
	}
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		v.Database.User, v.Database.Password,
		v.Database.Host, v.Database.Port,
		v.Database.Name)
	// create connection pool
	pool, err := NewPool(
		context.Background(),
		dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	// run up migration
	if err := RunMigrations("file://./database/schema", v.Database.Name, dsn); err != nil {
		log.Fatal(err)
	}
	logger.Info("migrations run successfully")

	// create enforcer
	enforcer, err := NewEnforcer(pool, logger, "./config")
	if err != nil {
		log.Fatal(err)
	}
	logger.Info("enforcer created successfully")

	tm := utils.NewTokenManager(v.Token.Key,
		v.Token.Duration, *paseto.NewV2())

	db := db.New(pool)

	// services

	userService := services.NewUser(*db, tm)

	// http handlers
	middlewares := handlers.NewMiddleware(tm, enforcer.Enforcer)
	userHandler := handlers.NewUser(userService)

	engine := gin.New()
	engine.Use(middlewares.ErrorMiddleware())
	engine.Use(middlewares.Cors())
	//register http routes
	routing.RegisterRoutes(engine, routing.Handlers{
		UserHandler: userHandler,
		Middlewares: middlewares,
	})

	// run server
	log.Fatal(http.ListenAndServe(":8888", engine))

}
