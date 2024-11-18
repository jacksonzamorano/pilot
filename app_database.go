package pilot

import (
	"os"
	"strconv"
)

type DatabaseConfiguration struct {
	Host     string
	Port     string
	Username string
	Password string
	Database string
}

func DatabaseFromEnvironmentWithFallback(host string, port int, username string, password string, database string) DatabaseConfiguration {
	cfg := DatabaseConfiguration{
		Host:     os.Getenv("DATABASE_HOST"),
		Port:     os.Getenv("DATABASE_PORT"),
		Username: os.Getenv("DATABASE_USERNAME"),
		Password: os.Getenv("DATABASE_PASSWORD"),
		Database: os.Getenv("DATABASE_DATABASE"),
	}
	if cfg.Host == "" {
		cfg.Host = host
	}
	if cfg.Port == "" {
		cfg.Port = strconv.Itoa(port)
	}
	if cfg.Username == "" {
		cfg.Username = username
	}
	if cfg.Password == "" {
		cfg.Password = password
	}
	if cfg.Database == "" {
		cfg.Database = database
	}
	return cfg
}

func (self *DatabaseConfiguration) GetConnectionString() string {
	return "postgres://" + self.Username + ":" + self.Password + "@" + self.Host + ":" + self.Port + "/" + self.Database
}
