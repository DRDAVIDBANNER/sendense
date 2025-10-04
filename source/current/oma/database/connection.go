// Package database provides database connection and management for OMA
// Following project rules: modular design, clean interfaces
package database

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// MariaDBConfig holds MariaDB connection configuration
type MariaDBConfig struct {
	Host     string `json:"host" yaml:"host"`
	Port     int    `json:"port" yaml:"port"`
	Database string `json:"database" yaml:"database"`
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
	Charset  string `json:"charset" yaml:"charset"`
}

// Connection represents a database connection interface
// Follows project rules: clean interfaces, modular design
type Connection interface {
	Close() error
	Ping() error
	GetStatus() string
	GetGormDB() *gorm.DB // Access to underlying GORM instance
}

// MariaDBConnection implements Connection for MariaDB
type MariaDBConnection struct {
	config    *MariaDBConfig
	db        *gorm.DB
	connected bool
}

// NewMariaDBConnection creates a new MariaDB connection
func NewMariaDBConnection(config *MariaDBConfig) (*MariaDBConnection, error) {
	if config == nil {
		return nil, fmt.Errorf("MariaDB config is required")
	}

	conn := &MariaDBConnection{
		config:    config,
		connected: false,
	}

	// Validate config
	if err := conn.validateConfig(); err != nil {
		return nil, fmt.Errorf("invalid MariaDB config: %w", err)
	}

	// Create DSN for MariaDB connection
	if config.Charset == "" {
		config.Charset = "utf8mb4"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
		config.Charset,
	)

	// Connect to database
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MariaDB: %w", err)
	}

	conn.db = db
	conn.connected = true

	log.WithFields(log.Fields{
		"host":     config.Host,
		"port":     config.Port,
		"database": config.Database,
		"username": config.Username,
	}).Info("MariaDB connection established successfully")

	return conn, nil
}

// Close closes the database connection
func (c *MariaDBConnection) Close() error {
	if c.connected && c.db != nil {
		sqlDB, err := c.db.DB()
		if err != nil {
			log.WithError(err).Error("Failed to get SQL DB for closing")
			return err
		}
		if err := sqlDB.Close(); err != nil {
			log.WithError(err).Error("Failed to close SQL DB")
			return err
		}
		c.connected = false
		log.Info("MariaDB connection closed")
	}
	return nil
}

// Ping tests the database connection
func (c *MariaDBConnection) Ping() error {
	if !c.connected || c.db == nil {
		return fmt.Errorf("not connected to database")
	}
	sqlDB, err := c.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get SQL DB: %w", err)
	}
	return sqlDB.Ping()
}

// GetStatus returns the connection status
func (c *MariaDBConnection) GetStatus() string {
	if c.connected && c.db != nil {
		if err := c.Ping(); err == nil {
			return "connected"
		}
		return "error"
	}
	return "disconnected"
}

// GetGormDB returns the underlying GORM database instance
func (c *MariaDBConnection) GetGormDB() *gorm.DB {
	return c.db
}

// validateConfig validates the MariaDB configuration
func (c *MariaDBConnection) validateConfig() error {
	if c.config.Host == "" {
		return fmt.Errorf("host is required")
	}
	if c.config.Port <= 0 || c.config.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if c.config.Database == "" {
		return fmt.Errorf("database name is required")
	}
	if c.config.Username == "" {
		return fmt.Errorf("username is required")
	}
	return nil
}

// MemoryConnection implements Connection for in-memory storage
type MemoryConnection struct{}

// NewMemoryConnection creates a new in-memory connection
func NewMemoryConnection() *MemoryConnection {
	log.Info("Using in-memory storage (no persistence)")
	return &MemoryConnection{}
}

// Close closes the memory connection (no-op)
func (c *MemoryConnection) Close() error {
	return nil
}

// Ping tests the memory connection (always succeeds)
func (c *MemoryConnection) Ping() error {
	return nil
}

// GetStatus returns the connection status
func (c *MemoryConnection) GetStatus() string {
	return "memory"
}

// GetGormDB returns nil for memory connection (no database)
func (c *MemoryConnection) GetGormDB() *gorm.DB {
	return nil
}
