-- Временные пароли: если password_expires_at задан, пароль выдан администратором,
-- годен до указанного времени и после входа должен быть сменён. NULL — обычный пароль.
ALTER TABLE users ADD COLUMN password_expires_at TIMESTAMPTZ;
