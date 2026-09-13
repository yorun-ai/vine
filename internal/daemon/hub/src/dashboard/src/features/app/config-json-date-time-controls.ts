import { formatConfigDateTime, splitConfigDateTime } from './config-json-date-time.ts'
import type { ConfigDateTimeParts } from './config-json-date-time.ts'

export function appendConfigDateTimeControls(
  popup: HTMLElement,
  button: HTMLButtonElement,
  type: string,
  labels: Record<keyof ConfigDateTimeParts, string>,
  getValue: () => unknown,
  onChange: (value: string) => void,
) {
  const document = popup.ownerDocument
  const container = document.createElement('div')
  container.className = 'grid gap-2 p-1'
  const controls = new Map<keyof ConfigDateTimeParts, HTMLInputElement>()
  const base = type.replace(/\?$/, '')
  const keys: Array<keyof ConfigDateTimeParts> = []
  if (base !== 'localtime') {
    keys.push('date')
  }
  if (base !== 'localdate') {
    keys.push('time', 'fraction')
  }
  if (base === 'timestamp') {
    keys.push('offset')
  }
  for (const key of keys) {
    const label = document.createElement('label')
    label.className = 'grid gap-1'
    const caption = document.createElement('span')
    caption.className = 'text-muted-foreground'
    caption.textContent = labels[key]
    const input = document.createElement('input')
    input.type = key === 'date' ? 'date' : key === 'time' ? 'time' : 'text'
    if (key === 'time') {
      input.step = '1'
    }
    if (key === 'fraction') {
      input.inputMode = 'numeric'
      input.maxLength = 9
    }
    input.className = 'rounded border border-input bg-background px-2 py-1'
    input.setAttribute('aria-label', labels[key])
    input.dataset.configControl = ''
    controls.set(key, input)
    input.addEventListener('input', () => {
      const parts = splitConfigDateTime(type, getValue())
      for (const [name, control] of controls) {
        parts[name] = control.value
      }
      const value = formatConfigDateTime(type, parts)
      input.setAttribute('aria-invalid', String(value === null))
      if (!button.disabled && value !== null) {
        onChange(value)
      }
    })
    label.append(caption, input)
    container.append(label)
  }
  popup.append(container)
  popup.addEventListener('beforetoggle', (event) => {
    if (event.newState === 'open') {
      const parts = splitConfigDateTime(type, getValue())
      for (const [key, input] of controls) {
        input.value = parts[key]
        input.removeAttribute('aria-invalid')
      }
    }
  })
}
