package schema

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
