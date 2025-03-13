#!/bin/bash

# Функция для очистки и выхода при ошибке
cleanup_and_exit() {
    echo "Ошибка: $1"
    exit 1
}

# Функция для успешного завершения
success_exit() {
    echo "Тест пройден успешно! Файл загружен и скачан без ошибок."
    echo "Оригинальный файл и скачанный файл имеют одинаковые размеры и контрольные суммы MD5."
    exit 0
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
TEST_FILE="large_test_file.bin"
if [ ! -f "$TEST_FILE" ]; then
    echo "Создание тестового файла размером 10MB..."
    dd if=/dev/urandom of="$TEST_FILE" bs=1M count=10 || cleanup_and_exit "Не удалось создать тестовый файл"
fi

echo
echo "===== ТЕСТИРОВАНИЕ ЗАГРУЗКИ БОЛЬШОГО ФАЙЛА ====="

# Загрузка файла
echo "Загрузка файла на сервер..."
./client -upload -file "$TEST_FILE" || cleanup_and_exit "Ошибка при загрузке файла"

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

# Очистка перед скачиванием
OUTPUT_DIR="downloaded_$TEST_FILE"
# Удаляем директорию, если она существует
rm -rf "$OUTPUT_DIR"

# Скачивание файла по ID
echo "Скачивание файла с ID: $FILE_ID в $OUTPUT_DIR"
./client -download -id "$FILE_ID" -output "$OUTPUT_DIR" || cleanup_and_exit "Ошибка при скачивании файла"

# Подождем немного, чтобы файл успел записаться
sleep 2

# Проверка наличия директории
echo "Проверка наличия директории $OUTPUT_DIR..."
if [ ! -d "$OUTPUT_DIR" ]; then
    echo "Директория $OUTPUT_DIR не найдена. Текущие файлы:"
    ls -la
    cleanup_and_exit "Директория для скачанного файла не была создана"
fi

echo "Директория $OUTPUT_DIR существует, проверяем файлы внутри..."
ls -la "$OUTPUT_DIR"

# Полный путь к скачанному файлу должен быть: OUTPUT_DIR/TEST_FILE
DOWNLOADED_FILE="$OUTPUT_DIR/$TEST_FILE"
echo "Проверка наличия файла: $DOWNLOADED_FILE"

if [ ! -f "$DOWNLOADED_FILE" ]; then
    echo "Файл не найден в директории $OUTPUT_DIR. Содержимое директории:"
    ls -la "$OUTPUT_DIR"
    find "$OUTPUT_DIR" -type f -print
    cleanup_and_exit "Файл не был скачан или был сохранен с другим именем"
fi

echo "Найден файл: $DOWNLOADED_FILE"

# Проверка размера файла
ORIGINAL_SIZE=$(stat -c%s "$TEST_FILE")
DOWNLOADED_SIZE=$(stat -c%s "$DOWNLOADED_FILE")
echo "Размер оригинального файла: $ORIGINAL_SIZE байт"
echo "Размер скачанного файла: $DOWNLOADED_SIZE байт"

if [ "$ORIGINAL_SIZE" -ne "$DOWNLOADED_SIZE" ]; then
    cleanup_and_exit "Размер скачанного файла ($DOWNLOADED_SIZE) не совпадает с размером оригинального файла ($ORIGINAL_SIZE)"
fi

# Проверка контрольной суммы MD5
ORIGINAL_MD5=$(md5sum "$TEST_FILE" | awk '{print $1}')
DOWNLOADED_MD5=$(md5sum "$DOWNLOADED_FILE" | awk '{print $1}')
echo "MD5 оригинального файла: $ORIGINAL_MD5"
echo "MD5 скачанного файла: $DOWNLOADED_MD5"

if [ "$ORIGINAL_MD5" != "$DOWNLOADED_MD5" ]; then
    cleanup_and_exit "Контрольная сумма MD5 скачанного файла не совпадает с оригинальной"
fi

# Очистка временных файлов
rm -f "$TEMP_LIST_FILE"

# Тест успешно пройден
success_exit