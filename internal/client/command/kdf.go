package command

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/F3dosik/GophKeeper/internal/domain"
)

// withKDFCheck выполняет op. Если сервер прислал более слабые параметры Argon2id, чем
// запомнены на этом устройстве, объясняет, что это может значить, и после явного
// подтверждения забывает запомненные параметры и повторяет op. Пароль заново не
// запрашивается: op держит уже введённый.
func (c *Commands) withKDFCheck(login string, op func() error) error {
	err := op()
	var downgrade *domain.KDFDowngradeError
	if !errors.As(err, &downgrade) {
		return err
	}

	fmt.Fprintf(os.Stderr, `
ВНИМАНИЕ: сервер прислал для учётки %q более слабые параметры Argon2id, чем раньше:
  было  %s
  стало %s
Ключ аутентификации с такими параметрами дешевле перебирать, поэтому подменённый или
взломанный сервер мог бы так добыть его для подбора мастер-пароля. Пароль на сервер
не отправлен.

Продолжайте, только если вы сами сменили пароль с этими параметрами на другом
устройстве (gophkeeper auth passwd --kdf-time/--kdf-memory). Иначе сообщите
администратору сервера.

`, login, downgrade.Known, downgrade.Received)

	answer, err := promptLine("Принять новые параметры? Введите «да»: ")
	if err != nil {
		return err
	}
	if a := strings.ToLower(strings.TrimSpace(answer)); a != "да" && a != "yes" {
		return ErrNotConfirmed
	}
	if err := c.authService.ForgetKDFParams(login); err != nil {
		return err
	}
	return op()
}
