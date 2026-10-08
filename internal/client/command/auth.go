package command

import (
	"errors"
	"fmt"
	"os"

	"github.com/F3dosik/GophKeeper/internal/client/session"
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
		c.newPasswdCmd(),
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

			password, err := promptNewPassword(promptMasterPassword)
			if err != nil {
				return err
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

// newPasswdCmd создаёт команду смены мастер-пароля.
// Все секреты перешифровываются ключом от нового пароля; на сервер они уходят
// одной транзакцией, поэтому при сбое пароль и данные остаются прежними.
func (c *Commands) newPasswdCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "passwd",
		Short: "Смена мастер-пароля с перешифровкой всех секретов",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := session.Load(c.cfg.SessionPath)
			if err != nil {
				return fmt.Errorf("не выполнен вход, запустите 'gophkeeper auth login': %w", err)
			}

			oldPassword, err := promptPassword(promptCurrentPassword)
			if err != nil {
				return err
			}
			secretSvc, err := c.newSecretService(cmd.Context(), sess.Login, oldPassword)
			if err != nil {
				return err
			}

			newPassword, err := promptNewPassword(promptNewMasterPassword)
			if err != nil {
				return err
			}
			if newPassword == oldPassword {
				return ErrSamePassword
			}

			reencrypt := func(newMasterKey []byte) ([]domain.ReencryptedSecret, error) {
				return secretSvc.Reencrypt(cmd.Context(), newMasterKey)
			}
			err = c.authService.ChangePassword(cmd.Context(), sess.Login, oldPassword, newPassword, reencrypt)
			if errors.Is(err, domain.ErrSecretsChanged) {
				return fmt.Errorf("секреты изменились во время смены пароля (например, с другого устройства), " +
					"пароль не изменён; повторите 'gophkeeper auth passwd'")
			}
			if err != nil {
				return err
			}

			fmt.Println("Пароль изменён. На остальных устройствах выполните 'gophkeeper auth login'.")
			return nil
		},
	}
}
