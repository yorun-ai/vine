import JSON5 from 'json5'
import { javascriptLanguage } from '@codemirror/lang-javascript'
import { styleTags, tags } from '@lezer/highlight'

export const configJsonLanguage = javascriptLanguage.configure({
  top: 'SingleExpression',
  props: [styleTags({
    PropertyDefinition: tags.propertyName,
    'String!': tags.string,
    ', :': tags.separator,
  })],
})

export function getConfigJsonKeyRanges(doc: string) {
  const ranges: Array<{ from: number; to: number }> = []
  configJsonLanguage.parser.parse(doc).iterate({
    enter: (node) => {
      if (node.name === 'Property') {
        const key = node.node.firstChild
        if (key) {
          ranges.push({ from: key.from, to: key.to })
        }
      }
    },
  })
  return ranges
}

export function getConfigJsonPropertyRanges(doc: string) {
  const ranges: Array<{ name: string; from: number; to: number }> = []
  configJsonLanguage.parser.parse(doc).iterate({
    enter: (node) => {
      if (node.name !== 'Property' || node.node.parent?.parent?.name !== 'SingleExpression') {
        return
      }
      const key = node.node.firstChild
      if (key) {
        try {
          const name = Object.keys(JSON5.parse(`{${doc.slice(key.from, key.to)}: null}`))[0]
          ranges.push({ name, from: node.from, to: node.to })
        } catch {
          return
        }
      }
    },
  })
  return ranges
}
