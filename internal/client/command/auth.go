package command

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// newAuthCmd создаёт команду для работы с аккаунтом пользователя.
func (c *Commands) newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "управление аккаунтом",
	}
	cmd.AddCommand(
		c.newRegisterCmd(),
		c.newLoginCmd(),
		c.newLogoutCmd(),
	)
	return cmd
}

// newRegisterCmd создаёт команду регистрации нового пользователя.
func (c *Commands) newRegisterCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "register <login>",
		Short: "Регистрация нового пользователя",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			login := args[0]

			password, err := promptPassword(promptMasterPassword)
			if err != nil {
				return err
			}

			if err := validatePassword(password); err != nil {
				return err
			}

			confirm, err := promptPassword(promptMasterPasswordConfirm)
			if err != nil {
				return err
			}

			if password != confirm {
				return fmt.Errorf("пароли не совпадают")
			}

			if err := c.authService.CreateUser(cmd.Context(), login, password); err != nil {
				return err
			}

			fmt.Println("Пользователь успешно зарегистрирован. Выполните вход командой 'login'.")
			return nil
		},
	}
}

// newLoginCmd создаёт команду входа в систему.
func (c *Commands) newLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login <login>",
		Short: "Вход в систему",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			login := args[0]

			password, err := promptPassword(promptMasterPassword)
			if err != nil {
				return err
			}

			if err := c.authService.Login(cmd.Context(), login, password); err != nil {
				return err
			}

			fmt.Println("Вход выполнен успешно.")
			return nil
		},
	}
}

// newLogoutCmd создаёт команду выхода из системы.
func (c *Commands) newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Выход из системы (удаление сохранённой сессии)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := os.Remove(c.cfg.SessionPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("logout: %w", err)
			}
			fmt.Println("Сессия удалена.")
			return nil
		},
	}
}
