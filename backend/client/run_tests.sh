#!/bin/bash

# Устанавливаем цвета для вывода
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Запуск тестов обмена ключами Диффи-Хеллмана${NC}"
echo -e "${YELLOW}=========================================${NC}"

# Устанавливаем зависимости
echo -e "${YELLOW}Устанавливаем необходимые зависимости...${NC}"
go get -v google.golang.org/grpc
go get -v google.golang.org/grpc/credentials/insecure
go get -v google.golang.org/protobuf/proto

# Проверяем, запущен ли сервер по порту 50051
echo -e "${YELLOW}Проверяем доступность сервера...${NC}"
if docker ps | grep -q "0.0.0.0:50051->50051/tcp"; then
    echo -e "${GREEN}Сервер обнаружен в Docker контейнере${NC}"
else
    echo -e "${YELLOW}Пытаемся подключиться напрямую к порту 50051...${NC}"
    if nc -z -w 2 localhost 50051 2>/dev/null; then
        echo -e "${GREEN}Сервер доступен на localhost:50051${NC}"
    else
        echo -e "${RED}Внимание: Не удалось подтвердить доступность сервера на порту 50051${NC}"
        echo -e "${YELLOW}Продолжаем выполнение тестов, но они могут завершиться с ошибкой, если сервер недоступен${NC}"
        echo -e "${YELLOW}Для запуска сервера используйте:${NC}"
        echo -e "cd .. && go run cmd/main.go"
        
        read -p "Продолжить выполнение тестов? (y/n) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
fi

echo -e "${YELLOW}Запускаем все тесты в одном процессе...${NC}"

# Запускаем все тесты в одном процессе, чтобы сохранялось состояние между тестами
go test -v 2>&1 | tee test_output.log

# Проверяем результат тестов
if [ ${PIPESTATUS[0]} -eq 0 ]; then
    echo -e "${GREEN}Тесты успешно выполнены!${NC}"
else
    echo -e "${RED}Некоторые тесты не прошли. Смотрите журнал выше для деталей.${NC}"
    exit 1
fi

echo -e "${YELLOW}Результаты тестов сохранены в test_output.log${NC}" 