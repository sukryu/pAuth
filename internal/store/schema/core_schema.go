package schema

import "time"

// CoreEntity는 모든 엔티티의 기본이 되는 구조체.
type CoreEntity struct {
	ID        string     `gorm:"primaryKey;type:uuid" json:"id"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `gorm:"index" json:"-"`
}

// CoreSchema는 시스템의 기본 스키마를 정의하는 구조체.
var CoreSchemas = []EntitySchema{
	{
		Name:        "users",
		Description: "User management table",
		Fields: []FieldDef{
			{Name: "username", Type: FieldTypeString, Required: true, Unique: true},
			{Name: "email", Type: FieldTypeString, Required: true, Unique: true},
			{Name: "password_hash", Type: FieldTypeString, Required: true},
			{Name: "is_active", Type: FieldTypeBoolean, Required: true, DefaultValue: true},
			{Name: "last_login", Type: FieldTypeTimestamp},
			{Name: "annotations", Type: FieldTypeJSON}, // JSON으로 처리되는 사용자 정의 필드
		},
		Indexes: []IndexDef{
			{Name: "idx_users_username", Columns: []string{"username"}, Unique: true},
			{Name: "idx_users_email", Columns: []string{"email"}, Unique: true},
		},
	},
	{
		Name:        "roles",
		Description: "Role definition table",
		Fields: []FieldDef{
			{Name: "name", Type: FieldTypeString, Required: true, Unique: true},
			{Name: "description", Type: FieldTypeString},
			{Name: "rules", Type: FieldTypeJSON}, // PolicyRules를 JSON으로 저장
		},
		Indexes: []IndexDef{
			{Name: "idx_roles_name", Columns: []string{"name"}, Unique: true},
		},
	},
	{
		Name:        "role_bindings",
		Description: "Role assignment table",
		Fields: []FieldDef{
			{Name: "name", Type: FieldTypeString, Required: true, Unique: true},
			{Name: "role_ref", Type: FieldTypeString, Required: true},
			{Name: "subjects", Type: FieldTypeJSON}, // Subject 목록을 JSON으로 저장
		},
		Indexes: []IndexDef{
			{Name: "idx_role_bindings_name", Columns: []string{"name"}, Unique: true},
			{Name: "idx_role_bindings_role_ref", Columns: []string{"role_ref"}},
		},
	},
}
