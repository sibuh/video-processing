package initiator

import (
	"backend/models"
	"fmt"

	"github.com/spf13/viper"
)

func LoadConfig(path string) (models.Config, error) {
	var config models.Config

	viper.AddConfigPath(path)     // folder where config.yaml is located
	viper.SetConfigName("config") // name of file (without extension)
	viper.SetConfigType("yaml")   // type of file
	viper.AutomaticEnv()          // read from environment variables too

	if err := viper.ReadInConfig(); err != nil {
		return config, fmt.Errorf("error reading config file: %w", err)
	}

	if err := viper.Unmarshal(&config); err != nil {
		return config, fmt.Errorf("unable to decode config into struct: %w", err)
	}

	return config, nil
}
