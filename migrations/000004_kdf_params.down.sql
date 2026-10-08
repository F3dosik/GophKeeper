-- Откат возвращает общие параметры из кода (t=1). Пользователи, созданные или сменившие
-- пароль с другими параметрами, после отката не смогут войти.
ALTER TABLE users
    DROP COLUMN IF EXISTS kdf_time,
    DROP COLUMN IF EXISTS kdf_memory_kib,
    DROP COLUMN IF EXISTS kdf_threads;
