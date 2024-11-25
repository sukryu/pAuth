package schema

import (
	"encoding/json"
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
	FieldTypeString    FieldType = "string"
	FieldTypeNumber    FieldType = "number"
	FieldTypeBoolean   FieldType = "boolean"
	FieldTypeTimestamp FieldType = "timestamp"
	FieldTypeJSON      FieldType = "json"
	FieldTypeArray     FieldType = "array"
)

type FieldDef struct {
	Name         string      `json:"name"`
	Type         FieldType   `json:"type"`
	Required     bool        `json:"required"`
	Unique       bool        `json:"unique"`
	DefaultValue interface{} `json:"defaultValue,omitempty"`
}

type IndexDef struct {
	Name   string   `json:"name"`
	Fields []string `json:"fields"`
	Unique bool     `json:"unique"`
}

type EntitySchemaModel struct {
	ID          string `gorm:"primarykey;type:uuid"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time `gorm:"index"`
	Name        string     `gorm:"uniqueIndex"`
	Description string
	Fields      string
	Indexes     string
}

func (EntitySchemaModel) TableName() string {
	return "entity_schemas"
}

func toModel(schema EntitySchema) (*EntitySchemaModel, error) {
	fieldsBytes, err := json.Marshal(schema.Fields)
	if err != nil {
		return nil, err
	}

	indexesBytes, err := json.Marshal(schema.Indexes)
	if err != nil {
		return nil, err
	}

	return &EntitySchemaModel{
		ID:          schema.ID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Name:        schema.Name,
		Description: schema.Description,
		Fields:      string(fieldsBytes),
		Indexes:     string(indexesBytes),
	}, nil
}
