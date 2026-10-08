package command

import (
	"errors"
	"fmt"
	"os"

	"github.com/F3dosik/GophKeeper/internal/domain"

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
	var all bool
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Выход из системы (отзыв токена на сервере и удаление сессии)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := os.Stat(c.cfg.SessionPath); os.IsNotExist(err) {
				fmt.Println("Вход не выполнен.")
				return nil
			}

			err := c.authService.Logout(cmd.Context(), all)
			if all && errors.Is(err, domain.ErrInvalidCredentials) {
				return fmt.Errorf("токен этого устройства недействителен (истёк или отозван): " +
					"выполните 'gophkeeper auth login' и повторите 'logout --all'")
			}
			if err != nil {
				return err
			}

			if all {
				fmt.Println("Выполнен выход на всех устройствах.")
			} else {
				fmt.Println("Выход выполнен, токен отозван.")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "отозвать токены на всех устройствах")
	return cmd
}
