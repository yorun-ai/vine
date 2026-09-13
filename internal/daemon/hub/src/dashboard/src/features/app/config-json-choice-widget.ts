import { isConfigDateTime } from './config-json-date-time.ts'
import { appendConfigDateTimeControls } from './config-json-date-time-controls.ts'
import { configDurationUnits, formatConfigDuration, splitConfigDuration } from './config-json-duration.ts'
import JSON5 from 'json5'
import { Decoration, EditorView, StateField, WidgetType } from '@uiw/react-codemirror'
import type { EditorState } from '@uiw/react-codemirror'
import type { ConfigJsonField, ConfigValueRange } from './config-json-document.ts'
import { getConfigJsonChoices, isConfigJsonMultiChoice, toggleConfigJsonChoice } from './config-json-choice.ts'
import type { ConfigJsonChoice } from './config-json-choice.ts'

interface ConfigEditorLabels {
  amount: string
  unit: string
  date: string
  time: string
  fraction: string
  offset: string
}

const defaultConfigEditorLabels: ConfigEditorLabels = { amount: 'Value', unit: 'Unit', date: 'Date', time: 'Time', fraction: 'Fractional seconds', offset: 'UTC offset' }

export class ConfigJsonChoiceWidget extends WidgetType {
  readonly dateTimeType: string
  readonly duration: boolean
  readonly durationLabels: ConfigEditorLabels
  readonly name: string
  readonly value: string
  readonly choices: ReadonlyArray<ConfigJsonChoice>
  readonly multiple: boolean
  readonly dirty: boolean
  readonly error: string
  readonly readOnly: boolean
  readonly ranges: StateField<Array<ConfigValueRange>>

  constructor(
    name: string,
    value: string,
    choices: ReadonlyArray<ConfigJsonChoice>,
    multiple: boolean,
    readOnly: boolean,
    ranges: StateField<Array<ConfigValueRange>>,
    error = '',
    duration = false,
    durationLabels = defaultConfigEditorLabels,
    dirty = false,
    dateTimeType = '',
  ) {
    super()
    this.dateTimeType = dateTimeType
    this.duration = duration
    this.durationLabels = durationLabels
    this.dirty = dirty
    this.error = error
    this.name = name
    this.value = value
    this.choices = choices
    this.multiple = multiple
    this.readOnly = readOnly
    this.ranges = ranges
  }

  eq(other: ConfigJsonChoiceWidget) {
    return this.dateTimeType === other.dateTimeType && this.duration === other.duration && JSON.stringify(this.durationLabels) === JSON.stringify(other.durationLabels) &&
      this.name === other.name && this.value === other.value && this.error === other.error && this.dirty === other.dirty &&
      this.multiple === other.multiple && this.readOnly === other.readOnly &&
      this.ranges === other.ranges && JSON.stringify(this.choices) === JSON.stringify(other.choices)
  }

  updateDOM(dom: HTMLElement) {
    if (dom.dataset.dateTimeType !== this.dateTimeType || dom.dataset.name !== this.name || dom.dataset.choices !== JSON.stringify(this.choices) ||
      dom.dataset.multiple !== String(this.multiple) ||
      dom.dataset.duration !== String(this.duration) || dom.dataset.durationLabels !== JSON.stringify(this.durationLabels)) {
      return false
    }
    const value: unknown = JSON.parse(this.value)
    const button = dom.querySelector('button')!
    button.textContent = `${Array.isArray(value) ? `[${value.join(', ')}]` : String(value)} ▾`
    button.disabled = this.readOnly
    button.style.borderColor = this.error ? 'var(--destructive)' : this.dirty ? 'var(--color-amber-400)' : ''
    button.style.backgroundColor = this.error ? 'color-mix(in srgb, var(--destructive) 15%, var(--background))' : this.dirty ? 'var(--color-amber-100)' : ''
    for (const input of dom.querySelectorAll<HTMLInputElement>('input[type=checkbox], input[type=radio]')) {
      const item: unknown = JSON.parse(input.value)
      input.checked = Array.isArray(value) ? value.includes(item) : value === item
      input.disabled = this.readOnly
    }
    for (const control of dom.querySelectorAll<HTMLInputElement | HTMLSelectElement>('[data-config-control]')) {
      control.disabled = this.readOnly
    }
    const popup = dom.querySelector<HTMLElement>('[popover]')!
    if (this.readOnly && popup.matches(':popover-open')) {
      popup.hidePopover()
    }
    return true
  }

