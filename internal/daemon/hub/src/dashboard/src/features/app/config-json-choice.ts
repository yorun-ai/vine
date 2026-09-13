import { isConfigDateTime } from './config-json-date-time.ts'
import type { ConfigJsonField } from './config-json-document.ts'

export interface ConfigJsonChoice {
  value: string
  label: string
  description?: string
}

export function getConfigJsonChoices(field: ConfigJsonField): Array<ConfigJsonChoice> {
  const type = field.type.replace(/\?$/, '')
  let choices: Array<ConfigJsonChoice>
  if (type === 'duration' || isConfigDateTime(type)) {
    choices = []
  } else if (type === 'bool') {
    choices = [
      { value: 'false', label: 'false' },
      { value: 'true', label: 'true' },
    ]
  } else if ((!type.includes('<') || /^list<[^<>]+>$/.test(type)) && field.enumItems?.length) {
    choices = field.enumItems.map((item) => ({
      value: JSON.stringify(item.name),
      label: item.name,
      description: item.description,
    }))
  } else {
    return []
  }
  if (field.type.endsWith('?')) {
    choices.push({ value: 'null', label: 'null' })
  }
  return choices
}

export function isConfigJsonMultiChoice(field: ConfigJsonField) {
  return /^list<[^<>]+>\??$/.test(field.type) && Boolean(field.enumItems?.length)
}

export function toggleConfigJsonChoice(value: unknown, choice: string) {
  if (choice === 'null') {
    return null
  }
  const item: unknown = JSON.parse(choice)
  const items: Array<unknown> = Array.isArray(value) ? value : []
  return items.includes(item) ? items.filter((entry) => entry !== item) : [...items, item]
}
