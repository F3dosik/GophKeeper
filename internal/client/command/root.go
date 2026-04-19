// Package command реализует CLI-команды клиента GophKeeper на базе Cobra.
// Пакет содержит определение корневой команды и её подкоманд (auth, secret, version),
// а также вспомогательные фабрики сервисов, используемые при обработке команд.
package command

import (
	"context"
	"fmt"
	"os"

	"github.com/F3dosik/GophKeeper/internal/client/config"
	"github.com/F3dosik/GophKeeper/internal/client/grpcclient"
	"github.com/F3dosik/GophKeeper/internal/client/service"
	"github.com/F3dosik/GophKeeper/internal/client/session"
	"github.com/spf13/cobra"
)

// Version и BuildDate задают версию и дату сборки бинарного файла клиента.
// Значения переопределяются при компиляции через -ldflags.
var (
	Version   = "dev"
	BuildDate = "unknown"
)

// Commands содержит зависимости, необходимые для выполнения CLI-команд клиента.
// Экземпляр создаётся в main.go и передаётся в Execute для запуска корневой команды.
type Commands struct {
	authService   service.AuthService
	secretsClient grpcclient.SecretsClient
	cfg           *config.Config
}

// New создаёт Commands с переданными зависимостями: сервисом аутентификации,
// gRPC клиентом секретов и конфигурацией приложения.
func New(
	authSvc service.AuthService,
	secretsClient grpcclient.SecretsClient,
	cfg *config.Config,
) *Commands {
	return &Commands{
		authService:   authSvc,
		secretsClient: secretsClient,
		cfg:           cfg,
	}
}

// newSecretService деривирует мастер-ключ из (login, password) и создаёт
// SecretsService, готовый для операций шифрования/дешифрования секретов.
// Чистый конструктор без ввода-вывода — пригоден для модульных тестов.
func (c *Commands) newSecretService(ctx context.Context, login, password string) (service.SecretsService, error) {
	masterKey, err := c.authService.DeriveMasterKey(ctx, login, password)
	if err != nil {
		return nil, err
	}
	return service.NewSecretsService(c.secretsClient, masterKey)
}

// unlockSecretService — интерактивная обёртка над newSecretService: загружает
// сохранённую сессию, запрашивает мастер-пароль у пользователя и создаёт
// SecretsService. Возвращает понятную пользователю ошибку, если сессия отсутствует
// или повреждена.
func (c *Commands) unlockSecretService(ctx context.Context) (service.SecretsService, error) {
	sess, err := session.Load(c.cfg.SessionPath)
	if err != nil {
		return nil, fmt.Errorf("не выполнен вход, запустите 'gophkeeper auth login': %w", err)
	}

	password, err := promptPassword(promptMasterPassword)
	if err != nil {
		return nil, err
	}

	return c.newSecretService(ctx, sess.Login, password)
}

// newVersionCmd создаёт подкоманду, выводящую версию и дату сборки клиента.
func (c *Commands) newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Версия и дата сборки",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Version: %s\nBuild date: %s\n", Version, BuildDate)
		},
	}
}

// Execute собирает дерево Cobra-команд (auth, secret, version) и запускает
// обработку аргументов командной строки. Возвращает ошибку, если Cobra или
// исполнение команды завершились неуспешно.
func (c *Commands) Execute() error {
	root := &cobra.Command{
		Use:   "gophkeeper",
		Short: "Менеджер паролей",
		Long:  "GophKeeper — безопасное хранилище паролей и приватных данных.",
	}
	root.AddCommand(
		c.newVersionCmd(),
		c.newAuthCmd(),
		c.newSecretCmd(),
		c.newCompletionCmd(root),
	)
	return root.Execute()
}

func (c *Commands) newCompletionCmd(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Генерация скрипта автодополнения",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return root.GenBashCompletion(os.Stdout)
			case "zsh":
				return root.GenZshCompletion(os.Stdout)
			case "fish":
				return root.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return root.GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("неизвестный шелл: %s", args[0])
			}
		},
	}
}
