package user_test

import (
	"video-processing/initiator"
	"video-processing/models"
)

// loadConfigForTest tries common relative locations so `go test ./...` works from repo root or subpackages.
func loadConfig(path string) (models.Config, error) {
	return initiator.LoadConfig(path)
}

// getMigrationsURL returns a file:// URL to the migrations directory that exists.
func getMigrations(path, testDbName, dsn string) error {
	return initiator.RunMigrations(path, testDbName, dsn)
}
