-- Entity Schemas 테이블 정의
CREATE TABLE IF NOT EXISTS entity_schemas (
    id TEXT PRIMARY KEY,                           -- 고유 ID
    name TEXT UNIQUE NOT NULL,                    -- 스키마 이름 (Unique)
    description TEXT,                             -- 스키마 설명
    fields TEXT NOT NULL,                         -- 필드 정의 (JSON 문자열)
    indexes TEXT,                                 -- 인덱스 정의 (JSON 문자열)
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP                          -- Soft Delete
);