  toDOM(view: EditorView) {
    const document = view.dom.ownerDocument
    const root = document.createElement('span')
    root.dataset.dateTimeType = this.dateTimeType
    root.dataset.duration = String(this.duration)
    root.dataset.durationLabels = JSON.stringify(this.durationLabels)
    root.dataset.name = this.name
    root.dataset.choices = JSON.stringify(this.choices)
    root.dataset.multiple = String(this.multiple)
    const button = document.createElement('button')
    button.type = 'button'
    button.className = 'cm-config-choice'
    button.setAttribute('aria-label', this.name)
    button.setAttribute('aria-haspopup', 'dialog')
    button.setAttribute('aria-expanded', 'false')
    const popup = document.createElement('div')
    popup.popover = 'auto'
    popup.className = 'rounded-md border border-border bg-popover p-2 text-[13px] text-popover-foreground shadow-md'
    popup.setAttribute('role', 'dialog')
    popup.setAttribute('aria-label', this.name)
    Object.assign(popup.style, { position: 'fixed', margin: '0', maxHeight: 'min(20rem, 60vh)', overflowY: 'auto' })
    button.addEventListener('click', () => {
      const bounds = button.getBoundingClientRect()
      popup.style.left = `${Math.max(8, Math.min(bounds.left, document.documentElement.clientWidth - 328))}px`
      popup.style.top = `${bounds.bottom + 4}px`
      popup.style.maxWidth = 'min(32rem, calc(100vw - 16px))'
      popup.togglePopover()
    })
    popup.addEventListener('toggle', () => {
      button.setAttribute('aria-expanded', String(popup.matches(':popover-open')))
    })
    for (const choice of this.duration ? [] : this.choices) {
      const label = document.createElement('label')
      label.className = 'flex cursor-pointer items-center gap-2 rounded px-2 py-1.5 hover:bg-accent'
      const input = document.createElement('input')
      input.type = this.multiple ? 'checkbox' : 'radio'
      input.value = choice.value
      input.style.accentColor = 'var(--primary)'
      input.setAttribute('aria-label', choice.label)
      const title = document.createElement('span')
      title.textContent = choice.label
      label.append(input, title)
      if (choice.description) {
        const description = document.createElement('span')
        description.className = 'text-muted-foreground'
        description.textContent = choice.description
        label.append(description)
      }
      input.addEventListener('change', () => {
        if (button.disabled) {
          return
        }
        const range = view.state.field(this.ranges).find((item) => item.name === this.name)!
        const current: unknown = JSON5.parse(view.state.doc.sliceString(range.from, range.to))
        const next = this.multiple ? toggleConfigJsonChoice(current, choice.value) : JSON.parse(choice.value)
        if (!this.multiple) {
          popup.hidePopover()
          button.focus()
        }
        view.dispatch({ changes: { from: range.from, to: range.to, insert: JSON.stringify(next) } })
      })
      popup.append(label)
    }
    if (this.duration) {
      const custom = document.createElement('div')
      custom.className = 'flex items-center gap-2'
      const amount = document.createElement('input')
      amount.type = 'number'
      amount.step = 'any'
      amount.className = 'w-24 rounded border border-input bg-background px-2 py-1'
      amount.setAttribute('aria-label', this.durationLabels.amount)
      amount.dataset.configControl = ''
      const unit = document.createElement('select')
      unit.className = 'rounded border border-input bg-background px-2 py-1'
      unit.setAttribute('aria-label', this.durationLabels.unit)
      unit.dataset.configControl = ''
      for (const name of configDurationUnits) {
        const option = document.createElement('option')
        option.value = name
        option.textContent = name
        unit.append(option)
      }
      const initial = splitConfigDuration(JSON.parse(this.value))
      amount.value = initial.amount
      unit.value = initial.unit
      const updateDuration = () => {
        const duration = formatConfigDuration(amount.value, unit.value)
        amount.setAttribute('aria-invalid', String(duration === null && unit.value !== 'null'))
        if (button.disabled || (duration === null && unit.value !== 'null')) {
          return
        }
        const range = view.state.field(this.ranges).find((item) => item.name === this.name)!
        const next = JSON.stringify(unit.value === 'null' ? null : duration)
        if (view.state.doc.sliceString(range.from, range.to) !== next) {
          view.dispatch({ changes: { from: range.from, to: range.to, insert: next } })
        }
      }
      if (this.choices.some((choice) => choice.value === 'null')) {
        const option = document.createElement('option')
        option.value = 'null'
        option.textContent = 'null'
        unit.append(option)
      }
      amount.addEventListener('input', updateDuration)
      unit.addEventListener('change', updateDuration)
      custom.append(amount, unit)
      popup.append(custom)
      popup.addEventListener('beforetoggle', (event) => {
        if (event.newState === 'open') {
          const range = view.state.field(this.ranges).find((item) => item.name === this.name)!
          const value: unknown = JSON5.parse(view.state.doc.sliceString(range.from, range.to))
          const current = splitConfigDuration(value)
          amount.value = current.amount
          unit.value = value === null && this.choices.some((choice) => choice.value === 'null') ? 'null' : current.unit
          amount.removeAttribute('aria-invalid')
        }
      })
    }
    if (this.dateTimeType) {
      appendConfigDateTimeControls(popup, button, this.dateTimeType, this.durationLabels, () => {
        const range = view.state.field(this.ranges).find((item) => item.name === this.name)!
        return JSON5.parse(view.state.doc.sliceString(range.from, range.to))
      }, (value) => {
        const range = view.state.field(this.ranges).find((item) => item.name === this.name)!
        const next = JSON.stringify(value)
        if (view.state.doc.sliceString(range.from, range.to) !== next) {
          view.dispatch({ changes: { from: range.from, to: range.to, insert: next } })
        }
      })
    }
    root.append(button, popup)
    this.updateDOM(root)
    return root
  }

