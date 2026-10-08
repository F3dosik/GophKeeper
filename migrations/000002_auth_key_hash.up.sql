-- Переход с хранения masterKey на хранение SHA-256(authKey), где authKey = HKDF-SHA256(masterKey, info="auth").
-- Раньше в password_hash лежал сам masterKey, из которого клиент выводит ключи шифрования,
-- т.е. владелец БД мог расшифровать все секреты. Пересчитываем хеши существующих пользователей,
-- чтобы их старые пароли продолжили работать, а masterKey исчез из БД.
--
-- HKDF с пустой солью и длиной вывода 32 байта (один блок):
--   PRK = HMAC-SHA256(key = 32 нулевых байта, msg = masterKey)
--   OKM = HMAC-SHA256(key = PRK, msg = info || 0x01)
CREATE EXTENSION IF NOT EXISTS pgcrypto;

UPDATE users
SET password_hash = digest(
    hmac(
        convert_to('auth', 'UTF8') || '\x01'::bytea,
        hmac(password_hash, decode(repeat('00', 32), 'hex'), 'sha256'),
        'sha256'
    ),
    'sha256'
);
