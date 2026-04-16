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
)
