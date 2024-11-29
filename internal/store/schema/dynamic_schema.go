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
	Name          string    `json:"name"`
	Type          FieldType `json:"type"`
	Required      bool      `json:"required"`
	Unique        bool      `json:"unique"`
	Nullable      bool
	DefaultValue  interface{} `json:"defaultValue,omitempty"`
	PrimaryKey    bool        `json:"primaryKey"`
	NotNull       bool        `json:"notNull"`
	AutoIncrement bool        `json:"autoIncrement"`
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

func ValidateFieldType(value interface{}, fieldType FieldType) error {
	switch fieldType {
	case FieldTypeString:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("value must be of type string")
		}
	case FieldTypeNumber, FieldTypeInteger:
		switch value.(type) {
		case int, int32, int64, float32, float64:
			return nil
		default:
			return fmt.Errorf("value must be of type number or integer")
		}
	case FieldTypeBoolean:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("value must be of type boolean")
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
			return fmt.Errorf("value must be of type timestamp")
		}
	case FieldTypeJSON:
		// JSON 타입은 직렬화 가능 여부로 검증
		if _, ok := value.(string); !ok {
			return fmt.Errorf("value must be JSON serializable")
		}
	default:
		return fmt.Errorf("unsupported field type: %s", fieldType)
	}
	return nil
}
