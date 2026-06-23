package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppPort       string
	PublicBaseURL string
	CORSOrigins   string
	CompanyID     int32 // 0 = monitor all companies
	SQLitePath    string
	WorkerTimeout time.Duration
	ClickHouse    ClickHouseConfig
	Redis         RedisConfig
	Mailtarget    MailtargetConfig
	Alert         AlertConfig
	KillSwitch    KillSwitchConfig
	AdminToken        string
	DashboardUsername string
	DashboardPassword string
	PostgresDSN       string
}

type ClickHouseConfig struct {
	Host     string
	Database string
	Username string
	Password string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type MailtargetConfig struct {
	APIKey          string
	TransmissionURL string
	APIConfigURL    string
}

type AlertConfig struct {
	FromEmail string
	FromName  string
	ToEmail   string
	ToName    string
}

type KillSwitchConfig struct {
	HMACSecret string
	TokenTTL   time.Duration
}

func Load() (*Config, error) {
	companyID, _ := strconv.ParseInt(getEnv("COMPANY_ID", "0"), 10, 32)

	redisDB, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		return nil, fmt.Errorf("invalid REDIS_DB: %w", err)
	}

	return &Config{
		AppPort:       getEnv("APP_PORT", "8080"),
		PublicBaseURL: getEnv("PUBLIC_BASE_URL", "http://localhost:8080"),
		CORSOrigins:   getEnv("CORS_ORIGINS", "http://localhost:5173,http://localhost:3000"),
		CompanyID:     int32(companyID),
		SQLitePath:    getEnv("SQLITE_PATH", "./data/sentinel.db"),
		WorkerTimeout: 60 * time.Second,
		ClickHouse: ClickHouseConfig{
			Host:     getEnv("CLICKHOUSE_HOST", "localhost:9000"),
			Database: getEnv("CLICKHOUSE_DATABASE", "default"),
			Username: getEnv("CLICKHOUSE_USERNAME", "default"),
			Password: getEnv("CLICKHOUSE_PASSWORD", ""),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       redisDB,
		},
		Mailtarget: MailtargetConfig{
			APIKey:          getEnv("MAILTARGET_API_KEY", ""),
			TransmissionURL: getEnv("MAILTARGET_TRANSMISSION_URL", "https://transmission.mailtarget.co/v1"),
			APIConfigURL:    getEnv("MAILTARGET_APICONFIG_URL", "https://apiconfig.mailtarget.co/v1"),
		},
		Alert: AlertConfig{
			FromEmail: getEnv("ALERT_FROM_EMAIL", "alerts@example.com"),
			FromName:  getEnv("ALERT_FROM_NAME", "Mailtarget Sentinel"),
			ToEmail:   getEnv("ALERT_TO_EMAIL", "ops@example.com"),
			ToName:    getEnv("ALERT_TO_NAME", "Ops Team"),
		},
		KillSwitch: KillSwitchConfig{
			HMACSecret: getEnv("KILL_SWITCH_HMAC_SECRET", "change-me-in-production"),
			TokenTTL:   24 * time.Hour,
		},
		AdminToken:        getEnv("SENTINEL_ADMIN_TOKEN", ""),
		DashboardUsername: getEnv("DASHBOARD_USERNAME", "admin"),
		DashboardPassword: getEnv("DASHBOARD_PASSWORD", "s3ntinelg0d"),
		PostgresDSN:       getEnv("POSTGRES_DSN", ""),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
