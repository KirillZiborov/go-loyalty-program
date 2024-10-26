package config

import (
	"flag"
	"os"
)

type Config struct {
	Address   string
	DBPath    string
	SysAdress string
}

func NewConfig() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.Address, "a", "localhost:8080", "Address of the HTTP server")
	flag.StringVar(&cfg.SysAdress, "r", "http://localhost:8080", "Address of the acrual system")
	flag.StringVar(&cfg.DBPath, "d", "", "Database address")

	flag.Parse()

	envAddress := os.Getenv("RUN_ADDRESS")
	sysAddress := os.Getenv("ACCRUAL_SYSTEM_ADDRESS")

	if envAddress != "" {
		cfg.Address = envAddress
	} else if cfg.Address == "" {
		cfg.Address = "localhost:8080"
	}

	if sysAddress != "" {
		cfg.SysAdress = sysAddress
	} else if cfg.SysAdress == "" {
		cfg.SysAdress = "http://" + cfg.Address
	}

	if dbPath := os.Getenv("DATABASE_URI"); dbPath != "" {
		cfg.DBPath = dbPath
	}

	return cfg
}
