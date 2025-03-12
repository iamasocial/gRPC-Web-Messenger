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

echo ""
echo "===== ТЕСТИРОВАНИЕ ЗАГРУЗКИ ФАЙЛА ====="
./client -upload -file test_file.txt

echo ""
echo "===== ПОЛУЧЕНИЕ СПИСКА ФАЙЛОВ ====="
./client -list

echo ""
echo "===== СКАЧИВАНИЕ ФАЙЛА ====="
FILE_ID="708d2b2c-cc62-4744-b42b-5ee5dbaf8e32"  # ID существующего файла
echo "Скачивание файла с ID: $FILE_ID"
./client -download -id "$FILE_ID"

echo ""
echo "===== ПРОВЕРКА СКАЧАННОГО ФАЙЛА ====="
if [ -f "downloads/test_file.txt" ]; then
    echo "Файл успешно скачан!"
    echo "Сравнение оригинального и скачанного файлов:"
    diff test_file.txt downloads/test_file.txt
    if [ $? -eq 0 ]; then
        echo "Файлы идентичны! Тест успешно пройден."
    else
        echo "Файлы отличаются! Тест не пройден."
    fi
else
    echo "Скачанный файл не найден!"
fi 