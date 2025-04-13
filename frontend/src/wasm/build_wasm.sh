#!/bin/bash

# Скрипт для компиляции WebAssembly модуля шифрования

# Выводим сообщение о начале сборки
echo "Начинаем сборку WebAssembly модуля шифрования..."

# Определяем директории
CURRENT_DIR=$(pwd)
TARGET_DIR=${CURRENT_DIR}/build
FRONTEND_DIR=$(realpath "${CURRENT_DIR}/..")
PROJECT_ROOT=$(realpath "${FRONTEND_DIR}/..")

# Создаем директорию для сборки, если её нет
mkdir -p ${TARGET_DIR}

# Копируем необходимый файл wasm_exec.js из Go
WASM_EXEC_PATH=$(go env GOROOT)/misc/wasm/wasm_exec.js
if [ ! -f "${WASM_EXEC_PATH}" ]; then
    echo "Ошибка: Не найден файл ${WASM_EXEC_PATH}"
    echo "Убедитесь, что Go установлен корректно."
    exit 1
fi

echo "Копирование wasm_exec.js из ${WASM_EXEC_PATH}..."
cp "${WASM_EXEC_PATH}" ${TARGET_DIR}/wasm_exec.js

# Создаем временную директорию для сборки
TEMP_DIR=$(mktemp -d)
echo "Создана временная директория: ${TEMP_DIR}"

# Копируем нужные файлы во временную директорию
echo "Копирование файлов во временную директорию..."
mkdir -p ${TEMP_DIR}/wasm
mkdir -p ${TEMP_DIR}/cipher

# Копируем main.go
cp ${CURRENT_DIR}/main.go ${TEMP_DIR}/wasm/

# Копируем все файлы из cipher
cp ${FRONTEND_DIR}/cipher/*.go ${TEMP_DIR}/cipher/

# Создаем go.mod во временной директории
echo "Создание go.mod во временной директории..."
cat > ${TEMP_DIR}/go.mod << EOF
module wasm_build

go 1.21

require (
	enveloup v0.0.0
)

replace enveloup => ./
EOF

# Компилируем Go код в WebAssembly
echo "Компиляция main.go в WebAssembly..."
cd ${TEMP_DIR}

# Используем модуль напрямую
echo "Компиляция с модулем enveloup..."
GOOS=js GOARCH=wasm GO111MODULE=on go build -o ${TARGET_DIR}/main.wasm ./wasm/main.go

if [ $? -ne 0 ]; then
    echo "Ошибка компиляции WebAssembly модуля."
    exit 1
fi

# Проверяем результат
if [ -f "${TARGET_DIR}/main.wasm" ]; then
    echo "WebAssembly модуль успешно скомпилирован в ${TARGET_DIR}/main.wasm"
    echo "Размер файла: $(ls -lh ${TARGET_DIR}/main.wasm | awk '{print $5}')"
else
    echo "Ошибка: Файл ${TARGET_DIR}/main.wasm не был создан."
    exit 1
fi

# Удаляем временную директорию
echo "Удаление временной директории ${TEMP_DIR}..."
rm -rf ${TEMP_DIR}

echo "Сборка завершена успешно!"
echo "Файлы в директории ${TARGET_DIR}:"
ls -la ${TARGET_DIR} 