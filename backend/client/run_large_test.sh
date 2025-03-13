#!/bin/bash

# Функция для очистки и выхода при ошибке
cleanup_and_exit() {
    echo "Ошибка: $1"
    exit 1
}

echo "Компиляция клиента..."
# Компиляция клиента
if ! go build -o client .; then
    cleanup_and_exit "Не удалось скомпилировать клиент"
fi

echo "Проверка доступности сервера..."
# Проверка доступности сервера
if ! timeout 1 bash -c "echo > /dev/tcp/localhost/50051" 2>/dev/null; then
    cleanup_and_exit "Сервер недоступен на порту 50051"
fi

echo "Примечание: Для работы тестов пользователь 'sas' должен существовать с паролем 'qwerty'."

# Создание тестового файла, если его нет
TEST_FILE="very_large_test_file.bin"

echo
echo "===== ТЕСТИРОВАНИЕ ЗАГРУЗКИ БОЛЬШОГО ФАЙЛА (100MB) ====="

# Замеряем время загрузки файла
echo "Загрузка файла на сервер..."
UPLOAD_START_TIME=$(date +%s.%N)
./client -upload -file "$TEST_FILE" || cleanup_and_exit "Ошибка при загрузке файла"
UPLOAD_END_TIME=$(date +%s.%N)
UPLOAD_DURATION=$(echo "$UPLOAD_END_TIME - $UPLOAD_START_TIME" | bc)
UPLOAD_SPEED=$(echo "scale=2; ${TEST_FILE_SIZE:-104857600} / 1024 / 1024 / $UPLOAD_DURATION" | bc)
echo "Время загрузки: $UPLOAD_DURATION секунд"
echo "Скорость загрузки: $UPLOAD_SPEED МБ/с"

# Получение списка файлов
echo "Получение списка файлов..."
TEMP_LIST_FILE=$(mktemp)
./client -list > "$TEMP_LIST_FILE" 2>&1 || cleanup_and_exit "Ошибка при получении списка файлов"
cat "$TEMP_LIST_FILE"

# Извлечение ID файла из списка
FIRST_FILE_LINE=$(grep "$TEST_FILE" "$TEMP_LIST_FILE" | head -1)
echo "Строка с первым файлом: $FIRST_FILE_LINE"

# Извлечение ID файла с помощью grep и sed
FILE_ID=$(echo "$FIRST_FILE_LINE" | grep -o "ID: [0-9a-zA-Z\-]*" | sed 's/ID: //')
echo "Извлеченный ID файла: $FILE_ID"

if [ -z "$FILE_ID" ]; then
    cleanup_and_exit "Не удалось извлечь ID файла из списка"
fi

# Замеряем время скачивания файла
OUTPUT_DIR="downloaded_very_large_test_file"
mkdir -p "$OUTPUT_DIR"
echo "Скачивание файла с ID: $FILE_ID в $OUTPUT_DIR"
DOWNLOAD_START_TIME=$(date +%s.%N)
./client -download -id "$FILE_ID" -output "$OUTPUT_DIR" || cleanup_and_exit "Ошибка при скачивании файла"
DOWNLOAD_END_TIME=$(date +%s.%N)
DOWNLOAD_DURATION=$(echo "$DOWNLOAD_END_TIME - $DOWNLOAD_START_TIME" | bc)
DOWNLOAD_SPEED=$(echo "scale=2; ${TEST_FILE_SIZE:-104857600} / 1024 / 1024 / $DOWNLOAD_DURATION" | bc)
echo "Время скачивания: $DOWNLOAD_DURATION секунд"
echo "Скорость скачивания: $DOWNLOAD_SPEED МБ/с"

# Проверка наличия файла
OUTPUT_FILE="$OUTPUT_DIR/$TEST_FILE"
if [ ! -f "$OUTPUT_FILE" ]; then
    cleanup_and_exit "Файл не был скачан"
fi

# Проверка размера файла
ORIGINAL_SIZE=$(stat -c%s "$TEST_FILE")
DOWNLOADED_SIZE=$(stat -c%s "$OUTPUT_FILE")
echo "Размер оригинального файла: $ORIGINAL_SIZE байт"
echo "Размер скачанного файла: $DOWNLOADED_SIZE байт"

if [ "$ORIGINAL_SIZE" -ne "$DOWNLOADED_SIZE" ]; then
    cleanup_and_exit "Размер скачанного файла ($DOWNLOADED_SIZE) не совпадает с размером оригинального файла ($ORIGINAL_SIZE)"
fi

# Проверка контрольной суммы MD5
ORIGINAL_MD5=$(md5sum "$TEST_FILE" | awk '{print $1}')
DOWNLOADED_MD5=$(md5sum "$OUTPUT_FILE" | awk '{print $1}')
echo "MD5 оригинального файла: $ORIGINAL_MD5"
echo "MD5 скачанного файла: $DOWNLOADED_MD5"

if [ "$ORIGINAL_MD5" != "$DOWNLOADED_MD5" ]; then
    cleanup_and_exit "Контрольная сумма MD5 скачанного файла не совпадает с оригинальной"
fi

# Очистка временных файлов
rm -f "$TEMP_LIST_FILE"

echo "Тест пройден успешно! Файл загружен и скачан без ошибок."
echo "Оригинальный файл и скачанный файл имеют одинаковые размеры и контрольные суммы MD5."
exit 0