package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/sukryu/pAuth/pkg/errors"
	"github.com/sukryu/pAuth/pkg/store/schema"
)

// TableManager handles the creation and management of database tables
type TableManager struct {
	db *gorm.DB
}

// EntitySchemaModel은 데이터베이스에 저장될 모델입니다.
type EntitySchemaModel struct {
	ID          string `gorm:"primarykey;type:uuid"`
	CreatedAt   string
	UpdatedAt   string
	DeletedAt   *string
	Name        string `gorm:"uniqueIndex"`
	Description string
	Fields      string // JSON string
	Indexes     string // JSON string
}

func (EntitySchemaModel) TableName() string {
	return "entity_schemas"
}

// Schema를 Model로 변환
func toModel(schema *schema.EntitySchema) (*EntitySchemaModel, error) {
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
		CreatedAt:   schema.CreatedAt.String(),
		UpdatedAt:   schema.UpdatedAt.String(),
		Name:        schema.Name,
		Description: schema.Description,
		Fields:      string(fieldsBytes),
		Indexes:     string(indexesBytes),
	}, nil
}

// NewTableManager creates a new TableManager instance
func NewTableManager(db *gorm.DB) *TableManager {
	return &TableManager{
		db: db,
	}
}

// Initialize creates the necessary tables and indexes for core schemas
func (tm *TableManager) Initialize(ctx context.Context) error {
	// EntitySchemaModel 테이블 생성
	if err := tm.db.AutoMigrate(&EntitySchemaModel{}); err != nil {
		return errors.ErrStorageOperation.WithReason(fmt.Sprintf("failed to create entity schema table: %v", err))
	}

	// CoreSchemas 생성
	for _, schemaObj := range schema.CoreSchemas {
		if err := tm.CreateTable(ctx, &schemaObj); err != nil {
			return errors.ErrStorageOperation.WithReason(fmt.Sprintf("failed to create core schema %s: %v", schemaObj.Name, err))
		}
	}

	return nil
}

func (tm *TableManager) CreateTable(ctx context.Context, schema *schema.EntitySchema) error {
	exists, err := tm.TableExists(ctx, schema.Name)
	if err != nil {
		return err
	}
	if exists {
		return errors.ErrAlreadyExists.WithReason(fmt.Sprintf("table %s already exists", schema.Name))
	}

	sql, err := tm.generateCreateTableSQL(schema)
	if err != nil {
		return err
	}

	return tm.db.Transaction(func(tx *gorm.DB) error {
		// Create table
		if err := tx.Exec(sql).Error; err != nil {
			return errors.ErrStorageOperation.WithReason(fmt.Sprintf("failed to create table: %v", err))
		}

		// Create indexes
		for _, idx := range schema.Indexes {
			idxSQL := tm.generateCreateIndexSQL(schema.Name, idx)
			if err := tx.Exec(idxSQL).Error; err != nil {
				return errors.ErrStorageOperation.WithReason(fmt.Sprintf("failed to create index: %v", err))
			}
		}

		// Store schema definition
		model, err := toModel(schema)
		if err != nil {
			return errors.ErrStorageOperation.WithReason(fmt.Sprintf("failed to convert schema: %v", err))
		}

		if err := tx.Create(model).Error; err != nil {
			return errors.ErrStorageOperation.WithReason(fmt.Sprintf("failed to store schema definition: %v", err))
		}

		return nil
	})
}

// DropTable drops a table and its schema definition
func (tm *TableManager) DropTable(ctx context.Context, tableName string) error {
	exists, err := tm.TableExists(ctx, tableName)
	if err != nil {
		return err
	}
	if !exists {
		return errors.ErrNotFound.WithReason(fmt.Sprintf("table %s does not exist", tableName))
	}

	return tm.db.Transaction(func(tx *gorm.DB) error {
		// Drop the actual table
		if err := tx.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)).Error; err != nil {
			return errors.ErrStorageOperation.WithReason(fmt.Sprintf("failed to drop table: %v", err))
		}

		// Remove schema definition
		if err := tx.Where("name = ?", tableName).Delete(&schema.EntitySchema{}).Error; err != nil {
			return errors.ErrStorageOperation.WithReason(fmt.Sprintf("failed to remove schema definition: %v", err))
		}

		return nil
	})
}

