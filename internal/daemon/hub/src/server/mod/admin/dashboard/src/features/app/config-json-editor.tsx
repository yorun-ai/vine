import { createConfigMapEnumExtension } from './config-map-enum-widget'
import { yamlLanguage } from '@codemirror/lang-yaml'
import { createConfigYamlDocument, normalizeConfigYaml, getConfigYamlErrors, getConfigYamlPropertyRanges } from './config-yaml-document'
import { useLocale } from '@/i18n'
import { createConfigChoiceExtension } from './config-json-choice-widget'
import * as React from 'react'
import CodeMirror, { Decoration, EditorView, GutterMarker, gutterLineClass, RangeSet, hoverTooltip } from '@uiw/react-codemirror'
import { configJsonLanguage, getConfigJsonKeyRanges, getConfigJsonPropertyRanges } from './config-json-language'
import {
  createConfigJsonDocument,
  extractConfigJson,
  normalizeConfigJson,
  getFreeConfigJsonErrors,
  getConfigJsonErrors,
  getConfigJsonDirtyLines,
  getConfigJsonTypeLinks,
} from './config-json-document'
import type { ConfigJsonField } from './config-json-document'
import { createConfigValueProtection } from './config-json-protection'

class ConfigDirtyGutterMarker extends GutterMarker {
  elementClass = 'cm-config-dirty-gutter'
}

const dirtyGutterMarker = new ConfigDirtyGutterMarker()

interface ConfigJsonEditorProps {
  rawValue?: boolean
  format?: 'json5' | 'yaml'
  value: string
  fields: ReadonlyArray<ConfigJsonField>
  typeIndex: ReadonlyMap<string, { skelName: string }>
  onTypeClick: (skelName: string) => void
  dirtyFields: ReadonlySet<string>
  mismatchMessages: ReadonlyMap<string, string>
  lockKeys: boolean
  readOnly: boolean
  onChange: (value: string) => void
  onInvalidChange: (invalid: boolean) => void
}

