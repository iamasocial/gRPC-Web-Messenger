CREATE TABLE IF NOT EXISTS files (
    id SERIAL PRIMARY KEY,
    file_id VARCHAR(255) NOT NULL UNIQUE,
    file_name VARCHAR(255) NOT NULL,
    mime_type VARCHAR(255) NOT NULL,
    size BIGINT NOT NULL,
    path VARCHAR(1024) NOT NULL,
    uploaded_by BIGINT NOT NULL REFERENCES users(id),
    chat_id BIGINT NOT NULL REFERENCES chats(id),
    checksum VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS file_uploads (
    id SERIAL PRIMARY KEY,
    upload_id VARCHAR(255) NOT NULL UNIQUE,
    file_name VARCHAR(255) NOT NULL,
    mime_type VARCHAR(255) NOT NULL,
    total_size BIGINT NOT NULL,
    received_chunks INT NOT NULL DEFAULT 0,
    chunk_size INT NOT NULL,
    temp_path VARCHAR(1024) NOT NULL,
    user_id BIGINT NOT NULL REFERENCES users(id),
    chat_id BIGINT NOT NULL REFERENCES chats(id),
    status VARCHAR(50) NOT NULL DEFAULT 'in_progress',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
); 