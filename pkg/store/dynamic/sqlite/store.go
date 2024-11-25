package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sukryu/pAuth/pkg/errors"
	"github.com/sukryu/pAuth/pkg/store/dynamic"
	"github.com/sukryu/pAuth/pkg/store/schema"
	"gorm.io/gorm"
)

const SQLiteDB dynamic.DatabaseType = "sqlite"

type SQLiteDynamicStore struct {
	db *gorm.DB
}

func NewSQLiteDynamicStore(db *gorm.DB) (dynamic.DynamicStore, error) {
	if db == nil {
		return nil, fmt.Errorf("db cannot be nil")
	}
	return &SQLiteDynamicStore{db: db}, nil
}

// getCoreEntityFields는 CoreEntity의 필드들을 FieldDef 슬라이스로 반환합니다.
func getCoreEntityFields() []schema.FieldDef {
	return []schema.FieldDef{
		{Name: "id", Type: schema.FieldTypeString, Required: true},
		{Name: "created_at", Type: schema.FieldTypeTimestamp, Required: true},
		{Name: "updated_at", Type: schema.FieldTypeTimestamp, Required: true},
		{Name: "deleted_at", Type: schema.FieldTypeTimestamp},
	}
}

func (s *SQLiteDynamicStore) Create(ctx context.Context, tableName string, data map[string]interface{}) error {
	schema, err := s.GetSchema(ctx, tableName)
	if err != nil {
		return err
	}

	// id 생성
	if _, exists := data["id"]; !exists {
		data["id"] = uuid.New().String()
	}

	// created_at 및 updated_at 설정
	now := time.Now()
	if _, exists := data["created_at"]; !exists {
		data["created_at"] = now
	}
	if _, exists := data["updated_at"]; !exists {
		data["updated_at"] = now
	}

	// 데이터 검증
	if err := s.ValidateData(data, schema); err != nil {
		return err
	}

	// Raw SQL Insert 사용
	columns := make([]string, 0)
	values := make([]interface{}, 0)
	placeholders := make([]string, 0)

	for _, field := range schema.Fields {
		if value, exists := data[field.Name]; exists {
			columns = append(columns, field.Name)
			values = append(values, value)
			placeholders = append(placeholders, "?")
		}
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	return s.db.WithContext(ctx).Exec(query, values...).Error
}

func (s *SQLiteDynamicStore) Get(ctx context.Context, tableName string, id string) (map[string]interface{}, error) {
	// schema, err := s.GetSchema(ctx, tableName)
	// if err != nil {
	// 	return nil, err
	// }

	// Raw SQL 쿼리 사용
	rows, err := s.db.WithContext(ctx).Table(tableName).Where("id = ?", id).Rows()
	if err != nil {
		return nil, errors.ErrStorageOperation.WithReason(err.Error())
	}
	defer rows.Close()

	// 결과가 없는 경우
	if !rows.Next() {
		return nil, errors.ErrNotFound.WithReason(fmt.Sprintf("%s/%s", tableName, id))
	}

	// 컬럼 정보 가져오기
	columns, err := rows.Columns()
	if err != nil {
		return nil, errors.ErrStorageOperation.WithReason(err.Error())
	}

	// 값을 저장할 슬라이스 준비
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	// 로우 스캔
	if err := rows.Scan(valuePtrs...); err != nil {
		return nil, errors.ErrStorageOperation.WithReason(err.Error())
	}

	// 결과 맵 생성
	result := make(map[string]interface{})
	for i, col := range columns {
		val := values[i]
		result[col] = val
	}

	return result, nil
}

func (s *SQLiteDynamicStore) Update(ctx context.Context, tableName string, id string, data map[string]interface{}) error {
	data["updated_at"] = time.Now()
	schema, err := s.GetSchema(ctx, tableName)
	if err != nil {
		return err
	}

	// 데이터 검증 및 변환
	converted := make(map[string]interface{})
	for key, value := range data {
		for _, field := range schema.Fields {
			if field.Name == key {
				convertedValue, err := convertValueToType(value, field.Type)
				if err != nil {
					return errors.ErrInvalidInput.WithReason(fmt.Sprintf("field '%s': %v", key, err))
				}
				converted[key] = convertedValue
			}
		}
	}

	result := s.db.WithContext(ctx).Table(tableName).Where("id = ?", id).Updates(converted)
	if result.Error != nil {
		return errors.ErrStorageOperation.WithReason(result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return errors.ErrNotFound.WithReason(fmt.Sprintf("%s/%s", tableName, id))
	}
	return nil
}

func (s *SQLiteDynamicStore) Delete(ctx context.Context, tableName string, id string) error {
	result := s.db.WithContext(ctx).Table(tableName).Where("id = ?", id).Delete(map[string]interface{}{})
	if result.Error != nil {
		return errors.ErrStorageOperation.WithReason(result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return errors.ErrNotFound.WithReason(fmt.Sprintf("%s/%s", tableName, id))
	}
	return nil
}

func (s *SQLiteDynamicStore) List(ctx context.Context, tableName string, filter map[string]interface{}, limit, offset int) ([]map[string]interface{}, error) {
	schema, err := s.GetSchema(ctx, tableName)
	if err != nil {
		return nil, err
	}

	query := s.db.WithContext(ctx).Table(tableName)

	// 필터 적용
	for key, value := range filter {
		query = query.Where(fmt.Sprintf("%s = ?", key), value)
	}

	// 페이징
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	var results []map[string]interface{}
	if err := query.Find(&results).Error; err != nil {
		return nil, errors.ErrStorageOperation.WithReason(err.Error())
	}

	// 결과 데이터 타입 변환
	for i, result := range results {
		for _, field := range schema.Fields {
			if value, exists := result[field.Name]; exists {
				converted, err := convertValueToType(value, field.Type)
				if err != nil {
					return nil, errors.ErrStorageOperation.WithReason(err.Error())
				}
				results[i][field.Name] = converted
			}
		}
	}

	return results, nil
}

func (s *SQLiteDynamicStore) Count(ctx context.Context, tableName string, filter map[string]interface{}) (int64, error) {
	var count int64
	query := s.db.WithContext(ctx).Table(tableName)

	for key, value := range filter {
		query = query.Where(fmt.Sprintf("%s = ?", key), value)
	}

	if err := query.Count(&count).Error; err != nil {
		return 0, errors.ErrStorageOperation.WithReason(err.Error())
	}
	return count, nil
}

func (s *SQLiteDynamicStore) Query(ctx context.Context, tableName string, query string, args ...interface{}) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	if err := s.db.WithContext(ctx).Table(tableName).Raw(query, args...).Scan(&results).Error; err != nil {
		return nil, errors.ErrStorageOperation.WithReason(err.Error())
	}
	return results, nil
}

func (s *SQLiteDynamicStore) GetSchema(ctx context.Context, tableName string) (*schema.EntitySchema, error) {
	var model schema.EntitySchemaModel
	if err := s.db.WithContext(ctx).Where("name = ?", tableName).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrNotFound.WithReason(fmt.Sprintf("table %s not found", tableName))
		}
		return nil, errors.ErrStorageOperation.WithReason(err.Error())
	}

	var fields []schema.FieldDef
	var indexes []schema.IndexDef

	if err := json.Unmarshal([]byte(model.Fields), &fields); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(model.Indexes), &indexes); err != nil {
		return nil, err
	}

	return &schema.EntitySchema{
		Name:        model.Name,
		Description: model.Description,
		Fields:      fields,
		Indexes:     indexes,
	}, nil
}

