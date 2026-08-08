package models_test

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"

	"github.com/piwriw/oas-go-template/internal/db/models"
)

// TestExampleSchemaMapping verifies the example model's table, columns, and managed fields.
func TestExampleSchemaMapping(t *testing.T) {
	parsed, err := schema.Parse(&models.Example{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("parse Example schema: %v", err)
	}
	if parsed.Table != "examples" {
		t.Fatalf("table = %q, want %q", parsed.Table, "examples")
	}

	wantColumns := map[string]string{
		"ID":        "id",
		"Name":      "name",
		"CreatedAt": "created_at",
		"UpdatedAt": "updated_at",
	}
	for fieldName, columnName := range wantColumns {
		field := parsed.LookUpField(fieldName)
		if field == nil {
			t.Fatalf("field %q is missing from parsed schema", fieldName)
		}
		if field.DBName != columnName {
			t.Errorf("field %s column = %q, want %q", fieldName, field.DBName, columnName)
		}
	}

	id := parsed.LookUpField("ID")
	if !id.PrimaryKey || !id.AutoIncrement {
		t.Errorf("ID primary key settings = primaryKey:%t autoIncrement:%t, want both true", id.PrimaryKey, id.AutoIncrement)
	}
	if parsed.LookUpField("CreatedAt").AutoCreateTime == 0 {
		t.Error("CreatedAt is not configured as an auto-create timestamp")
	}
	if parsed.LookUpField("UpdatedAt").AutoUpdateTime == 0 {
		t.Error("UpdatedAt is not configured as an auto-update timestamp")
	}
}
