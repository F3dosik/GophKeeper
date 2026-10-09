package export

import (
	"context"
	"errors"

	"github.com/F3dosik/GophKeeper/internal/domain"
)

// Saver сохраняет секреты в хранилище; реализуется service.SecretsService и vault.Vault.
type Saver interface {
	CreateSecret(ctx context.Context, payload *domain.SecretPayload) error
	UpdateSecret(ctx context.Context, payload *domain.SecretPayload) error
}

// Failure — секрет, который не удалось импортировать.
type Failure struct {
	Name  string            `json:"name"`
	Type  domain.SecretType `json:"type"`
	Error string            `json:"error"`
}

// Report — итог импорта.
type Report struct {
	Created int       `json:"created"`
	Updated int       `json:"updated"`
	Skipped int       `json:"skipped"`
	Failed  []Failure `json:"failed"`
}

// Import сохраняет секреты архива. Секрет с тем же именем и типом, что уже есть в
// хранилище, пропускается или, при overwrite, перезаписывается. Ошибка отдельного
// секрета не останавливает импорт и попадает в Report.Failed; прерывают импорт только
// ошибки, после которых продолжать бессмысленно (хранилище заблокировано, сессия
// истекла, сервер недоступен) — они возвращаются вместе с частичным отчётом.
func Import(ctx context.Context, saver Saver, archive *Archive, overwrite bool, fatal func(error) bool) (Report, error) {
	var report Report
	for i := range archive.Secrets {
		payload := &archive.Secrets[i]

		err := saver.CreateSecret(ctx, payload)
		switch {
		case err == nil:
			report.Created++
			continue
		case errors.Is(err, domain.ErrAlreadyExists) && !overwrite:
			report.Skipped++
			continue
		case errors.Is(err, domain.ErrAlreadyExists):
			if err = saver.UpdateSecret(ctx, payload); err == nil {
				report.Updated++
				continue
			}
		}

		if fatal != nil && fatal(err) {
			return report, err
		}
		report.Failed = append(report.Failed, Failure{Name: payload.Name, Type: payload.Type, Error: err.Error()})
	}
	return report, nil
}
