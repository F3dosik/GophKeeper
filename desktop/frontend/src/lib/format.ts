export const secretTypes = [
  { value: 'credentials', label: 'Логин и пароль', icon: '🔑' },
  { value: 'text', label: 'Заметка', icon: '📝' },
  { value: 'card', label: 'Банковская карта', icon: '💳' },
  { value: 'binary', label: 'Файл', icon: '📎' },
] as const

export function typeLabel(type: string): string {
  return secretTypes.find((t) => t.value === type)?.label ?? type
}

export function typeIcon(type: string): string {
  return secretTypes.find((t) => t.value === type)?.icon ?? '•'
}

export function formatDate(iso: string): string {
  if (!iso) return ''
  return new Date(iso).toLocaleString('ru-RU', { dateStyle: 'medium', timeStyle: 'short' })
}

export function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} Б`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} КБ`
  return `${(bytes / 1024 / 1024).toFixed(1)} МБ`
}

/** Номер карты группами по 4 цифры. */
export function formatCardNumber(number: string): string {
  return number.replace(/\s+/g, '').replace(/(.{4})/g, '$1 ').trim()
}

/** Номер карты, где видны только последние 4 цифры. */
export function maskCardNumber(number: string): string {
  const digits = number.replace(/\s+/g, '')
  return '•••• ' + digits.slice(-4)
}
