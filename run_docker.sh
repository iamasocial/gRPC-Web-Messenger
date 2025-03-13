#!/bin/bash

# Переходим в директорию docker
cd backend/docker

# Останавливаем текущие контейнеры (если они запущены)
docker-compose down

# Перед запуском создаем директорию для монтирования на хосте
mkdir -p ../../storage/files
mkdir -p ../../storage/temp

# Устанавливаем правильные права доступа
chmod -R 755 ../../storage

# Запускаем контейнеры
docker-compose up -d

echo "Сервер запущен. Файлы сохраняются в директории storage."
echo "Для просмотра логов выполните: docker-compose logs -f app" 