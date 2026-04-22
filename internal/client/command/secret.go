package command

import (
	"fmt"
	"strings"

	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/spf13/cobra"
)

// newSecretCmd создаёт команду-контейнер для работы с секретами.
// Объединяет подкоманды: create, get, update, delete, list.
func (c *Commands) newSecretCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secret",
		Short: "Управление секретами",
	}
	cmd.AddCommand(
		c.newCreateCmd(),
		c.newUpdateCmd(),
		c.newGetCmd(),
		c.newListCmd(),
		c.newDeleteCmd(),
	)
	return cmd
}

// newCreateCmd создаёт подкоманду создания нового секрета.
// Тип секрета определяет, какие поля будут запрошены интерактивно.
func (c *Commands) newCreateCmd() *cobra.Command {
	var name, secretType string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Создать новый секрет",
		RunE: func(cmd *cobra.Command, args []string) error {
			t, err := domain.ParseSecretType(secretType)
			if err != nil {
				return err
			}

			secretSvc, err := c.unlockSecretService(cmd.Context())
			if err != nil {
				return err
			}

			data, err := promptSecretData(t)
			if err != nil {
				return err
			}

			metadata, err := promptSecretMetadata()
			if err != nil {
				return err
			}

			if err := secretSvc.CreateSecret(cmd.Context(), &domain.SecretPayload{
				Name:     name,
				Type:     t,
				Data:     data,
				Metadata: metadata,
			}); err != nil {
				return err
			}
			fmt.Println("Секрет создан")
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Имя секрета (обязательно)")
	cmd.Flags().StringVar(&secretType, "type", "", "Тип: credentials|text|card|binary (обязательно)")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("type")
	return cmd
}

// newUpdateCmd создаёт подкоманду изменения секрета.
// Тип секрета определяет, какие поля будут запрошены интерактивно.
func (c *Commands) newUpdateCmd() *cobra.Command {
	var name, secretType string
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Изменить секрет",
		RunE: func(cmd *cobra.Command, args []string) error {
			t, err := domain.ParseSecretType(secretType)
			if err != nil {
				return err
			}

			secretSvc, err := c.unlockSecretService(cmd.Context())
			if err != nil {
				return err
			}

			data, err := promptSecretData(t)
			if err != nil {
				return err
			}

			metadata, err := promptSecretMetadata()
			if err != nil {
				return err
			}

			if err := secretSvc.UpdateSecret(cmd.Context(), &domain.SecretPayload{
				Name:     name,
				Type:     t,
				Data:     data,
				Metadata: metadata,
			}); err != nil {
				return err
			}
			fmt.Println("Секрет обновлен")
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Имя секрета (обязательно)")
	cmd.Flags().StringVar(&secretType, "type", "", "Тип: credentials|text|card|binary (обязательно)")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("type")
	return cmd
}

// newGetCmd создаёт подкоманду получения секрета.
// Поддерживает форматирование (--json) и сохранение в файл (--output).
// Для binary с --output записывает сырые байты, для остальных типов — отформатированный текст.
func (c *Commands) newGetCmd() *cobra.Command {
	var name, secretType string
	var jsonOutput bool
	var outputPath string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Получить секрет",
		RunE: func(cmd *cobra.Command, args []string) error {
			t, err := domain.ParseSecretType(secretType)
			if err != nil {
				return err
			}

			secretSvc, err := c.unlockSecretService(cmd.Context())
			if err != nil {
				return err
			}

			info, err := secretSvc.GetSecret(cmd.Context(), name, t)
			if err != nil {
				return err
			}

			return WriteSecret(info, OutputOptions{JSON: jsonOutput, OutputPath: outputPath})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Имя секрета (обязательно)")
	cmd.Flags().StringVar(&secretType, "type", "", "Тип: credentials|text|card|binary (обязательно)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Вывести в формате JSON")
	cmd.Flags().StringVar(&outputPath, "output", "", "Сохранить в файл (для binary — сырые данные)")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("type")

	return cmd
}

// newListCmd создаёт подкоманду вывода списка секретов пользователя.
// Выводит таблицу с метаинформацией (имя, тип, метаданные, дата обновления), без значений секретов.
// Поддерживает фильтрацию по типам (--type).
func (c *Commands) newListCmd() *cobra.Command {
	var filterType string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Список секретов",
		RunE: func(cmd *cobra.Command, args []string) error {
			secretSvc, err := c.unlockSecretService(cmd.Context())
			if err != nil {
				return err
			}

			secrets, err := secretSvc.ListSecrets(cmd.Context())
			if err != nil {
				return err
			}

			return WriteSecretList(secrets, filterType)
		},
	}
	cmd.Flags().StringVar(&filterType, "type", "", "Фильтр по типам: credentials|text|card|binary")
	return cmd
}

// newDeleteCmd создаёт подкоманду удаления секрета.
// По умолчанию запрашивает интерактивное подтверждение; флаг --yes пропускает его.
func (c *Commands) newDeleteCmd() *cobra.Command {
	var name, secretType string
	var skipConfirm bool
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Удаление секрета",
		RunE: func(cmd *cobra.Command, args []string) error {
			t, err := domain.ParseSecretType(secretType)
			if err != nil {
				return err
			}

			secretSvc, err := c.unlockSecretService(cmd.Context())
			if err != nil {
				return err
			}

			if !skipConfirm {
				confirm, err := promptLine(fmt.Sprintf("Удалить секрет %q типа %s? [y/N]: ", name, t))
				if err != nil {
					return err
				}
				if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
					fmt.Println("Отменено")
					return nil
				}
			}

			if err := secretSvc.DeleteSecret(cmd.Context(), name, t); err != nil {
				return err
			}
			fmt.Printf("Секрет %q (%s) удалён\n", name, t)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Имя секрета (обязательно)")
	cmd.Flags().StringVar(&secretType, "type", "", "Тип: credentials|text|card|binary (обязательно)")
	cmd.Flags().BoolVarP(&skipConfirm, "yes", "y", false, "Пропустить подтверждение")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("type")
	return cmd
}