export function ConfigJsonEditor({
  value,
  format = 'json5',
  rawValue = false,
  fields,
  mismatchMessages,
  dirtyFields,
  typeIndex,
  onTypeClick,
  readOnly,
  lockKeys,
  onChange,
  onInvalidChange,
}: ConfigJsonEditorProps) {
  const { t } = useLocale()
  const isYaml = format === 'yaml'
  const createDocument = React.useCallback((text: string, definitions: ReadonlyArray<ConfigJsonField>, locked: boolean) => {
    if (rawValue) {
      return createConfigJsonDocument(text, definitions, false)
    }
    return (isYaml ? createConfigYamlDocument : createConfigJsonDocument)(text, definitions, locked)
  }, [isYaml, rawValue])
  const freeErrors = isYaml ? getConfigYamlErrors : getFreeConfigJsonErrors
  const fieldErrors = isYaml ? getConfigYamlErrors : getConfigJsonErrors
  const propertyRanges = isYaml ? getConfigYamlPropertyRanges : getConfigJsonPropertyRanges
  const language = isYaml ? yamlLanguage : configJsonLanguage
  const durationLabels = React.useMemo(() => ({
    amount: t('appConfig.durationAmount'),
    unit: t('appConfig.durationUnit'),
    date: t('appConfig.dateValue'),
    time: t('appConfig.timeValue'),
    fraction: t('appConfig.timeFraction'),
    offset: t('appConfig.timeOffset'),
  }), [t])
  const renderedFields = React.useRef(fields)
  const lastEmittedValue = React.useRef(value)
  const [document, setDocument] = React.useState(() =>
    createDocument(value, fields, lockKeys),
  )
  const [draft, setDraft] = React.useState(document.doc)

  React.useEffect(() => {
    if (value === lastEmittedValue.current && document.lockKeys === lockKeys && renderedFields.current === fields) {
      return
    }
    if (freeErrors(draft).length > 0 && value === lastEmittedValue.current) return
    renderedFields.current = fields
    const nextDocument = createDocument(value, fields, lockKeys)
    lastEmittedValue.current = value
    setDocument(nextDocument)
    setDraft(nextDocument.doc)
  }, [value, fields, lockKeys, document.lockKeys, createDocument, draft, freeErrors])

  React.useEffect(() => {
    onInvalidChange((isYaml
      ? getConfigYamlErrors(document.doc, lockKeys ? document.ranges : [])
      : getFreeConfigJsonErrors(document.doc)).length > 0)
    return () => onInvalidChange(false)
  }, [document, isYaml, lockKeys, onInvalidChange])

  const protection = React.useMemo(
    () => createConfigValueProtection(document.ranges),
    [document],
  )
  const { ranges, extensions } = React.useMemo(() => {
    const { ranges } = protection
    return {
      ranges,
      extensions: [
        language,
        EditorView.theme({
          '.cm-tooltip.cm-config-error-tooltip': {
            padding: '6px 8px',
            maxWidth: 'min(32rem, 90vw)',
            fontSize: '13px',
            whiteSpace: 'pre-wrap',
            color: 'var(--destructive)',
            backgroundColor: 'var(--popover)',
            border: '1px solid var(--border)',
            borderRadius: '6px',
          },
          '.cm-config-dirty-line, .cm-gutterElement.cm-config-dirty-gutter': {
            backgroundColor: 'var(--color-amber-100)',
          },
          '.cm-lineNumbers .cm-config-dirty-gutter': {
            boxShadow: 'inset 2px 0 var(--color-amber-400)',
          },
          '.cm-config-choice': {
            font: 'inherit',
            color: 'var(--foreground)',
            backgroundColor: 'var(--background)',
            border: '1px solid var(--input)',
            borderRadius: '4px',
            padding: '0 4px',
            maxWidth: '100%',
            cursor: 'pointer',
          },
          '.cm-config-choice:disabled': {
            opacity: '0.6',
            cursor: 'default',
          },
          '.cm-config-choice:focus-visible': {
            outline: '2px solid var(--ring)',
          },
          '.cm-config-key, .cm-config-key *': {
            color: 'var(--foreground)',
          },
          '.cm-config-comment, .cm-config-comment *': {
            color: '#6b7280',
            fontWeight: 'normal',
            fontStyle: 'normal',
          },
          '.cm-config-type-link, .cm-config-type-link *': {
            color: 'var(--primary)',
            cursor: 'pointer',
            textDecoration: 'none',
            textUnderlineOffset: '2px',
          },
          '.cm-config-type-link:hover, .cm-config-type-link:focus-visible': {
            textDecoration: 'underline',
          },
        }),
        ...(lockKeys ? protection.extensions : [ranges]),
        hoverTooltip((view, position) => {
          const valueRanges = view.state.field(ranges)
          const hoverRanges = isYaml
            ? [...getConfigYamlErrors(view.state.doc.toString(), lockKeys ? valueRanges : []), ...valueRanges]
            : lockKeys ? valueRanges : freeErrors(view.state.doc.toString())
          const range = hoverRanges.find((item) =>
            position >= (item.from === item.to ? item.from - 1 : item.from) && position <= item.to,
          )
          if (!range) {
            return null
          }
          const error = (lockKeys
            ? fieldErrors(view.state.doc.toString(), valueRanges)
            : freeErrors(view.state.doc.toString()))
            .find((item) => item.name === range.name)?.message ?? mismatchMessages.get(range.name)
          if (!error) {
            return null
          }
          return {
            pos: Math.max(0, range.from === range.to ? range.from - 1 : range.from),
            end: range.to,
            above: true,
            create: () => {
              const dom = view.dom.ownerDocument.createElement('div')
              dom.className = 'cm-config-error-tooltip'
              dom.setAttribute('role', 'tooltip')
              dom.textContent = error
              return { dom }
            },
          }
        }, { hoverTime: 1, hideOnChange: true }),
        createConfigChoiceExtension(fields, readOnly, ranges, mismatchMessages, durationLabels, dirtyFields, isYaml),
        ...createConfigMapEnumExtension(fields, ranges, readOnly, isYaml, mismatchMessages, dirtyFields),
        gutterLineClass.compute([ranges], (state) => {
          const doc = state.doc.toString()
          const valueRanges = state.field(ranges)
          const diagnostics = lockKeys ? fieldErrors(doc, valueRanges) : freeErrors(doc)
          if (diagnostics.length > 0) {
            return RangeSet.empty
          }
          const dirtyRanges = lockKeys ? valueRanges : propertyRanges(doc)
          return RangeSet.of(getConfigJsonDirtyLines(doc, dirtyRanges, dirtyFields, mismatchMessages)
            .map((line) => dirtyGutterMarker.range(line)))
        }),
        EditorView.domEventHandlers({
          keydown: (event) => {
            const link = event.target instanceof Element
              ? event.target.closest<HTMLAnchorElement>('a.cm-config-type-link')
              : null
            if (!link || event.key !== 'Enter' || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) {
              return false
            }
            event.preventDefault()
            onTypeClick(link.dataset.skelName!)
            return true
          },
          click: (event) => {
            const link = event.target instanceof Element
              ? event.target.closest<HTMLAnchorElement>('a.cm-config-type-link')
              : null
            if (!link || event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) {
              return false
            }
            event.preventDefault()
            onTypeClick(link.dataset.skelName!)
            return true
          },
        }),
        EditorView.decorations.of((view) => {
          const marks = (isYaml ? [] : getConfigJsonKeyRanges(view.state.doc.toString())).map((range) =>
            Decoration.mark({ class: 'cm-config-key' }).range(range.from, range.to),
          )
          const tree = language.parser.parse(view.state.doc.toString())
          tree.iterate({
            enter: (node) => {
              if (node.name === 'LineComment' || node.name === 'BlockComment' || node.name === 'Comment') {
                marks.push(
                  Decoration.mark({ class: 'cm-config-comment' }).range(node.from, node.to),
                )
              }
            },
          })
          const doc = view.state.doc.toString()

          const valueRanges = view.state.field(ranges)
          for (const link of getConfigJsonTypeLinks(document, valueRanges, typeIndex)) {
            marks.push(Decoration.mark({
              tagName: 'a',
              class: 'cm-config-type-link',
              attributes: {
                href: link.href,
                'data-skel-name': link.skelName,
                tabindex: '0',
              },
            }).range(link.from, link.to))
          }
          const diagnostics = lockKeys ? fieldErrors(doc, valueRanges) : freeErrors(doc)
          const errors = new Map(diagnostics.map((error) => [error.name, error.message]))
          if (isYaml) {
            for (const diagnostic of diagnostics) {
              if (diagnostic.from < diagnostic.to) {
                marks.push(Decoration.mark({
                  class: 'rounded bg-destructive/15 px-0.5 ring-1 ring-destructive/30',
                }).range(diagnostic.from, diagnostic.to))
              }
            }
          }
          for (const range of isYaml && diagnostics.length > 0 ? [] : lockKeys ? valueRanges : diagnostics) {
            if (range.to > 0 && (errors.has(range.name) || mismatchMessages.has(range.name))) {
              marks.push(Decoration.mark({
                class: 'rounded bg-destructive/15 px-0.5 ring-1 ring-destructive/30',
              }).range(
                range.from === range.to ? Math.max(0, range.from - 1) : range.from,
                range.to,
              ))
            }
          }
          if (diagnostics.length === 0) {
            const dirtyRanges = lockKeys ? valueRanges : propertyRanges(doc)
            for (const line of getConfigJsonDirtyLines(doc, dirtyRanges, dirtyFields, mismatchMessages)) {
              marks.push(Decoration.line({ class: 'cm-config-dirty-line' }).range(line))
            }
          }
          return Decoration.set(marks, true)
        }),
      ],
    }
  }, [document, protection, fields, readOnly, lockKeys, mismatchMessages, dirtyFields, durationLabels, typeIndex, onTypeClick, isYaml, language, freeErrors, fieldErrors, propertyRanges])

  return (
    <CodeMirror
      key={document.doc}
      value={draft}
      extensions={extensions}
      readOnly={readOnly}
      onChange={(nextDraft, update) => {
        setDraft(nextDraft)
        let nextValue: string
        try {
          if (isYaml && getConfigYamlErrors(nextDraft, lockKeys ? update.state.field(ranges) : []).length > 0) {
            throw new Error('Invalid YAML configuration')
          }
          nextValue = isYaml ? normalizeConfigYaml(nextDraft) : lockKeys
            ? extractConfigJson(nextDraft, update.state.field(ranges))
            : normalizeConfigJson(nextDraft)
        } catch {
          onInvalidChange(true)
          return
        }
        onInvalidChange(false)
        lastEmittedValue.current = nextValue
        onChange(nextValue)
      }}
      basicSetup={{
        lineNumbers: true,
        foldGutter: true,
        bracketMatching: true,
        closeBrackets: true,
      }}
      minHeight="28rem"
      className="overflow-hidden bg-background text-[13px]"
      theme="light"
    />
  )
}
