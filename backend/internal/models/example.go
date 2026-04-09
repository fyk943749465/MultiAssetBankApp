package models

import "time"

// Example is a placeholder entity; replace or extend for your domain.
type Example struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Title     string    `gorm:"size:255;not null" json:"title"`
	CreatedAt time.Time `json:"created_at"`
}
