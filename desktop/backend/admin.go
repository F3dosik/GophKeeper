package backend

import (
	"context"
	"time"

	"github.com/F3dosik/GophKeeper/internal/client/grpcclient"
	"github.com/F3dosik/GophKeeper/internal/client/service"
	"github.com/F3dosik/GophKeeper/internal/domain"
	pb "github.com/F3dosik/GophKeeper/proto/gen"
	"google.golang.org/grpc"
)

// adminProbeTimeout — сколько ждать ответа сервера при определении административного порта.
// Проверка выполняется при сохранении настроек, поэтому таймаут короткий: недоступный
// адрес не должен надолго задерживать интерфейс.
const adminProbeTimeout = 2 * time.Second

// UserRow — пользователь в административной панели.
type UserRow struct {
	Login       string `json:"login"`
	CreatedAt   string `json:"createdAt"`
	SecretCount int    `json:"secretCount"`
	// TemporaryUntil — для временного пароля время, до которого им можно войти; пустое для постоянного.
	TemporaryUntil string `json:"temporaryUntil"`
	// TemporaryExpired — временный пароль истёк: войти им уже нельзя.
	TemporaryExpired bool `json:"temporaryExpired"`
	KDFTime          int  `json:"kdfTime"`
	KDFMemoryMiB     int  `json:"kdfMemoryMiB"`
}

// TemporaryUser — созданная учётка с временным паролем.
type TemporaryUser struct {
	Login     string `json:"login"`
	Password  string `json:"password"`
	ExpiresAt string `json:"expiresAt"`
}

// adminConn — соединение с административным портом.
type adminConn struct {
	conn   *grpc.ClientConn
	admin  grpcclient.AdminClient
	auth   service.AuthService
	active bool
}

// openAdminLocked подключается к серверу как администратор и проверяет, принимает ли
// он административные методы. На публичном порту сервиса Admin нет — тогда
// административный режим выключен. Вызывается под a.mu.
func (a *App) openAdminLocked(caPEM []byte) {
	a.closeAdminLocked()

	conn, err := grpcclient.DialWithOptions(a.settings.ServerAddress, grpcclient.DialOptions{
		CACertPEM: caPEM,
		Insecure:  a.opts.Insecure,
	})
	if err != nil {
		return
	}
	adm := &adminConn{
		conn:  conn,
		admin: grpcclient.NewAdminClient(pb.NewAdminClient(conn)),
		// Временную учётку создаёт AuthService; сессия и токен ему для этого не нужны.
		auth: service.NewAuthService(grpcclient.NewAuthClient(pb.NewAuthClient(conn)), "", nil),
	}

	ctx, cancel := context.WithTimeout(a.ctx, adminProbeTimeout)
	defer cancel()
	_, err = adm.admin.ListUsers(ctx)
	adm.active = err == nil
	a.admin = adm
}

// closeAdminLocked закрывает административное соединение. Вызывается под a.mu.
func (a *App) closeAdminLocked() {
	if a.admin != nil {
		_ = a.admin.conn.Close()
		a.admin = nil
	}
}

// currentAdmin возвращает административный клиент или ошибку, если приложение
// подключено не к административному порту.
func (a *App) currentAdmin() (*adminConn, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.admin == nil || !a.admin.active {
		return nil, &Error{Code: "NOT_SUPPORTED", Message: "Приложение подключено не к административному порту сервера"}
	}
	return a.admin, nil
}

// AdminListUsers возвращает список пользователей.
func (a *App) AdminListUsers() ([]UserRow, error) {
	adm, err := a.currentAdmin()
	if err != nil {
		return nil, err
	}
	users, err := adm.admin.ListUsers(a.context())
	if err != nil {
		return nil, toUIError(err)
	}
	rows := make([]UserRow, 0, len(users))
	now := time.Now()
	for _, u := range users {
		row := UserRow{
			Login:        u.Login,
			CreatedAt:    formatTime(u.CreatedAt),
			SecretCount:  u.SecretCount,
			KDFTime:      int(u.KDF.Time),
			KDFMemoryMiB: int(u.KDF.MemoryKiB / 1024),
		}
		if u.PasswordExpiresAt != nil {
			row.TemporaryUntil = formatTime(*u.PasswordExpiresAt)
			row.TemporaryExpired = now.After(*u.PasswordExpiresAt)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// AdminCreateTemporaryUser создаёт учётку со сгенерированным временным паролем.
// Пароль возвращается один раз: передайте его пользователю, при первом входе он задаст свой.
func (a *App) AdminCreateTemporaryUser(login string) (*TemporaryUser, error) {
	adm, err := a.currentAdmin()
	if err != nil {
		return nil, err
	}
	if err := checkLogin(login); err != nil {
		return nil, err
	}
	password, expiresAt, err := adm.auth.CreateTemporaryUser(a.context(), login, domain.DefaultKDFParams)
	if err != nil {
		return nil, toUIError(err)
	}
	return &TemporaryUser{Login: login, Password: password, ExpiresAt: formatTime(expiresAt)}, nil
}

// AdminRevokeSessions завершает все сессии пользователя.
func (a *App) AdminRevokeSessions(login string) error {
	adm, err := a.currentAdmin()
	if err != nil {
		return err
	}
	return toUIError(adm.admin.RevokeSessions(a.context(), login))
}

// AdminDeleteUser удаляет пользователя со всеми секретами.
func (a *App) AdminDeleteUser(login string) error {
	adm, err := a.currentAdmin()
	if err != nil {
		return err
	}
	return toUIError(adm.admin.DeleteUser(a.context(), login))
}

// checkLogin проверяет логин новой учётки на клиенте.
func checkLogin(login string) error {
	if login == "" || len(login) > 64 {
		return validationError("Логин — от 1 до 64 символов")
	}
	return nil
}
