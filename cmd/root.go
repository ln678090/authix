package cmd

import (
	"os"
	"strings"

	"git.blinkpay.vn/software/authix/config"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	home, err := os.Getwd()
	cobra.CheckErr(err)

	_ = godotenv.Load(home + "/.env")

	viper.AddConfigPath(home)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Fatal().Msg("Config file not found")
		} else {
			log.Fatal().Err(err).Msg("Error on read config")
		}
	}

	if cfg, err := config.New(); err == nil {
		log.Debug().Any("config", cfg.LogValue()).Msg("read config")
	}
}

func Execute() {
	rootCmd.Execute()
}
