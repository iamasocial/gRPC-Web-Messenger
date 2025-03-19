-- Добавляем тип перечисления для статуса обмена ключами
CREATE TYPE key_exchange_status AS ENUM (
    'NOT_STARTED',
    'INITIATED',
    'COMPLETED',
    'FAILED'
);

-- Создаем таблицу для хранения параметров обмена ключами
CREATE TABLE dh_key_exchanges (
    id SERIAL PRIMARY KEY,
    chat_id INTEGER NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    initiator_id INTEGER NOT NULL REFERENCES users(id), -- ID пользователя, инициировавшего обмен ключами
    recipient_id INTEGER NOT NULL REFERENCES users(id), -- ID пользователя, которому предназначены ключи
    dh_g TEXT, -- Генератор
    dh_p TEXT, -- Простое число
    dh_a TEXT, -- Публичный ключ инициатора
    dh_b TEXT, -- Публичный ключ получателя
    status key_exchange_status NOT NULL DEFAULT 'NOT_STARTED',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Убедимся, что для каждого чата есть только один активный обмен ключами
    CONSTRAINT unique_active_key_exchange_per_chat UNIQUE (chat_id, status)
    DEFERRABLE INITIALLY DEFERRED
);

-- Создаем индексы для быстрого поиска
CREATE INDEX idx_dh_key_exchanges_chat_id ON dh_key_exchanges(chat_id);
CREATE INDEX idx_dh_key_exchanges_status ON dh_key_exchanges(status);
CREATE INDEX idx_dh_key_exchanges_initiator_id ON dh_key_exchanges(initiator_id);
CREATE INDEX idx_dh_key_exchanges_recipient_id ON dh_key_exchanges(recipient_id);

-- Создаем триггерную функцию для автоматического обновления поля updated_at
CREATE OR REPLACE FUNCTION update_dh_key_exchanges_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE 'plpgsql';

-- Создаем триггер для автоматического обновления поля updated_at
CREATE TRIGGER update_dh_key_exchanges_updated_at
BEFORE UPDATE ON dh_key_exchanges
FOR EACH ROW
EXECUTE PROCEDURE update_dh_key_exchanges_updated_at(); 