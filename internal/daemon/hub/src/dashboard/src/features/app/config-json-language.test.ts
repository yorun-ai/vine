import assert from 'node:assert/strict'
import { test } from 'node:test'
import { jsonLanguage } from '@codemirror/lang-json'
import { getStyleTags } from '@lezer/highlight'
import { configJsonLanguage, getConfigJsonKeyRanges } from './config-json-language.ts'

test('JSON5 values and separators use the same highlighting tags as JSON', () => {
  const source = '{"name":"hello", "port":7099, "enabled":true, "optional":null}'
  function tokenTags(language: typeof jsonLanguage) {
    const result = new Map<string, unknown>()
    language.parser.parse(source).iterate({
      enter: (node) => {
        if (node.node.firstChild === null) {
          result.set(source.slice(node.from, node.to), getStyleTags(node)?.tags)
        }
      },
    })
    return result
  }
  const json = tokenTags(jsonLanguage)
  const json5 = tokenTags(configJsonLanguage)
  for (const text of ['"hello"', '7099', 'true', 'null', ':', ',']) {
    assert.deepEqual(json5.get(text), json.get(text), text)
  }
})

test('key decorations cover quoted, unquoted and nested keys without including values', () => {
  const source = `{name: 'value', 'port': 7099, "nested": {"host": "localhost"}}`
  assert.deepEqual(getConfigJsonKeyRanges(source).map((range) =>
    source.slice(range.from, range.to),
  ), ['name', "'port'", '"nested"', '"host"'])
})
