package utils

import (
	"os"

	"gorm.io/gorm"
)

type EmployeeConfig struct {
	DbUser string
	DbPass string
	DbHost string
	DbPort string
	DbName string

	db *gorm.DB
}

func IntiConfig() *EmployeeConfig {
	var empConfig EmployeeConfig
	empConfig.DbUser = os.Getenv("DB_USER")
	empConfig.DbPass = os.Getenv("DB_PASS")
	empConfig.DbHost = os.Getenv("DB_HOST")
	empConfig.DbPort = os.Getenv("DB_PORT")
	empConfig.DbName = os.Getenv("DB_NAME")

	return &empConfig
}
