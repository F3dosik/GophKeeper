package command

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/F3dosik/GophKeeper/internal/domain"
	"golang.org/x/term"
)

const (
	// Промпты мастер-пароля.
	promptMasterPassword        = "Введите мастер-пароль: "
	promptMasterPasswordConfirm = "Повторите мастер-пароль: "

	// Промпты SecretTypeCredentials.
	promptSecretCredLogin    = "Введите логин: "
	promptSecretCredPassword = "Введите пароль: "

	// Промпты SecretTypeText.
	promptSecretText = "Введите текст: "

	// Промпты SecretTypeBinary.
	promptSecretBinPath = "Путь к файлу: "

	//Промпты SecretTypeCard.
	promptSecretCardNumber = "Номер карты: "
	promptSecretCardHolder = "Держатель: "
	promptSecretCardExpiry = "Срок (MM/YY): "
	promptSecretCardCVV    = "CVV: "

	//Промпты SecretMetadata.
	promptSecretMD = "Введите дополнительную информацию для секрета (по желанию): "

	// minPasswordLength — минимальная допустимая длина мастер-пароля.
	minPasswordLength = 8
)

var (
	// ErrPasswordsMismatch возвращается, когда пароль и подтверждение не совпадают.
	ErrPasswordsMismatch = errors.New("пароли не совпадают")
	// ErrPasswordTooShort возвращается, когда пароль короче минимально допустимой длины.
	ErrPasswordTooShort = fmt.Errorf("пароль должен быть не короче %d символов", minPasswordLength)
	// ErrUnknownSecretType возвращается, когда тип секрета не определен.
	ErrUnknownSecretType = errors.New("неизвестный тип секрета")
)

// promptPassword выводит prompt и читает пароль со стандартного ввода без отображения
// введённых символов. Возвращает введённое значение без завершающего перевода строки.
func promptPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	pwd, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	return string(pwd), nil
}

// validatePassword проверяет, что пароль удовлетворяет минимальным требованиям
// (длина не короче minPasswordLength).
func validatePassword(password string) error {
	if len(password) < minPasswordLength {
		return ErrPasswordTooShort
	}
	return nil
}

// promptLine выводит prompt и читает одну строку со стандартного ввода.
// Возвращает строку без завершающих символов перевода строки (\n, \r\n).
func promptLine(prompt string) (string, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read line: %w", err)
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// promptSecretData интерактивно запрашивает у пользователя данные секрета в зависимости
// от его типа и возвращает их в виде JSON-сериализованного payload.
//
// Для credentials запрашивается логин и пароль (скрытый ввод);
// для text — одна строка текста;
// для binary — путь к файлу, содержимое которого читается с диска;
// для card — реквизиты карты; введённые данные валидируются через CardSecret.Validate.
func promptSecretData(t domain.SecretType) (json.RawMessage, error) {
	switch t {
	case domain.SecretTypeCredentials:
		login, err := promptLine(promptSecretCredLogin)
		if err != nil {
			return nil, err
		}
		password, err := promptPassword(promptSecretCredPassword)
		if err != nil {
			return nil, err
		}
		return json.Marshal(&domain.CredentialsSecret{Login: login, Password: password})
	case domain.SecretTypeText:
		text, err := promptLine(promptSecretText)
		if err != nil {
			return nil, err
		}
		return json.Marshal(&domain.TextSecret{Text: text})

	case domain.SecretTypeBinary:
		path, err := promptLine(promptSecretBinPath)
		if err != nil {
			return nil, err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read file: %w", err)
		}
		return json.Marshal(&domain.BinarySecret{Data: data})
	case domain.SecretTypeCard:
		number, err := promptLine(promptSecretCardNumber)
		if err != nil {
			return nil, err
		}
		holder, err := promptLine(promptSecretCardHolder)
		if err != nil {
			return nil, err
		}
		expiry, err := promptLine(promptSecretCardExpiry)
		if err != nil {
			return nil, err
		}
		cvv, err := promptPassword(promptSecretCardCVV)
		if err != nil {
			return nil, err
		}
		card := &domain.CardSecret{
			Number: number,
			Holder: holder,
			Expiry: expiry,
			CVV:    cvv,
		}
		if err := card.Validate(); err != nil {
			return nil, err
		}
		return json.Marshal(card)
	default:
		return nil, fmt.Errorf("%s: %s", ErrUnknownSecretType, t)
	}
}

// promptSecretMetadata запрашивает у пользователя опциональную метаинформацию к секрету
func promptSecretMetadata() (string, error) {
	return promptLine(promptSecretMD)
}
