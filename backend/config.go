package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Cfg contiene la configuración de la aplicación, cargada de .env o valores por defecto.
var Cfg *Config

type Config struct {
	Port                 string
	SendBufferSize       int
	NumPlayers           int
	NumDice              int
	GameCodeLength       int
	GameCodeChars        string
	DefaultVictoryScore  int
	MinVictoryScore      int
	MaxVictoryScore      int
	FinishedGameRetention time.Duration
	CleanupInterval       time.Duration
}

func init() {
	_ = godotenv.Load()

	Cfg = &Config{
		Port:                 getEnv("FARKLE_PORT", "8080"),
		SendBufferSize:       getEnvInt("FARKLE_SEND_BUFFER_SIZE", 256),
		NumPlayers:           getEnvInt("FARKLE_NUM_PLAYERS", 2),
		NumDice:              getEnvInt("FARKLE_NUM_DICE", 6),
		GameCodeLength:       getEnvInt("FARKLE_GAME_CODE_LENGTH", 5),
		GameCodeChars:        getEnv("FARKLE_GAME_CODE_CHARS", "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"),
		DefaultVictoryScore:  getEnvInt("FARKLE_DEFAULT_VICTORY_SCORE", 2000),
		MinVictoryScore:      getEnvInt("FARKLE_MIN_VICTORY_SCORE", 100),
		MaxVictoryScore:      getEnvInt("FARKLE_MAX_VICTORY_SCORE", 100000),
		FinishedGameRetention: getEnvDuration("FARKLE_FINISHED_GAME_RETENTION", 5*time.Minute),
		CleanupInterval:       getEnvDuration("FARKLE_CLEANUP_INTERVAL", 1*time.Minute),
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			log.Printf("config: %s inválido (%q), usando %d", key, v, defaultVal)
			return defaultVal
		}
		return n
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			log.Printf("config: %s inválido (%q), usando %v", key, v, defaultVal)
			return defaultVal
		}
		return d
	}
	return defaultVal
}
