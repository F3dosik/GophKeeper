package backend

// UI — возможности оболочки приложения (Wails), которыми пользуется App:
// системные диалоги, буфер обмена и события для интерфейса. В тестах подменяется.
type UI interface {
	// OpenFile показывает диалог выбора файла; пустой путь — пользователь отменил выбор.
	OpenFile(title string, filters []FileFilter) (string, error)
	// SaveFile показывает диалог сохранения; пустой путь — пользователь отменил.
	SaveFile(title, defaultName string) (string, error)
	// ClipboardSet и ClipboardGet работают с системным буфером обмена.
	ClipboardSet(text string) error
	ClipboardGet() (string, error)
	// Emit отправляет событие в интерфейс.
	Emit(event string, data ...any)
}

// FileFilter — фильтр файлов в диалоге выбора, например {"Сертификаты", "*.crt;*.pem"}.
type FileFilter struct {
	DisplayName string
	Pattern     string
}

// EventLocked — событие блокировки хранилища; данные — причина (vault.LockReason).
const EventLocked = "vault:locked"