func toModel(schemas *schema.EntitySchema) (*schema.EntitySchemaModel, error) {
	fieldsBytes, err := json.Marshal(schemas.Fields)
	if err != nil {
		return nil, err
	}

	indexesBytes, err := json.Marshal(schemas.Indexes)
	if err != nil {
		return nil, err
	}

	return &schema.EntitySchemaModel{
		ID:          schemas.ID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Name:        schemas.Name,
		Description: schemas.Description,
		Fields:      string(fieldsBytes),
		Indexes:     string(indexesBytes),
	}, nil
}

func (s *SQLiteDynamicStore) ValidateData(data map[string]interface{}, schema *schema.EntitySchema) error {
	for _, field := range schema.Fields {
		value, exists := data[field.Name]

		if field.Required && !exists {
			return errors.ErrInvalidInput.WithReason(fmt.Sprintf("field '%s' is required", field.Name))
		}

		if exists {
			if _, err := convertValueToType(value, field.Type); err != nil {
				return errors.ErrInvalidInput.WithReason(fmt.Sprintf("field '%s': %v", field.Name, err))
			}
		}
	}
	return nil
}

func validateFieldValue(value interface{}, field schema.FieldDef) error {
	switch field.Type {
	case schema.FieldTypeJSON:
		if str, ok := value.(string); ok {
			var js interface{}
			if err := json.Unmarshal([]byte(str), &js); err != nil {
				return errors.ErrInvalidJSON.WithReason(fmt.Sprintf("field '%s': %v", field.Name, err))
			}
			return nil
		}
		switch value.(type) {
		case map[string]interface{}, []interface{}:
			return nil
		}
		return errors.ErrInvalidJSON.WithReason(fmt.Sprintf("field '%s': invalid JSON type", field.Name))

	case schema.FieldTypeTimestamp:
		str, ok := value.(string)
		if !ok {
			return errors.ErrInvalidTimestamp.WithReason(fmt.Sprintf("field '%s': not a string value", field.Name))
		}
		if !validateTimestamp(str) {
			return errors.ErrInvalidTimestamp.WithReason(fmt.Sprintf("field '%s': invalid format", field.Name))
		}
		return nil

	case schema.FieldTypeString:
		if _, ok := value.(string); !ok {
			return errors.ErrInvalidFieldType.WithReason(fmt.Sprintf("field '%s': expected string", field.Name))
		}

	case schema.FieldTypeNumber:
		switch value.(type) {
		case int, int32, int64, float32, float64:
			return nil
		}
		return errors.ErrInvalidFieldType.WithReason(fmt.Sprintf("field '%s': expected number", field.Name))

	case schema.FieldTypeBoolean:
		if _, ok := value.(bool); !ok {
			return errors.ErrInvalidFieldType.WithReason(fmt.Sprintf("field '%s': expected boolean", field.Name))
		}
	}

	return nil
}

