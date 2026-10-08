package command

import (
	"errors"
	"fmt"

	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/spf13/cobra"
)

// newAdminCmd создаёт группу административных команд. Они работают только через
// административный порт сервера (GOPHKEEPER_SERVER=localhost:<ADMIN_PORT>).
func (c *Commands) newAdminCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Администрирование сервера (через административный порт)",
		Long: "Административные команды. Работают только при подключении к административному\n" +
			"порту сервера (ADMIN_PORT), который доступен лишь с машины сервера:\n\n" +
			"  GOPHKEEPER_SERVER=localhost:50052 gophkeeper admin delete-user bob",
	}
	cmd.AddCommand(c.newDeleteUserCmd())
	return cmd
}

// newDeleteUserCmd создаёт команду удаления пользователя администратором.
func (c *Commands) newDeleteUserCmd() *cobra.Command {
	var skipConfirm bool
	cmd := &cobra.Command{
		Use:   "delete-user <login>",
		Short: "Безвозвратно удалить пользователя со всеми секретами",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			login := args[0]

			if !skipConfirm {
				fmt.Printf("Пользователь %s и все его секреты будут удалены без возможности восстановления.\n", login)
				if err := confirmByTyping(login); err != nil {
					return err
				}
			}

			err := c.adminClient.DeleteUser(cmd.Context(), login)
			switch {
			case errors.Is(err, domain.ErrNotSupported):
				return fmt.Errorf("сервер не принимает административные команды на этом порту: " +
					"подключитесь к ADMIN_PORT (например, GOPHKEEPER_SERVER=localhost:50052)")
			case errors.Is(err, domain.ErrNotFound):
				return fmt.Errorf("пользователь %s не найден", login)
			case err != nil:
				return err
			}

			fmt.Printf("Пользователь %s удалён.\n", login)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&skipConfirm, "yes", "y", false, "Пропустить подтверждение")
	return cmd
}
