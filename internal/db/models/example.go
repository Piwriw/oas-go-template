// Package models defines persistent Gorm models shared by application packages.
package models

import "time"

const exampleTableName = "examples"

// Example demonstrates an explicit one-to-one mapping between a Gorm model and a database table.
type Example struct {
	// ID uniquely identifies the example row.
	ID uint64 `gorm:"column:id;primaryKey;autoIncrement"`
	// Name stores the example's display name.
	Name string `gorm:"column:name;size:255;not null"`
	// CreatedAt records when the example row was created.
	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	// UpdatedAt records when the example row was last changed.
	UpdatedAt time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

// TableName binds Example to the examples table.
func (Example) TableName() string {
	return exampleTableName
}
