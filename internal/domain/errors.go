package domain

import "errors"

// Сентинельные ошибки домена, возвращаемые репозиториями и сервисами.
// Вызывающий код должен сравнивать ошибки через errors.Is.
var (
	// ErrUserAlreadyExists возвращается при попытке создать пользователя с уже существующим логином.
	ErrUserAlreadyExists = errors.New("user already exists")

	// ErrUserNotFound возвращается, если пользователь с заданными параметрами не найден.
	ErrUserNotFound = errors.New("user not found")

	// ErrSecretAlreadyExists возвращается при попытке создать секрет с уже существующим blind index для данного пользователя.
	ErrSecretAlreadyExists = errors.New("secret already exists")

	// ErrSecretNotFound возвращается, если секрет с заданными параметрами не найден.
	ErrSecretNotFound = errors.New("secret not found")

	// ErrInvalidCredentials возвращается при несовпадении логина или пароля во время аутентификации.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrInvalidArgument возвращается при некорректных входных данных.
	ErrInvalidArgument = errors.New("invalid argument")

	// ErrNotFound возвращается когда запрашиваемый ресурс не найден.
	ErrNotFound = errors.New("not found")

	// ErrAlreadyExists возвращается когда ресурс уже существует.
	ErrAlreadyExists = errors.New("already exists")

	// ErrSecretTooLarge возвращается, если данные секрета превышают допустимый размер.
	ErrSecretTooLarge = errors.New("secret too large")

	// ErrSecretQuotaExceeded возвращается, если у пользователя достигнут лимит количества секретов.
	ErrSecretQuotaExceeded = errors.New("secret quota exceeded")

	// ErrSecretsChanged возвращается, если секреты пользователя изменились во время
	// смены пароля (создан, изменён или удалён секрет). Смену нужно повторить.
	ErrSecretsChanged = errors.New("secrets changed during password change")

	// ErrResourceExhausted возвращается клиенту, когда сервер отклонил запрос из-за лимита
	// (частоты запросов или квоты хранилища).
	ErrResourceExhausted = errors.New("limit exceeded")
)
