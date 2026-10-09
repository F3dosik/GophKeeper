package backend

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/F3dosik/GophKeeper/internal/client/export"
	"github.com/F3dosik/GophKeeper/internal/client/vault"
	"github.com/F3dosik/GophKeeper/internal/domain"
)

// maxExportFileSize — ограничение на размер импортируемого файла.
const maxExportFileSize = 1 << 30

// ImportFailure — секрет, который не удалось импортировать.
//
// Собственный тип вместо export.Failure: Wails создаёт в TypeScript пространство имён по
// имени Go-пакета, а «export» — зарезервированное слово, и сгенерированный код не компилируется.
type ImportFailure struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Error string `json:"error"`
}

// ImportResult — итог импорта для интерфейса.
type ImportResult struct {
	// SourceLogin и ExportedAt — откуда и когда сделан экспорт.
	SourceLogin string          `json:"sourceLogin"`
	ExportedAt  string          `json:"exportedAt"`
	Total       int             `json:"total"`
	Created     int             `json:"created"`
	Updated     int             `json:"updated"`
	Skipped     int             `json:"skipped"`
	Failed      []ImportFailure `json:"failed"`
	// Interrupted — импорт прерван (сессия истекла, сервер недоступен, квота); остальные
	// секреты не обработаны. Причина — в InterruptReason.
	Interrupted     bool   `json:"interrupted"`
	InterruptReason string `json:"interruptReason"`
}

// ExportVault выгружает все секреты в файл, зашифрованный паролем экспорта, и сохраняет
// его через системный диалог. Возвращает путь; пустой — пользователь отменил сохранение.
func (a *App) ExportVault(password string) (string, error) {
	v, err := a.currentVault()
	if err != nil {
		return "", err
	}
	if len(password) < export.MinPasswordLength {
		return "", validationError(export.ErrWeakPassword.Error())
	}

	secrets, err := v.ExportSecrets(a.context())
	if err != nil {
		return "", toUIError(err)
	}
	data, err := export.Encrypt(&export.Archive{
		ExportedAt: time.Now().UTC(),
		Login:      v.CurrentLogin(),
		Secrets:    secrets,
	}, password)
	if err != nil {
		return "", toUIError(err)
	}

	name := fmt.Sprintf("gophkeeper-%s-%s.gkx", v.CurrentLogin(), time.Now().Format("2006-01-02"))
	path, err := a.opts.UI.SaveFile("Экспорт хранилища", name)
	if err != nil || path == "" {
		return "", toUIError(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", toUIError(err)
	}
	return path, nil
}

// ChooseExportFile показывает диалог выбора файла экспорта; пустой путь — отмена.
func (a *App) ChooseExportFile() (*ChosenFile, error) {
	path, err := a.opts.UI.OpenFile("Файл экспорта GophKeeper", []FileFilter{
		{DisplayName: "Экспорт GophKeeper (*.gkx)", Pattern: "*.gkx"},
	})
	if err != nil || path == "" {
		return &ChosenFile{}, toUIError(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, toUIError(err)
	}
	return &ChosenFile{Path: path, Name: info.Name(), Size: info.Size()}, nil
}

// ImportVault импортирует секреты из файла экспорта в текущую учётку. Существующие
// секреты (то же имя и тип) пропускаются или, при overwrite, перезаписываются.
func (a *App) ImportVault(path, password string, overwrite bool) (*ImportResult, error) {
	v, err := a.currentVault()
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, toUIError(err)
	}
	if info.Size() > maxExportFileSize {
		return nil, validationError("Файл слишком большой для экспорта GophKeeper")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, toUIError(err)
	}

	archive, err := export.Decrypt(data, password)
	if err != nil {
		code := vault.CodeWrongPassword
		if !errors.Is(err, export.ErrWrongPassword) {
			code = CodeValidation
		}
		return nil, &Error{Code: code, Message: err.Error()}
	}

	report, err := export.Import(a.context(), v, archive, overwrite, isFatalImportError)
	result := &ImportResult{
		SourceLogin: archive.Login,
		ExportedAt:  formatTime(archive.ExportedAt),
		Total:       len(archive.Secrets),
		Created:     report.Created,
		Updated:     report.Updated,
		Skipped:     report.Skipped,
		Failed:      make([]ImportFailure, 0, len(report.Failed)),
	}
	for _, f := range report.Failed {
		result.Failed = append(result.Failed, ImportFailure{Name: f.Name, Type: string(f.Type), Error: f.Error})
	}
	if err != nil {
		result.Interrupted = true
		var uiErr *Error
		if errors.As(toUIError(err), &uiErr) {
			result.InterruptReason = uiErr.Message
		}
	}
	return result, nil
}

// isFatalImportError — ошибки, после которых продолжать импорт бессмысленно.
func isFatalImportError(err error) bool {
	return errors.Is(err, vault.ErrLocked) ||
		errors.Is(err, vault.ErrSessionExpired) ||
		errors.Is(err, domain.ErrUnavailable) ||
		errors.Is(err, domain.ErrResourceExhausted)
}
