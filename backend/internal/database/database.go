package database

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// Connect establishes a connection to the database using the connection string from .env
func Connect() (*gorm.DB, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using default connection string")
	}

	connectionString := os.Getenv("DATABASE_URL")
	if connectionString == "" {
		connectionString = "host=localhost user=postgres password=postgres dbname=receipts_db port=5432 sslmode=disable"
		log.Println("DATABASE_URL not found in environment, using default connection string")
	}

	db, err := gorm.Open(postgres.Open(connectionString), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	log.Println("Connected to database successfully")
	DB = db
	return db, nil
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return DB
}
