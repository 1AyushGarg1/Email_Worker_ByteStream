package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds application configuration
type Config struct {
	ENV          string
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	SMTPSender   string
	RabbitMQURL  string
	RabbitMQQueueName string
}

// Config is the global configuration instance, accessible from other packages.
var Cfg *Config

func init() {
	// Load configuration once and store it in the global variable.
	Cfg = newConfig()
}

// newConfig loads configuration from environment
func newConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	return &Config{
		ENV:          getEnv("ENV", "dev"),
		SMTPHost:     getEnv("SMTP_HOST", ""),
		SMTPPort:     getEnv("SMTP_PORT", "587"),
		SMTPUser:     getEnv("SMTP_USER", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		SMTPSender:   getEnv("SMTP_SENDER", ""),
		RabbitMQURL:  getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		RabbitMQQueueName: getEnv("RABBITMQ_QUEUE_NAME", "email_queue"),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
