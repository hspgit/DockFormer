package models

import (
	"fmt"
	"time"
)

// ContainerStatus defines the possible states of a container
type ContainerStatus string

// Container statuses as enum values
const (
	StatusRunning    ContainerStatus = "running"
	StatusStopped    ContainerStatus = "stopped"
	StatusCreated    ContainerStatus = "created"
	StatusRestarting ContainerStatus = "restarting"
	StatusPaused     ContainerStatus = "paused"
	StatusExited     ContainerStatus = "exited"
)

// Container represents a container in the database.
type Container struct {
	ID          uint            `gorm:"primaryKey;autoIncrement"`
	Name        string          `gorm:"column:name;not null"`
	Image       string          `gorm:"column:image;not null"`
	ContainerID string          `gorm:"column:container_id;not null"`
	Ports       string          `gorm:"column:ports;not null"`
	Status      ContainerStatus `gorm:"column:status;type:varchar(20);not null"`
	CreatedAt   time.Time       `gorm:"column:created_at;not null"`
	UpdatedAt   time.Time       `gorm:"column:updated_at;not null"`
}

// TableName specifies the table name for the Container model
func (Container) TableName() string {
	return "containers"
}

// String returns a string representation of the Container
func (c Container) String() string {
	return fmt.Sprintf("Container{ID: %d, Name: %s, Status: %s}", c.ID, c.Name, c.Status)
}