// UpdateTable updates a table's schema
func (tm *TableManager) UpdateTable(ctx context.Context, schema *schema.EntitySchema) error {
	exists, err := tm.TableExists(ctx, schema.Name)
	if err != nil {
		return err
	}
	if !exists {
		return errors.ErrNotFound.WithReason(fmt.Sprintf("table %s does not exist", schema.Name))
	}

	currentSchema, err := tm.GetTableSchema(ctx, schema.Name)
	if err != nil {
		return err
	}

	return tm.db.Transaction(func(tx *gorm.DB) error {
		// Add new columns
		for _, newField := range schema.Fields {
			if !tm.fieldExists(currentSchema.Fields, newField.Name) {
				sql, err := tm.generateAddColumnSQL(schema.Name, newField)
				if err != nil {
					return err
				}
				if err := tx.Exec(sql).Error; err != nil {
					return errors.ErrStorageOperation.WithReason(fmt.Sprintf("failed to add column: %v", err))
				}
			}
		}

		// Add new indexes
		for _, newIndex := range schema.Indexes {
			if !tm.indexExists(currentSchema.Indexes, newIndex.Name) {
				sql := tm.generateCreateIndexSQL(schema.Name, newIndex)
				if err := tx.Exec(sql).Error; err != nil {
					return errors.ErrStorageOperation.WithReason(fmt.Sprintf("failed to create index: %v", err))
				}
			}
		}

		// Update schema definition in the entity_schemas table
		updateData := map[string]interface{}{
			"description": schema.Description,
			"fields":      schema.Fields,
			"indexes":     schema.Indexes,
		}

		if err := tx.Table("entity_schemas").Where("name = ?", schema.Name).Updates(updateData).Error; err != nil {
			return errors.ErrStorageOperation.WithReason(fmt.Sprintf("failed to update schema definition: %v", err))
		}

		return nil
	})
}

// ListTables returns all table schemas
func (tm *TableManager) ListTables(ctx context.Context) ([]*schema.EntitySchema, error) {
	var schemas []*schema.EntitySchema
	if err := tm.db.Find(&schemas).Error; err != nil {
		return nil, errors.ErrStorageOperation.WithReason(fmt.Sprintf("failed to list tables: %v", err))
	}
	return schemas, nil
}

// TableExists checks if a table exists
func (tm *TableManager) TableExists(ctx context.Context, tableName string) (bool, error) {
	var count int64
	err := tm.db.Model(&schema.EntitySchema{}).Where("name = ?", tableName).Count(&count).Error
	if err != nil {
		return false, errors.ErrStorageOperation.WithReason(fmt.Sprintf("failed to check table existence: %v", err))
	}
	return count > 0, nil
}

// GetTableSchema retrieves the schema for a specific table
func (tm *TableManager) GetTableSchema(ctx context.Context, tableName string) (*schema.EntitySchema, error) {
	var tableSchema schema.EntitySchema
	err := tm.db.Where("name = ?", tableName).First(&tableSchema).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrNotFound.WithReason(fmt.Sprintf("table %s not found", tableName))
		}
		return nil, errors.ErrStorageOperation.WithReason(fmt.Sprintf("failed to get table schema: %v", err))
	}
	return &tableSchema, nil
}

// Helper methods

func (tm *TableManager) generateCreateTableSQL(schema *schema.EntitySchema) (string, error) {
	var columns []string

	// Add core entity fields
	columns = append(columns, []string{
		"id TEXT PRIMARY KEY",
		"created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP",
		"updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP",
		"deleted_at TIMESTAMP",
		"name TEXT UNIQUE NOT NULL",
	}...)

	// Add schema-specific fields
	for _, field := range schema.Fields {
		column, err := tm.generateColumnDef(field)
		if err != nil {
			return "", err
		}
		columns = append(columns, column)
	}

	sql := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		%s
	)`, schema.Name, strings.Join(columns, ",\n\t"))

	return sql, nil
}

func (tm *TableManager) generateColumnDef(field schema.FieldDef) (string, error) {
	var def strings.Builder
	def.WriteString(field.Name)
	def.WriteString(" ")

	switch field.Type {
	case schema.FieldTypeString:
		def.WriteString("TEXT")
	case schema.FieldTypeNumber:
		def.WriteString("NUMERIC")
	case schema.FieldTypeBoolean:
		def.WriteString("BOOLEAN")
	case schema.FieldTypeTimestamp:
		def.WriteString("TIMESTAMP")
	case schema.FieldTypeJSON:
		def.WriteString("TEXT")
	case schema.FieldTypeArray:
		def.WriteString("TEXT")
	default:
		return "", errors.ErrInvalidInput.WithReason(fmt.Sprintf("unsupported field type: %s", field.Type))
	}

	if field.Required {
		def.WriteString(" NOT NULL")
	}

	if field.Unique {
		def.WriteString(" UNIQUE")
	}

	if field.DefaultValue != nil {
		def.WriteString(fmt.Sprintf(" DEFAULT %v", field.DefaultValue))
	}

	return def.String(), nil
}

func (tm *TableManager) generateCreateIndexSQL(tableName string, idx schema.IndexDef) string {
	unique := ""
	if idx.Unique {
		unique = "UNIQUE"
	}
	return fmt.Sprintf("CREATE %s INDEX IF NOT EXISTS %s ON %s (%s)",
		unique,
		idx.Name,
		tableName,
		strings.Join(idx.Fields, ", "))
}

func (tm *TableManager) generateAddColumnSQL(tableName string, field schema.FieldDef) (string, error) {
	columnDef, err := tm.generateColumnDef(field)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", tableName, columnDef), nil
}

func (tm *TableManager) fieldExists(fields []schema.FieldDef, fieldName string) bool {
	for _, field := range fields {
		if field.Name == fieldName {
			return true
		}
	}
	return false
}

func (tm *TableManager) indexExists(indexes []schema.IndexDef, indexName string) bool {
	for _, index := range indexes {
		if index.Name == indexName {
			return true
		}
	}
	return false
}
