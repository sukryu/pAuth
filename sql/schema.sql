-- Entity Schemas 테이블 정의
CREATE TABLE IF NOT EXISTS entity_schemas (
    id TEXT PRIMARY KEY,                           -- 고유 ID
    name TEXT UNIQUE NOT NULL,                    -- 스키마 이름 (Unique)
    description TEXT,                             -- 스키마 설명
    fields TEXT NOT NULL,                         -- 필드 정의 (JSON 문자열)
    indexes TEXT,                                 -- 인덱스 정의 (JSON 문자열)
    annotations TEXT,                             -- 사용자 정의 설정 (JSON 문자열)
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP                          -- Soft Delete
);

-- Schema Versions 테이블 정의
CREATE TABLE IF NOT EXISTS schema_versions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,         -- 고유 ID
    schema_name TEXT NOT NULL,                    -- 관련 스키마 이름
    version INTEGER NOT NULL,                     -- 버전 번호
    changes TEXT NOT NULL,                        -- 변경 내용 (JSON 문자열)
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (schema_name) REFERENCES entity_schemas(name)
);

-- Schema Logs 테이블 정의
CREATE TABLE IF NOT EXISTS schema_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    schema_name TEXT NOT NULL,                    -- 관련 스키마 이름
    operation TEXT NOT NULL,                      -- 작업 유형 ('CREATE', 'UPDATE', 'DELETE')
    operator TEXT,                                -- 작업자 ID
    details TEXT,                                 -- 변경 내용 (JSON 문자열)
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (schema_name) REFERENCES entity_schemas(name)
);

-- Schema Dependencies 테이블 정의
CREATE TABLE IF NOT EXISTS schema_dependencies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    parent_schema TEXT NOT NULL,                  -- 부모 스키마 이름
    child_schema TEXT NOT NULL,                   -- 자식 스키마 이름
    dependency_type TEXT NOT NULL,                -- 종속성 유형 ('foreign-key', 'reference', etc.)
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (parent_schema) REFERENCES entity_schemas(name),
    FOREIGN KEY (child_schema) REFERENCES entity_schemas(name)
);
