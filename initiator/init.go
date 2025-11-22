package initiator

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"video-processing/database/db"
	"video-processing/handlers"
	"video-processing/routing"
	"video-processing/services"
	"video-processing/utils"

	"github.com/gin-gonic/gin"
	"github.com/o1egl/paseto"
)

func Init() {
	logger := NewLogger()
	config, err := LoadConfig("./config")
	if err != nil {
		log.Fatal(err)
	}
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		config.Database.User, config.Database.Password,
		config.Database.Host, config.Database.Port,
		config.Database.Name)
	// create connection pool
	pool, err := NewPool(
		context.Background(),
		dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	// run up migration
	if err := RunMigrations("file://./database/schema", config.Database.Name, dsn); err != nil {
		log.Fatal(err)
	}
	logger.Info("migrations run successfully")

	// create enforcer
	enforcer, err := NewEnforcer(pool, logger, "./config")
	if err != nil {
		log.Fatal(err)
	}
	logger.Info("enforcer created successfully")

	tm := utils.NewTokenManager(config.Token.Key,
		config.Token.Duration, *paseto.NewV2())

	db := db.New(pool)
	// init redis
	redisClient := NewRedisClient(logger, config)
	// init minio client
	minioClient := InitMinio(logger, config)
	// init streamer
	streamer := services.NewRedisStreamer("video_stream", logger, redisClient)
	// init consumer and run it in a separate goroutine
	consumer := services.NewRedisConsumer("video_stream", "video_group", "video_consumer_1", logger, redisClient, minioClient)
	go func() {
		if err := consumer.Consume(context.Background()); err != nil {
			logger.Error("❌ Consumer error", "error", err)
		}
	}()

	// services
	userService := services.NewUser(*db, tm)
	videoService := services.NewVideoProcessor(logger, minioClient, db, streamer, config.Minio.UrlExpiry)

	// http handlers
	middlewares := handlers.NewMiddleware(tm, enforcer.Enforcer, logger)
	userHandler := handlers.NewUser(userService)
	videoHandler := handlers.NewVideoHandler(logger, config.Timeout.Duration, videoService)

	engine := gin.New()
	engine.Use(middlewares.ErrorMiddleware())
	engine.Use(middlewares.Cors())
	//register http routes
	routing.RegisterRoutes(engine, routing.Handlers{
		UserHandler:  userHandler,
		VideoHandler: videoHandler,
		Middlewares:  middlewares,
	})

	// run server
	log.Fatal(http.ListenAndServe(":8888", engine))

}
