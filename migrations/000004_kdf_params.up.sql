-- Параметры Argon2id у каждого пользователя.
-- Существующие пользователи получают параметры, которыми были зашиты в коде до этой миграции
-- (t=1, m=64 MiB, p=4), поэтому их ключи не меняются. После заполнения значения по умолчанию
-- удаляются: при создании пользователя параметры должны передаваться явно.
ALTER TABLE users
    ADD COLUMN kdf_time INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN kdf_memory_kib INTEGER NOT NULL DEFAULT 65536,
    ADD COLUMN kdf_threads SMALLINT NOT NULL DEFAULT 4;

ALTER TABLE users
    ALTER COLUMN kdf_time DROP DEFAULT,
    ALTER COLUMN kdf_memory_kib DROP DEFAULT,
    ALTER COLUMN kdf_threads DROP DEFAULT;