func (s *SQLiteDynamicStore) TransactionWithOptions(ctx context.Context, opts dynamic.TransactionOptions, fn func(tx dynamic.DynamicStore) error) error {
	defaultOpts := DefaultTransactionOptions(SQLiteDB)
	finalOpts := MergeTransactionOptions(defaultOpts, opts)

	if finalOpts.IsolationLevel != dynamic.Serializable {
		log.Printf("SQLite only supports SERIALIZABLE isolation level, ignoring requested level: %v", finalOpts.IsolationLevel)
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if opts.ReadOnly {
			if err := tx.Exec("PRAGMA query_only = ON").Error; err != nil {
				return err
			}
		}

		txStore := &SQLiteDynamicStore{db: tx}
		return fn(txStore)
	})
}

func DefaultTransactionOptions(dbType dynamic.DatabaseType) dynamic.TransactionOptions {
	return dynamic.TransactionOptions{
		IsolationLevel: dynamic.Serializable,
		ReadOnly:       false,
	}
}

func MergeTransactionOptions(defaultOpts, userOpts dynamic.TransactionOptions) dynamic.TransactionOptions {
	return dynamic.TransactionOptions{
		IsolationLevel: func() dynamic.IsolationLevel {
			if userOpts.IsolationLevel != 0 {
				return userOpts.IsolationLevel
			}
			return defaultOpts.IsolationLevel
		}(),
		ReadOnly: userOpts.ReadOnly || defaultOpts.ReadOnly,
	}
}

func validateTimestamp(value string) bool {
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
	}

	for _, layout := range layouts {
		if _, err := time.Parse(layout, value); err == nil {
			return true
		}
	}
	return false
}

func (s *SQLiteDynamicStore) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Utils

func convertValueToType(value interface{}, fieldType schema.FieldType) (interface{}, error) {
	switch fieldType {
	case schema.FieldTypeString:
		return toString(value)
	case schema.FieldTypeNumber:
		return toNumber(value)
	case schema.FieldTypeBoolean:
		return toBool(value)
	case schema.FieldTypeTimestamp:
		return toTimestamp(value)
	case schema.FieldTypeJSON:
		return toJSON(value)
	default:
		return nil, fmt.Errorf("unsupported field type: %s", fieldType)
	}
}

func toString(value interface{}) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

func toNumber(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case int, int32, int64, float32, float64:
		return v, nil
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f, nil
		}
		return nil, fmt.Errorf("cannot convert string '%s' to number", v)
	default:
		return nil, fmt.Errorf("cannot convert type %T to number", value)
	}
}

func toBool(value interface{}) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		return strconv.ParseBool(v)
	case int:
		return v != 0, nil
	default:
		return false, fmt.Errorf("cannot convert type %T to boolean", value)
	}
}

func toTimestamp(value interface{}) (time.Time, error) {
	switch v := value.(type) {
	case time.Time:
		return v, nil
	case string:
		for _, layout := range []string{
			time.RFC3339,
			"2006-01-02T15:04:05Z",
			"2006-01-02 15:04:05",
		} {
			if t, err := time.Parse(layout, v); err == nil {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("cannot parse timestamp: %s", v)
	default:
		return time.Time{}, fmt.Errorf("cannot convert type %T to timestamp", value)
	}
}

func toJSON(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		var result interface{}
		if err := json.Unmarshal([]byte(v), &result); err != nil {
			return nil, fmt.Errorf("invalid JSON string: %v", err)
		}
		return result, nil
	case map[string]interface{}, []interface{}:
		return v, nil
	default:
		return nil, fmt.Errorf("cannot convert type %T to JSON", value)
	}
}
