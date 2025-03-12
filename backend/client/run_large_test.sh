#!/bin/bash

# Компилируем клиент
echo "Компиляция клиента..."
go build -o client

if [ $? -ne 0 ]; then
    echo "Ошибка компиляции! Выход."
    exit 1
fi

# Проверяем, работает ли сервер
echo "Проверяем доступность сервера..."
(echo > /dev/tcp/localhost/50051) >/dev/null 2>&1
if [ $? -ne 0 ]; then
    echo "Сервер не доступен на порту 50051. Убедитесь, что сервер запущен."
    echo "Вы можете запустить сервер командой: cd ../docker && docker-compose up -d"
    exit 1
fi

echo "Примечание: Для работы тестов пользователь 'sas' должен существовать с паролем 'qwerty'."

# Создаем директорию для скачанных файлов
mkdir -p downloads

# Проверяем, существует ли тестовый файл
if [ ! -f "large_test_file.bin" ]; then
    echo "Создаем тестовый файл размером 10 МБ..."
    dd if=/dev/urandom of=large_test_file.bin bs=1M count=10
fi

echo ""
echo "===== ТЕСТИРОВАНИЕ ЗАГРУЗКИ БОЛЬШОГО ФАЙЛА ====="
UPLOAD_OUTPUT=$(./client -upload -file large_test_file.bin)
echo "$UPLOAD_OUTPUT"

# Извлекаем ID загруженного файла
FILE_ID=$(echo "$UPLOAD_OUTPUT" | grep -o 'ID: [a-zA-Z0-9-]*' | head -1 | cut -d ' ' -f2)
if [ -n "$FILE_ID" ]; then
    echo "Сохранен ID файла: $FILE_ID для последующего скачивания"
else
    echo "Не удалось получить ID файла. Проверка списка файлов..."
    # Получаем список файлов и выбираем первый с нужным именем
    FILES_OUTPUT=$(./client -list)
    echo "$FILES_OUTPUT"
    FILE_ID=$(echo "$FILES_OUTPUT" | grep "large_test_file.bin" | grep -o 'ID: [a-zA-Z0-9-]*' | head -1 | cut -d ' ' -f2)
    
    if [ -z "$FILE_ID" ]; then
        echo "Не удалось найти ID файла. Выход."
        exit 1
    else
        echo "Найден ID файла: $FILE_ID"
    fi
fi

echo ""
echo "===== СКАЧИВАНИЕ БОЛЬШОГО ФАЙЛА ====="
echo "Скачивание файла с ID: $FILE_ID"
START_TIME=$(date +%s.%N)
./client -download -id "$FILE_ID" -output "./downloads"
END_TIME=$(date +%s.%N)
ELAPSED=$(echo "$END_TIME - $START_TIME" | bc)
echo "Время скачивания: $ELAPSED секунд"

echo ""
echo "===== ПРОВЕРКА СКАЧАННОГО ФАЙЛА ====="
if [ -f "downloads/large_test_file.bin" ]; then
    echo "Файл успешно скачан!"
    ORIG_SIZE=$(stat -c%s "large_test_file.bin")
    DOWN_SIZE=$(stat -c%s "downloads/large_test_file.bin")
    echo "Размер оригинального файла: $ORIG_SIZE байт"
    echo "Размер скачанного файла: $DOWN_SIZE байт"
    
    if [ "$ORIG_SIZE" -eq "$DOWN_SIZE" ]; then
        echo "Размеры файлов совпадают!"
        # Проверяем контрольные суммы
        ORIG_MD5=$(md5sum large_test_file.bin | cut -d ' ' -f1)
        DOWN_MD5=$(md5sum downloads/large_test_file.bin | cut -d ' ' -f1)
        echo "MD5 оригинального файла: $ORIG_MD5"
        echo "MD5 скачанного файла: $DOWN_MD5"
        
        if [ "$ORIG_MD5" = "$DOWN_MD5" ]; then
            echo "Контрольные суммы совпадают! Тест успешно пройден."
        else
            echo "Контрольные суммы отличаются! Тест не пройден."
        fi
    else
        echo "Размеры файлов отличаются! Тест не пройден."
    fi
else
    echo "Скачанный файл не найден!"
fi 