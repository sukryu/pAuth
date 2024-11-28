package schema

import (
	"fmt"
	"time"
)

type EntitySchema struct {
	CoreEntity
	Name        string     `json:"name" gorm:"unique;not null"`
	Description string     `json:"description"`
	Fields      []FieldDef `json:"fields" gorm:"type:jsonb"`
	Indexes     []IndexDef `json:"indexes" gorm:"type:jsonb"`
}

type FieldType string

const (
	FieldTypeString    FieldType = "TEXT"
	FieldTypeNumber    FieldType = "NUMERIC"
	FieldTypeInteger   FieldType = "INTEGER"
	FieldTypeBoolean   FieldType = "BOOLEAN"
	FieldTypeTimestamp FieldType = "TIMESTAMP"
	FieldTypeJSON      FieldType = "JSON"
)

type FieldDef struct {
	Name         string    `json:"name"`
	Type         FieldType `json:"type"`
	Required     bool      `json:"required"`
	Unique       bool      `json:"unique"`
	Nullable     bool
	DefaultValue interface{} `json:"defaultValue,omitempty"`
}

type IndexDef struct {
	Name    string   `json:"name"`
	Columns []string `json:"fields"`
	Unique  bool     `json:"unique"`
}

type TableOptions struct {
	Fields  []FieldDef
	Indexes []IndexDef
}

func (f FieldDef) GenerateColumnDef() string {
	columnDef := f.Name + " " + string(f.Type)
	if !f.Nullable {
		columnDef += " NOT NULL"
	}
	if f.DefaultValue != nil {
		columnDef += fmt.Sprintf(" DEFAULT %v", f.DefaultValue)
	}
	return columnDef
}

func validateFieldType(value interface{}, fieldType FieldType) error {
	switch fieldType {
	case FieldTypeString:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("value must be string type")
		}
	case FieldTypeNumber, FieldTypeInteger:
		switch value.(type) {
		case int, int32, int64, float32, float64:
			return nil
		default:
			return fmt.Errorf("value must be numeric type")
		}
	case FieldTypeBoolean:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("value must be boolean type")
		}
	case FieldTypeTimestamp:
		switch v := value.(type) {
		case string:
			if _, err := time.Parse(time.RFC3339, v); err != nil {
				return fmt.Errorf("invalid timestamp format")
			}
		case time.Time:
			return nil
		default:
			return fmt.Errorf("value must be timestamp type")
		}
	}
	return nil
}
