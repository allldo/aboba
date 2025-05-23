#!/bin/sh

# Ждем пока база данных будет готова
echo "Ожидание готовности базы данных..."
until pg_isready -h postgres -p 5432 -U postgres; do
    echo "База данных не готова - ждем..."
    sleep 3
done

echo "База данных готова!"
sleep 5

echo "Выполняем миграции..."

# Выполняем миграции SQL файлов
for migration in /app/migrations/*.up.sql; do
    if [ -f "$migration" ]; then
        echo "Выполняем миграцию: $migration"
        PGPASSWORD=postgres psql -h postgres -U postgres -d mango -f "$migration" || true
    fi
done

echo "Миграции завершены! Запускаем приложение..."
exec ./main