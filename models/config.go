package models

import "time"

type Config struct {
	Database struct {
		Host     string `mapstructure:"host"`
		Port     string `mapstructure:"port"`
		Name     string `mapstructure:"name"`
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
	} `mapstructure:"database"`
	TestDB struct {
		Name     string `mapstructure:"name"`
		Host     string `mapstructure:"host"`
		Port     string `mapstructure:"port"`
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
	} `mapstructure:"testdb"`
	Token struct {
		Duration time.Duration `mapstructure:"duration"`
		Key      string        `mapstructure:"key"`
	} `mapstructure:"token"`
}