  ignoreEvent() {
    return true
  }
}

export function createConfigChoiceExtension(
  fields: ReadonlyArray<ConfigJsonField>,
  readOnly: boolean,
  ranges: StateField<Array<ConfigValueRange>>,
  errors: ReadonlyMap<string, string> = new Map(),
  durationLabels: ConfigEditorLabels = defaultConfigEditorLabels,
  dirtyFields: ReadonlySet<string> = new Set(),
) {
  function decorations(state: EditorState) {
    const marks: Array<ReturnType<Decoration['range']>> = []
    for (const range of state.field(ranges)) {
      const field = fields.find((item) => item.name === range.name)
      if (!field) {
        continue
      }
      const choices = getConfigJsonChoices(field)
      if ((choices.length === 0 && !/^duration\??$/.test(field.type) && !isConfigDateTime(field.type)) || range.from === range.to) {
        continue
      }
      try {
        const value = JSON.stringify(JSON5.parse(state.doc.sliceString(range.from, range.to)))
        marks.push(Decoration.replace({
          widget: new ConfigJsonChoiceWidget(field.name, value, choices, isConfigJsonMultiChoice(field), readOnly, ranges, errors.get(field.name), /^duration\??$/.test(field.type), durationLabels, dirtyFields.has(field.name), isConfigDateTime(field.type) ? field.type : ''),
        }).range(range.from, range.to))
      } catch {
        continue
      }
    }
    return Decoration.set(marks, true)
  }
  return StateField.define({
    create: decorations,
    update: (_value, transaction) => decorations(transaction.state),
    provide: (field) => EditorView.decorations.from(field),
  })
}
