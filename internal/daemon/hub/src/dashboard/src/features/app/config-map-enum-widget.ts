import JSON5 from 'json5'
import { Decoration, EditorView, StateField } from '@uiw/react-codemirror'
import type { EditorState } from '@uiw/react-codemirror'
import { ConfigJsonChoiceWidget } from './config-json-choice-widget.ts'
import { configMapEntries, configMapEnums, configMapTypes } from './config-map-enum.ts'
import { parseConfigYaml } from './config-yaml-document.ts'
import type { ConfigJsonField, ConfigValueRange } from './config-json-document.ts'

export function createConfigMapEnumExtension(
  fields: ReadonlyArray<ConfigJsonField>,
  parentRanges: StateField<Array<ConfigValueRange>>,
  readOnly: boolean,
  yaml: boolean,
  errors: ReadonlyMap<string, string>,
  dirtyFields: ReadonlySet<string>,
) {
  function entries(state: EditorState) {
    return state.field(parentRanges).flatMap((parent) => {
      const field = fields.find((field) => field.name === parent.name)
      if (!field || !configMapTypes(field.type)) {
        return []
      }
      const enums = configMapEnums(field)
      const items = configMapEntries(state.doc.toString(), parent, yaml)
      return items.flatMap((item) => {
        return (['key', 'value'] as const).flatMap((part) => {
          if (!enums[part].length) {
            return []
          }
          const range = item[`${part}Range`]
          if (range.from >= range.to) {
            return []
          }
          try {
            const value = part === 'key' ? item.key : (yaml ? parseConfigYaml : JSON5.parse)(state.doc.sliceString(range.from, range.to))
            const choices = enums[part]
              .filter((option) => part !== 'key' || option.name === item.key || !items.some((entry) => entry.key === option.name))
              .map((option) => ({ value: JSON.stringify(option.name), label: option.name, description: option.description }))
            if (part === 'value' && configMapTypes(field.type)!.value.endsWith('?')) {
              choices.push({ value: 'null', label: 'null', description: '' })
            }
            return [{ ...range, name: JSON.stringify([parent.name, item.key, part]), parent: parent.name, value: JSON.stringify(value), choices }]
          } catch {
            return []
          }
        })
      })
    })
  }
  const ranges = StateField.define<Array<ConfigValueRange>>({ create: entries, update: (_value, transaction) => entries(transaction.state) })
  const decorations = StateField.define({
    create: render,
    update: (_value, transaction) => render(transaction.state),
    provide: (field) => EditorView.decorations.from(field),
  })
  function render(state: EditorState) {
    return Decoration.set(entries(state).map((entry) => Decoration.replace({
      widget: new ConfigJsonChoiceWidget(entry.name, entry.value, entry.choices, false, readOnly, ranges, errors.get(entry.parent), false, undefined, dirtyFields.has(entry.parent), '', yaml, false),
    }).range(entry.from, entry.to)), true)
  }
  return [ranges, decorations] as const
}
