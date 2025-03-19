-- Удаляем триггер
DROP TRIGGER IF EXISTS update_dh_key_exchanges_updated_at ON dh_key_exchanges;

-- Удаляем триггерную функцию
DROP FUNCTION IF EXISTS update_dh_key_exchanges_updated_at();

-- Удаляем таблицу dh_key_exchanges
DROP TABLE IF EXISTS dh_key_exchanges;

-- Удаляем тип перечисления
DROP TYPE IF EXISTS key_exchange_status; 