import * as React from 'react'
import { useNavigate, useRouterState } from '@tanstack/react-router'
import CodeMirror, { Decoration, EditorView } from '@uiw/react-codemirror'
import { json } from '@codemirror/lang-json'
import {
  Braces,
  CalendarIcon,
  Copy,
  Loader2,
  Plus,
  RefreshCw,
  RotateCcw,
  Save,
  Search,
  Trash2,
} from 'lucide-react'
import { toast } from 'sonner'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Calendar } from '@/components/ui/calendar'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  ResizableListHandle,
  useReservedScrollbar,
  useResizableListPanel,
} from '@/components/ui/resizable-list-panel'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { vrpcClient } from '@/config/vrpc-client'
import { useLocale } from '@/i18n'
import { cn } from '@/lib/utils'
import {
  createAppConfigService,
  createSkeletonService,
} from '@/skeled'
import type {
  AppConfigItem,
  AppConfigSchema,
  SkeletonData,
} from '@/skeled'

const appConfigService = createAppConfigService(vrpcClient)
const skeletonService = createSkeletonService(vrpcClient)
const jsonExtensions = [json()]
const APP_CONFIG_LIST_DEFAULT_WIDTH = 352
const APP_CONFIG_LIST_WIDTH_STORAGE_KEY = 'vinehub_app_config_list_width_v2'

interface AppConfigPageProps {
  routeKey?: string
}

interface JsonFieldRange {
  name: string
  from: number
  valueFrom: number
  to: number
}

type TypeDefinitionIndex = Map<string, SkeletonData>
type AppConfigStatus = 'NORMAL' | 'UNUSED' | 'UNCONFIGURED' | 'MISMATCH'

const emptyConfigValue = '{}'

interface ConfigMismatchIssue {
  fieldName?: string
  text: string
}

function shortConfigName(key: string) {
  return key.split('.').at(-1) ?? key
}

function configName(config: AppConfigItem) {
  return config.schema?.name ?? shortConfigName(config.key)
}

function configSkelName(config: AppConfigItem) {
  return config.schema?.skelName ?? config.key
}

function configIsUnused(config: AppConfigItem) {
  return config.status === 'UNUSED' || config.schema === null
}

function configIsUnconfigured(config: AppConfigItem) {
  return config.status === 'UNCONFIGURED'
}

function configIsMismatched(config: AppConfigItem) {
  return config.status === 'MISMATCH'
}

function configStatus(config: AppConfigItem): AppConfigStatus {
  if (configIsUnused(config)) {
    return 'UNUSED'
  }
  if (configIsUnconfigured(config)) {
    return 'UNCONFIGURED'
  }
  if (configIsMismatched(config)) {
    return 'MISMATCH'
  }
  return 'NORMAL'
}

function isValidConfigSkelName(skelName: string) {
  const parts = skelName.split('.')

  if (parts.length < 2) {
    return false
  }

  const configName = parts.at(-1) ?? ''

  return (
    configName.endsWith('Config') &&
    parts.every((part) => /^[A-Za-z_][A-Za-z0-9_]*$/.test(part))
  )
}

function appConfigPath(key: string) {
  return `/app/config/${encodeURIComponent(key)}`
}

function appConfigListItemDomId(key: string) {
  return `app-config-list-item:${encodeURIComponent(key)}`
}

function skeletonConfigPath(skelName: string) {
  return `/skeleton/config/${encodeURIComponent(skelName)}`
}

function skeletonDomainPath(domain: string) {
  return `/skeleton/domain/${encodeURIComponent(domain)}`
}

function splitConfigSkelName(skelName: string) {
  const index = skelName.lastIndexOf('.')
  if (index < 0) {
    return { domainPart: '', restPart: skelName }
  }
  return {
    domainPart: skelName.slice(0, index),
    restPart: skelName.slice(index + 1),
  }
}

function shouldUseBrowserNavigation(
  event: React.MouseEvent<HTMLAnchorElement>,
) {
  return (
    event.defaultPrevented ||
    event.button !== 0 ||
    event.metaKey ||
    event.altKey ||
    event.ctrlKey ||
    event.shiftKey
  )
}

function formatConfigValue(value: string) {
  try {
    return JSON.stringify(JSON.parse(value), null, 2)
  } catch {
    return value
  }
}

function isValidJson(value: string) {
  try {
    JSON.parse(value)
    return true
  } catch {
    return false
  }
}

function parseJsonString(value: string) {
  try {
    return JSON.parse(value) as unknown
  } catch {
    return value
  }
}

function parseConfigObject(value: string) {
  try {
    const parsed = JSON.parse(value) as unknown

    if (
      parsed === null ||
      Array.isArray(parsed) ||
      typeof parsed !== 'object'
    ) {
      return null
    }

    return parsed as Record<string, unknown>
  } catch {
    return null
  }
}

function stringifyConfigObject(value: Record<string, unknown>) {
  return JSON.stringify(value, null, 2)
}

function stringifyFieldJsonValue(value: unknown) {
  return JSON.stringify(value, null, 2)
}

function getErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : 'Request failed'
}

function skipJsonString(value: string, start: number) {
  let position = start + 1

  while (position < value.length) {
    const char = value[position]

    if (char === '\\') {
      position += 2
      continue
    }

    if (char === '"') {
      return position + 1
    }

    position += 1
  }

  return value.length
}

function skipWhitespace(value: string, start: number) {
  let position = start

  while (position < value.length && /\s/.test(value[position])) {
    position += 1
  }

  return position
}

function findJsonValueEnd(value: string, start: number) {
  let position = start
  let depth = 0
  let inString = false
  let escaped = false

  while (position < value.length) {
    const char = value[position]

    if (inString) {
      if (escaped) {
        escaped = false
      } else if (char === '\\') {
        escaped = true
      } else if (char === '"') {
        inString = false
      }

      position += 1
      continue
    }

    if (char === '"') {
      inString = true
    } else if (char === '{' || char === '[') {
      depth += 1
    } else if (char === '}' || char === ']') {
      if (depth === 0) {
        break
      }

      depth -= 1
    } else if (char === ',' && depth === 0) {
      break
    }

    position += 1
  }

  return position
}

function getTopLevelJsonFieldRanges(value: string) {
  const ranges: Array<JsonFieldRange> = []
  let position = skipWhitespace(value, 0)

  if (value[position] !== '{') {
    return ranges
  }

  position += 1

  while (position < value.length) {
    position = skipWhitespace(value, position)

    if (value[position] === '}') {
      break
    }

    if (value[position] !== '"') {
      position += 1
      continue
    }

    const keyFrom = position
    const keyTo = skipJsonString(value, keyFrom)
    const colonPosition = skipWhitespace(value, keyTo)

    if (value[colonPosition] !== ':') {
      position = keyTo
      continue
    }

    let name: string

    try {
      name = JSON.parse(value.slice(keyFrom, keyTo)) as string
    } catch {
      position = keyTo
      continue
    }

    const valueFrom = skipWhitespace(value, colonPosition + 1)
    const valueTo = findJsonValueEnd(value, valueFrom)

    ranges.push({
      name,
      from: keyFrom,
      valueFrom,
      to: valueTo,
    })

    position = valueTo

    if (value[position] === ',') {
      position += 1
    }
  }

  return ranges
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return value !== null && !Array.isArray(value) && typeof value === 'object'
}

function valueToInputText(value: unknown) {
  if (value === null || value === undefined) {
    return ''
  }

  return String(value)
}

function valuesEqual(left: unknown, right: unknown) {
  return JSON.stringify(left) === JSON.stringify(right)
}

function baseConfigType(typeText: string) {
  return typeText.endsWith('?') ? typeText.slice(0, -1) : typeText
}

function defaultConfigFieldValue(
  typeText: string,
  enumItems: Array<{ name: string }>,
): unknown {
  if (typeText.endsWith('?')) {
    return null
  }

  const type = baseConfigType(typeText)

  if (type.startsWith('list<')) {
    return []
  }
  if (type.startsWith('map<')) {
    return {}
  }
  if (enumItems.length > 0) {
    return enumItems[0]?.name ?? ''
  }
  if (type === 'bool') {
    return false
  }
  if (isNumericType(type)) {
    return type === 'decimal' ? '0' : 0
  }
  if (type === 'json') {
    return '{}'
  }

  return ''
}

function defaultConfigObject(schema: AppConfigSchema | null) {
  const ret: Record<string, unknown> = {}

  for (const field of schema?.fields ?? []) {
    ret[field.name] = defaultConfigFieldValue(field.type, field.enumItems)
  }

  return ret
}

function defaultConfigValue(schema: AppConfigSchema | null) {
  return stringifyConfigObject(defaultConfigObject(schema))
}

function completeConfigValue(value: string, schema: AppConfigSchema | null) {
  const current = parseConfigObject(value) ?? {}

  return stringifyConfigObject({
    ...defaultConfigObject(schema),
    ...current,
  })
}

function jsonValueType(value: unknown) {
  if (value === null) {
    return 'null'
  }
  if (Array.isArray(value)) {
    return 'list'
  }
  return typeof value
}

function configTypeLabel(typeText: string) {
  return baseConfigType(typeText)
}

function jsonValueMatchesConfigType(
  value: unknown,
  typeText: string,
  enumItems: Array<{ name: string }>,
): boolean {
  if (value === null) {
    return typeText.endsWith('?')
  }

  const type = baseConfigType(typeText)
  if (type.startsWith('list<')) {
    return Array.isArray(value)
  }
  if (type.startsWith('map<')) {
    return isPlainObject(value)
  }
  if (enumItems.length > 0) {
    return (
      typeof value === 'string' && enumItems.some((item) => item.name === value)
    )
  }
  if (type === 'bool') {
    return typeof value === 'boolean'
  }
  if (isNumericType(type)) {
    return (
      typeof value === 'number' ||
      (type === 'decimal' && typeof value === 'string')
    )
  }
  if (type === 'json') {
    return true
  }
  return typeof value === 'string'
}

function collectConfigMismatchIssues(
  value: string,
  schema: AppConfigSchema | null,
  t: ReturnType<typeof useLocale>['t'],
) {
  if (!schema) {
    return []
  }

  const parsed = parseConfigObject(value)
  if (!parsed) {
    return [{ text: t('appConfig.valueMustBeObject') }]
  }

  const issues: Array<ConfigMismatchIssue> = []
  const fieldsByName = new Map(
    schema.fields.map((field) => [field.name, field]),
  )

  for (const field of schema.fields) {
    if (!Object.prototype.hasOwnProperty.call(parsed, field.name)) {
      issues.push({
        text: t('appConfig.missingField').replace('{field}', field.name),
      })
      continue
    }

    const fieldValue = parsed[field.name]
    if (
      !jsonValueMatchesConfigType(fieldValue, field.type, field.enumItems ?? [])
    ) {
      issues.push({
        fieldName: field.name,
        text: t('appConfig.typeMismatch')
          .replace('{field}', field.name)
          .replace('{expected}', configTypeLabel(field.type))
          .replace('{actual}', jsonValueType(fieldValue)),
      })
    }
  }

  for (const key of Object.keys(parsed)) {
    if (!fieldsByName.has(key)) {
      issues.push({
        fieldName: key,
        text: t('appConfig.unknownField').replace('{field}', key),
      })
    }
  }

  return issues
}

function collectConfigMismatchFieldNames(
  value: string,
  schema: AppConfigSchema | null,
) {
  if (!schema) {
    return new Set<string>()
  }

  const parsed = parseConfigObject(value)
  if (!parsed) {
    return new Set<string>()
  }

  const ret = new Set<string>()
  const fieldsByName = new Map(
    schema.fields.map((field) => [field.name, field]),
  )

  for (const field of schema.fields) {
    if (!Object.prototype.hasOwnProperty.call(parsed, field.name)) {
      continue
    }

    const fieldValue = parsed[field.name]
    if (
      !jsonValueMatchesConfigType(fieldValue, field.type, field.enumItems ?? [])
    ) {
      ret.add(field.name)
    }
  }

  for (const key of Object.keys(parsed)) {
    if (!fieldsByName.has(key)) {
      ret.add(key)
    }
  }

  return ret
}

function getMapKeyType(typeText: string) {
  const matched = /^map<([^,>]+),/.exec(baseConfigType(typeText))
  return matched?.[1]?.trim() ?? ''
}

function getMapValueType(typeText: string) {
  const matched = /^map<[^,>]+,\s*([^>]+)>$/.exec(baseConfigType(typeText))
  return matched?.[1]?.trim() ?? ''
}

function getListValueType(typeText: string) {
  const matched = /^list<([^>]+)>$/.exec(baseConfigType(typeText))
  return matched?.[1]?.trim() ?? ''
}

function isIntegerKey(value: string) {
  return /^-?\d+$/.test(value)
}

function isNumberText(value: string) {
  return value.trim() !== '' && Number.isFinite(Number(value))
}

function isNumericType(typeText: string) {
  return typeText === 'int' || typeText === 'float' || typeText === 'decimal'
}

function buildTypeDefinitionIndex(items: Array<SkeletonData>) {
  const index: TypeDefinitionIndex = new Map()
  for (const item of items) {
    index.set(item.skelName, item)
  }
  return index
}

function ConfigTypeText({
  type,
  typeIndex,
  onTypeClick,
}: {
  type: string
  typeIndex: TypeDefinitionIndex
  onTypeClick: (skelName: string) => void
}) {
  const parts: Array<React.ReactNode> = []
  const pattern = /[A-Za-z_][A-Za-z0-9_.]*/g
  let cursor = 0
  let match: RegExpExecArray | null

  while ((match = pattern.exec(type)) !== null) {
    const token = match[0]
    const start = match.index
    const definition = typeIndex.get(token)

    if (start > cursor) {
      parts.push(type.slice(cursor, start))
    }

    if (definition) {
      parts.push(
        <a
          key={`${token}:${start}`}
          href={`/skeleton/data/${encodeURIComponent(definition.skelName)}`}
          className="font-mono text-primary underline-offset-2 hover:underline"
          onClick={(event) => {
            if (shouldUseBrowserNavigation(event)) {
              return
            }
            event.preventDefault()
            onTypeClick(definition.skelName)
          }}
        >
          {token}
        </a>,
      )
    } else {
      parts.push(token)
    }

    cursor = start + token.length
  }

  if (cursor < type.length) {
    parts.push(type.slice(cursor))
  }

  return <>{parts}</>
}

function parseMapInputValue(value: string, valueType: string) {
  if (isNumericType(valueType)) {
    return Number(value)
  }

  return value
}

function BooleanSelect({
  value,
  onChange,
}: {
  value: boolean
  onChange: (value: boolean) => void
}) {
  const { t } = useLocale()

  return (
    <Select
      value={String(value)}
      onValueChange={(nextValue) => onChange(nextValue === 'true')}
    >
      <SelectTrigger className="w-full">
        <SelectValue placeholder={t('common.select')} />
      </SelectTrigger>
      <SelectContent align="start">
        <SelectItem value="true">true</SelectItem>
        <SelectItem value="false">false</SelectItem>
      </SelectContent>
    </Select>
  )
}

function isTimeScalarType(typeText: string) {
  return (
    typeText === 'timestamp' ||
    typeText === 'duration' ||
    typeText === 'localdate' ||
    typeText === 'localtime' ||
    typeText === 'localdatetime'
  )
}

function splitTimeFraction(value: string) {
  const matched = /^(.*?)(?:\.(\d+))?$/.exec(value)

  return {
    base: matched?.[1] ?? value,
    fraction: matched?.[2] ?? '',
  }
}

function splitTimestampValue(value: string) {
  const matched =
    /^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(?::\d{2})?)(?:\.(\d+))?(Z|[+-]\d{2}:\d{2})?$/.exec(
      value,
    )

  if (!matched) {
    return {
      dateTime: value.replace(/(Z|[+-]\d{2}:\d{2})$/, ''),
      fraction: '',
    }
  }

  return {
    dateTime: matched[1],
    fraction: matched[2] ?? '',
  }
}

function joinTimeFraction(base: string, fraction: string) {
  const normalizedFraction = fraction.replace(/\D/g, '')

  return normalizedFraction ? `${base}.${normalizedFraction}` : base
}

function joinTimestampValue(dateTime: string, fraction: string) {
  return `${joinTimeFraction(dateTime, fraction)}Z`
}

function parseDurationParts(value: string) {
  const unitSeconds = new Map([
    ['h', 3600],
    ['m', 60],
    ['s', 1],
  ])
  const tokenPattern = /(\d+)(h|m|s)/g
  let matched: RegExpExecArray | null
  let totalSeconds = 0
  let consumed = ''

  while ((matched = tokenPattern.exec(value)) !== null) {
    totalSeconds += Number(matched[1]) * (unitSeconds.get(matched[2]) ?? 0)
    consumed += matched[0]
  }

  if (consumed !== value || !Number.isFinite(totalSeconds)) {
    return null
  }

  let rest = Math.floor(totalSeconds)
  const years = Math.floor(rest / 31_536_000)
  rest %= 31_536_000
  const months = Math.floor(rest / 2_592_000)
  rest %= 2_592_000
  const days = Math.floor(rest / 86_400)
  rest %= 86_400
  const hours = Math.floor(rest / 3600)
  rest %= 3600
  const minutes = Math.floor(rest / 60)
  const seconds = rest % 60

  return {
    years,
    months,
    days,
    hours,
    minutes,
    seconds,
  }
}

interface DurationParts {
  years: number
  months: number
  days: number
  hours: number
  minutes: number
  seconds: number
}

function formatDurationParts(parts: DurationParts) {
  const segments: Array<string> = []
  const totalHours =
    parts.years * 365 * 24 +
    parts.months * 30 * 24 +
    parts.days * 24 +
    parts.hours

  if (totalHours !== 0) {
    segments.push(`${totalHours}h`)
  }

  if (parts.minutes !== 0) {
    segments.push(`${parts.minutes}m`)
  }

  if (parts.seconds !== 0) {
    segments.push(`${parts.seconds}s`)
  }

  if (segments.length === 0) {
    segments.push('0s')
  }

  return segments.join('')
}

function parseLocalDate(value: string) {
  const matched = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value)

  if (!matched) {
    return undefined
  }

  return new Date(
    Number(matched[1]),
    Number(matched[2]) - 1,
    Number(matched[3]),
  )
}

function formatLocalDate(date: Date) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')

  return `${year}-${month}-${day}`
}

function splitLocalDateTime(value: string) {
  const [date = '', time = ''] = value.split('T')
  const { base, fraction } = splitTimeFraction(time)

  return {
    date,
    time: base,
    fraction,
  }
}

function DatePickerInput({
  value,
  onChange,
}: {
  value: string
  onChange: (value: string) => void
}) {
  const { t } = useLocale()
  const selectedDate = parseLocalDate(value)

  return (
    <Popover>
      <PopoverTrigger
        render={
          <Button
            type="button"
            variant="outline"
            className={cn(
              'w-full justify-start text-left font-normal',
              !value && 'text-muted-foreground',
            )}
          />
        }
      >
        <CalendarIcon className="size-4" />
        {value || t('common.selectDate')}
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0" align="start">
        <Calendar
          mode="single"
          selected={selectedDate}
          defaultMonth={selectedDate}
          captionLayout="dropdown"
          onSelect={(date) => {
            if (date) {
              onChange(formatLocalDate(date))
            }
          }}
        />
      </PopoverContent>
    </Popover>
  )
}

function TimeScalarInput({
  type,
  value,
  onChange,
}: {
  type: string
  value: string
  onChange: (value: string) => void
}) {
  const { t } = useLocale()

  if (type === 'localdate') {
    return <DatePickerInput value={value} onChange={onChange} />
  }

  if (type === 'localtime') {
    const { base } = splitTimeFraction(value)

    return (
      <Input
        type="time"
        step="1"
        value={base}
        onChange={(event) => onChange(event.target.value)}
      />
    )
  }

  if (type === 'localdatetime') {
    const { date, time } = splitLocalDateTime(value)

    return (
      <div className="grid gap-2 sm:grid-cols-[minmax(10rem,1fr)_10rem]">
        <DatePickerInput
          value={date}
          onChange={(nextDate) => onChange(`${nextDate}T${time}`)}
        />
        <Input
          type="time"
          step="1"
          value={time}
          onChange={(event) => onChange(`${date}T${event.target.value}`)}
        />
      </div>
    )
  }

  if (type === 'timestamp') {
    const { dateTime } = splitTimestampValue(value)
    const { date, time } = splitLocalDateTime(dateTime)

    return (
      <div className="grid gap-2 sm:grid-cols-[minmax(10rem,1fr)_10rem_auto]">
        <DatePickerInput
          value={date}
          onChange={(nextDate) =>
            onChange(joinTimestampValue(`${nextDate}T${time}`, ''))
          }
        />
        <Input
          type="time"
          step="1"
          value={time}
          onChange={(event) =>
            onChange(joinTimestampValue(`${date}T${event.target.value}`, ''))
          }
        />
        <span className="flex h-9 items-center px-1 text-sm font-medium text-muted-foreground">
          Z
        </span>
      </div>
    )
  }

  const durationParts = parseDurationParts(value)

  if (!durationParts) {
    return (
      <Input value={value} onChange={(event) => onChange(event.target.value)} />
    )
  }

  const updateDurationPart = (key: keyof DurationParts, nextValue: number) => {
    onChange(
      formatDurationParts({
        ...durationParts,
        [key]: nextValue,
      }),
    )
  }

  return (
    <div className="grid gap-2 sm:grid-cols-6">
      {[
        ['years', t('common.durationYears')],
        ['months', t('common.durationMonths')],
        ['days', t('common.durationDays')],
        ['hours', t('common.durationHours')],
        ['minutes', t('common.durationMinutes')],
        ['seconds', t('common.durationSeconds')],
      ].map(([key, label]) => (
        <label key={key} className="grid gap-1">
          <span className="text-xs text-muted-foreground">{label}</span>
          <Input
            type="number"
            min="0"
            step="1"
            value={durationParts[key as keyof DurationParts] as number}
            onChange={(event) =>
              updateDurationPart(
                key as keyof DurationParts,
                Number(event.target.value),
              )
            }
          />
        </label>
      ))}
    </div>
  )
}

export function AppConfigPage({ routeKey }: AppConfigPageProps) {
  const { t } = useLocale()
  const navigate = useNavigate()
  const pathnameRouteKey = useRouterState({
    select: (state) => {
      const prefix = '/app/config/'
      const pathname = state.location.pathname

      if (!pathname.startsWith(prefix)) {
        return undefined
      }

      const encodedKey = pathname.slice(prefix.length).replace(/\/$/, '')

      return encodedKey ? decodeURIComponent(encodedKey) : undefined
    },
  })
  const effectiveRouteKey = routeKey ?? pathnameRouteKey
  const routeKeyRef = React.useRef(effectiveRouteKey)
  const scrollHideTimers = React.useRef(new WeakMap<Element, number>())
  const mapKeyInputRefs = React.useRef(new Map<string, HTMLInputElement>())
  const appConfigsRef = React.useRef<Array<AppConfigItem>>([])
  const [appConfigs, setAppConfigs] = React.useState<Array<AppConfigItem>>([])
  const [typeDefinitions, setTypeDefinitions] = React.useState<
    Array<SkeletonData>
  >([])
  const [selectedKey, setSelectedKey] = React.useState<string | null>(
    effectiveRouteKey ?? null,
  )
  const [selectedAppConfig, setSelectedAppConfig] =
    React.useState<AppConfigItem | null>(null)
  const [query, setQuery] = React.useState('')
  const listPanel = useResizableListPanel({
    defaultWidth: APP_CONFIG_LIST_DEFAULT_WIDTH,
    storageKey: APP_CONFIG_LIST_WIDTH_STORAGE_KEY,
  })
  const handleListScroll = useReservedScrollbar()
  const [value, setValue] = React.useState('')
  const [configView, setConfigView] = React.useState('fields')
  const [listLoading, setListLoading] = React.useState(true)
  const [detailLoading, setDetailLoading] = React.useState(false)
  const [saving, setSaving] = React.useState(false)
  const [creating, setCreating] = React.useState(false)
  const [removing, setRemoving] = React.useState(false)
  const [errorMessage, setErrorMessage] = React.useState<string | null>(null)
  const [createDialogOpen, setCreateDialogOpen] = React.useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [createSkelName, setCreateSkelName] = React.useState('')
  const [createValue, setCreateValue] = React.useState(emptyConfigValue)
  const [createSkelNameError, setCreateSkelNameError] = React.useState<
    string | null
  >(null)
  const [createValueError, setCreateValueError] = React.useState<string | null>(
    null,
  )
  const [createMessage, setCreateMessage] = React.useState<string | null>(null)
  const [createMatchedConfig, setCreateMatchedConfig] =
    React.useState<AppConfigItem | null>(null)
  const [mapKeyDrafts, setMapKeyDrafts] = React.useState<
    Record<string, string>
  >({})
  const [mapValueDrafts, setMapValueDrafts] = React.useState<
    Record<string, string>
  >({})
  const [fieldValueDrafts, setFieldValueDrafts] = React.useState<
    Record<string, string>
  >({})
  const [listValueDrafts, setListValueDrafts] = React.useState<
    Record<string, string>
  >({})

  const filteredConfigs = React.useMemo(() => {
    const keyword = query.trim().toLowerCase()

    if (!keyword) {
      return appConfigs
    }

    return appConfigs.filter((config) => {
      return configSkelName(config).toLowerCase().includes(keyword)
    })
  }, [appConfigs, query])
  const createDraftSkelName = createSkelName.trim()
  const createExistingConfig = React.useMemo(() => {
    if (!createDraftSkelName) {
      return null
    }

    return (
      appConfigs.find(
        (config) =>
          configSkelName(config) === createDraftSkelName ||
          config.key === createDraftSkelName,
      ) ?? null
    )
  }, [appConfigs, createDraftSkelName])

  const selectedSchema = React.useMemo(() => {
    return selectedAppConfig?.schema ?? null
  }, [selectedAppConfig])
  const typeIndex = React.useMemo(
    () => buildTypeDefinitionIndex(typeDefinitions),
    [typeDefinitions],
  )

  const valueIsValidJson = React.useMemo(() => isValidJson(value), [value])
  const configObject = React.useMemo(() => parseConfigObject(value), [value])
  const selectedIsUnconfigured = selectedAppConfig
    ? configIsUnconfigured(selectedAppConfig)
    : false
  const selectedIsUnused = selectedAppConfig
    ? configIsUnused(selectedAppConfig)
    : false
  const selectedIsMismatched = selectedAppConfig
    ? configIsMismatched(selectedAppConfig)
    : false
  const selectedSavedValue = selectedIsUnconfigured
    ? defaultConfigValue(selectedSchema)
    : (selectedAppConfig?.value ?? '')
  const savedConfigObject = React.useMemo(
    () => parseConfigObject(selectedSavedValue),
    [selectedSavedValue],
  )
  const mismatchIssues = React.useMemo(
    () => collectConfigMismatchIssues(value, selectedSchema, t),
    [selectedSchema, t, value],
  )
  const mismatchFieldNames = React.useMemo(
    () =>
      selectedIsMismatched
        ? collectConfigMismatchFieldNames(value, selectedSchema)
        : new Set<string>(),
    [selectedIsMismatched, selectedSchema, value],
  )
  const dirtyFields = React.useMemo(() => {
    if (!configObject || !savedConfigObject) {
      return new Set<string>()
    }

    return new Set(
      Object.keys(configObject).filter(
        (key) => !valuesEqual(configObject[key], savedConfigObject[key]),
      ),
    )
  }, [configObject, savedConfigObject])
  const jsonPreviewExtensions = React.useMemo(() => {
    if (dirtyFields.size === 0 && mismatchFieldNames.size === 0) {
      return jsonExtensions
    }

    const dirtyMark = Decoration.mark({
      class: 'rounded bg-amber-100 px-0.5 ring-1 ring-amber-200',
    })
    const mismatchMark = Decoration.mark({
      class: 'rounded bg-destructive/15 px-0.5 ring-1 ring-destructive/30',
    })

    return [
      ...jsonExtensions,
      EditorView.decorations.of((view) => {
        const decorations = getTopLevelJsonFieldRanges(
          view.state.doc.toString(),
        )
          .filter(
            (range) =>
              dirtyFields.has(range.name) || mismatchFieldNames.has(range.name),
          )
          .map((range) => {
            const mark = mismatchFieldNames.has(range.name)
              ? mismatchMark
              : dirtyMark
            return mark.range(range.from, range.to)
          })

        return Decoration.set(decorations, true)
      }),
    ]
  }, [dirtyFields, mismatchFieldNames])

  const hasChanges =
    selectedAppConfig !== null &&
    (selectedIsUnconfigured || value !== selectedSavedValue)
  const hasMapErrors =
    Object.keys(mapKeyDrafts).length > 0 ||
    Object.keys(mapValueDrafts).length > 0
  const hasFieldValueErrors = Object.keys(fieldValueDrafts).length > 0
  const hasListValueErrors = Object.keys(listValueDrafts).length > 0
  const canSave =
    selectedAppConfig !== null &&
    hasChanges &&
    valueIsValidJson &&
    !hasMapErrors &&
    !hasFieldValueErrors &&
    !hasListValueErrors &&
    !saving

  const handleScrollAreaScroll = React.useCallback(
    (event: React.UIEvent<HTMLElement>) => {
      const target = event.currentTarget
      target.dataset.scrolling = 'true'

      const currentTimer = scrollHideTimers.current.get(target)
      if (currentTimer !== undefined) {
        window.clearTimeout(currentTimer)
      }

      const nextTimer = window.setTimeout(() => {
        delete target.dataset.scrolling
        scrollHideTimers.current.delete(target)
      }, 900)

      scrollHideTimers.current.set(target, nextTimer)
    },
    [],
  )

  const navigateToConfig = React.useCallback(
    (key: string, replace = false) => {
      setCreateDialogOpen(false)
      setCreateSkelName('')
      setCreateValue(emptyConfigValue)
      setCreateSkelNameError(null)
      setCreateValueError(null)
      setCreateMessage(null)
      setCreateMatchedConfig(null)
      setSelectedKey(key)
      void navigate({
        to: '/app/config/$configKey',
        params: { configKey: key },
        replace,
      })
    },
    [navigate],
  )
  const navigateToTypeDefinition = React.useCallback(
    (skelName: string) => {
      void navigate({
        to: '/skeleton/data/$skelName',
        params: { skelName },
      })
    },
    [navigate],
  )
  const navigateToConfigDefinition = React.useCallback(
    (skelName: string) => {
      void navigate({
        to: '/skeleton/config/$skelName',
        params: { skelName },
      })
    },
    [navigate],
  )
  const navigateToDomainDefinition = React.useCallback(
    (domain: string) => {
      void navigate({
        to: '/skeleton/domain/$domain',
        params: { domain },
      })
    },
    [navigate],
  )

  const loadTypeDefinitions = React.useCallback(async () => {
    try {
      setTypeDefinitions(await skeletonService.listData(null))
    } catch (error) {
      toast.error(getErrorMessage(error))
    }
  }, [])

  const updateConfigs = React.useCallback((items: Array<AppConfigItem>) => {
    appConfigsRef.current = items
    setAppConfigs(items)
  }, [])

  const loadList = React.useCallback(async () => {
    setListLoading(true)
    setErrorMessage(null)

    try {
      const items = await appConfigService.list(null)
      const pathKey = routeKeyRef.current
      const nextKey =
        pathKey && items.some((item) => item.key === pathKey)
          ? pathKey
          : (items[0]?.key ?? null)

      updateConfigs(items)
      setSelectedKey(nextKey)

      if (nextKey && nextKey !== pathKey) {
        navigateToConfig(nextKey, true)
      }
    } catch (error) {
      setErrorMessage(getErrorMessage(error))
    } finally {
      setListLoading(false)
    }
  }, [navigateToConfig, updateConfigs])

  const loadAppConfig = React.useCallback(async (key: string) => {
    const listedConfig = appConfigsRef.current.find(
      (config) => config.key === key,
    )
    if (!listedConfig) {
      setSelectedAppConfig(null)
      setValue('')
      return
    }

    const listedValue = configIsUnconfigured(listedConfig)
      ? defaultConfigValue(listedConfig.schema)
      : formatConfigValue(listedConfig.value)

    setSelectedAppConfig({ ...listedConfig, value: listedValue })
    setValue(listedValue)
    setDetailLoading(false)
    setErrorMessage(null)

    if (configIsUnconfigured(listedConfig)) {
      return
    }

    try {
      const config = await appConfigService.get({ id: listedConfig.id })
      const formattedValue = formatConfigValue(config.value)
      const nextValue = formattedValue

      setSelectedAppConfig({ ...config, value: nextValue })
      setValue(nextValue)
    } catch (error) {
      setErrorMessage(getErrorMessage(error))
    }
  }, [])

  React.useEffect(() => {
    void loadList()
  }, [loadList])

  React.useEffect(() => {
    void loadTypeDefinitions()
  }, [loadTypeDefinitions])

  React.useEffect(() => {
    routeKeyRef.current = effectiveRouteKey
    setSelectedKey(effectiveRouteKey ?? null)
  }, [effectiveRouteKey])

  React.useEffect(() => {
    setMapKeyDrafts({})
    setMapValueDrafts({})
    setFieldValueDrafts({})
    setListValueDrafts({})
  }, [selectedKey])

  React.useEffect(() => {
    if (selectedIsUnused || selectedIsMismatched) {
      setConfigView('json')
      return
    }
    if (selectedAppConfig) {
      setConfigView('fields')
    }
  }, [selectedAppConfig, selectedIsMismatched, selectedIsUnused])

  React.useEffect(() => {
    if (!selectedKey) {
      return
    }
    window.requestAnimationFrame(() => {
      document
        .getElementById(appConfigListItemDomId(selectedKey))
        ?.scrollIntoView({
          block: 'nearest',
          inline: 'nearest',
        })
    })
  }, [filteredConfigs, selectedKey])

  React.useEffect(() => {
    for (const [draftKey, draftValue] of Object.entries(mapKeyDrafts)) {
      if (draftValue !== '') {
        continue
      }

      const input = mapKeyInputRefs.current.get(draftKey)
      if (!input) {
        continue
      }

      input.focus()
      input.select()
    }
  }, [mapKeyDrafts])

  React.useEffect(() => {
    if (!selectedKey) {
      setSelectedAppConfig(null)
      setValue('')
      return
    }

    void loadAppConfig(selectedKey)
  }, [appConfigs, loadAppConfig, selectedKey])

  async function handleSave() {
    if (!selectedAppConfig || !canSave) {
      return
    }

    setSaving(true)
    setErrorMessage(null)

    try {
      const normalizedValue = selectedIsUnconfigured
        ? completeConfigValue(value, selectedSchema)
        : formatConfigValue(value)
      const updated = selectedIsUnconfigured
        ? await appConfigService.create({
            creation: {
              skelName: selectedAppConfig.key,
              value: normalizedValue,
            },
          })
        : await appConfigService.update({
            id: selectedAppConfig.id,
            update: {
              value:
                normalizedValue === selectedAppConfig.value
                  ? null
                  : normalizedValue,
            },
          })

      const formattedValue = formatConfigValue(updated.value)
      const formattedUpdated = { ...updated, value: formattedValue }

      setSelectedAppConfig(formattedUpdated)
      setValue(formattedValue)
      updateConfigs(
        appConfigsRef.current.map((config) =>
          config.key === updated.key ? formattedUpdated : config,
        ),
      )
      toast.success(t('appConfig.saved'))
    } catch (error) {
      setErrorMessage(getErrorMessage(error))
    } finally {
      setSaving(false)
    }
  }

  async function handleCreateConfig() {
    const skelName = createSkelName.trim()

    setCreateSkelNameError(null)
    setCreateValueError(null)
    setCreateMessage(null)
    setCreateMatchedConfig(null)

    if (!skelName) {
      setCreateSkelNameError(t('appConfig.skelNameRequired'))
      return
    }
    if (!isValidConfigSkelName(skelName)) {
      setCreateSkelNameError(
        t('appConfig.skelNameInvalid'),
      )
      return
    }
    const matchedConfig = appConfigsRef.current.find(
      (config) =>
        configSkelName(config) === skelName || config.key === skelName,
    )

    if (matchedConfig && !configIsUnused(matchedConfig)) {
      setCreateMatchedConfig(matchedConfig)
      setCreateMessage(t('appConfig.exists'))
      return
    }
    if (matchedConfig) {
      setCreateMatchedConfig(matchedConfig)
      setCreateMessage(t('appConfig.exists'))
      return
    }
    if (!isValidJson(createValue)) {
      setCreateValueError(t('appConfig.valueInvalidJson'))
      return
    }

    setCreating(true)
    setErrorMessage(null)

    try {
      const normalizedValue = formatConfigValue(createValue)
      const created = await appConfigService.create({
        creation: {
          skelName,
          value: normalizedValue,
        },
      })

      const formattedValue = formatConfigValue(created.value)
      const formattedCreated = { ...created, value: formattedValue }

      setSelectedAppConfig(formattedCreated)
      setValue(formattedValue)
      setCreateDialogOpen(false)
      setCreateSkelName('')
      setCreateValue(emptyConfigValue)
      setCreateSkelNameError(null)
      setCreateValueError(null)
      setCreateMessage(null)
      setCreateMatchedConfig(null)
      navigateToConfig(created.key)
      void loadList()
      toast.success(t('appConfig.saved'))
    } catch (error) {
      setCreateMessage(getErrorMessage(error))
    } finally {
      setCreating(false)
    }
  }

  async function handleRemoveConfig() {
    if (!selectedAppConfig || !selectedIsUnused) {
      return
    }

    setRemoving(true)
    setErrorMessage(null)

    try {
      await appConfigService.remove({ id: selectedAppConfig.id })
      const nextConfigs = appConfigsRef.current.filter(
        (config) => config.id !== selectedAppConfig.id,
      )
      const nextKey = nextConfigs[0]?.key ?? null

      updateConfigs(nextConfigs)
      setDeleteDialogOpen(false)
      setSelectedAppConfig(null)
      setValue('')

      if (nextKey) {
        navigateToConfig(nextKey, true)
      } else {
        setSelectedKey(null)
        void navigate({ to: '/app/config', replace: true })
      }
      toast.success(t('appConfig.deleted'))
    } catch (error) {
      setErrorMessage(getErrorMessage(error))
    } finally {
      setRemoving(false)
    }
  }

  function handleResetChanges() {
    if (!selectedAppConfig || !hasChanges) {
      return
    }

    setValue(selectedSavedValue)
    setMapKeyDrafts({})
    setMapValueDrafts({})
    setFieldValueDrafts({})
    setListValueDrafts({})
  }

  async function handleCopyConfigJson() {
    try {
      await navigator.clipboard.writeText(value)
      toast.success(t('appConfig.jsonCopied'))
    } catch (error) {
      toast.error(getErrorMessage(error))
    }
  }

  function updateConfigField(fieldName: string, nextValue: unknown) {
    setValue((current) => {
      const currentObject = parseConfigObject(current)

      if (!currentObject) {
        return current
      }

      return stringifyConfigObject({
        ...currentObject,
        [fieldName]: nextValue,
      })
    })
  }

  function handleEnumValueChange(fieldName: string, enumValue: string) {
    updateConfigField(fieldName, enumValue)
  }

  return (
    <section className="flex h-[calc(100dvh-3.5rem)] flex-col overflow-hidden bg-white">
      <div
        className="grid min-h-0 flex-1 grid-cols-1 lg:grid-cols-[var(--list-panel-width)_minmax(0,1fr)]"
        style={listPanel.gridStyle}
      >
        <aside className="relative flex min-h-0 flex-col border-b border-border/70 lg:border-r lg:border-b-0">
          <div className="border-b border-border/70 p-4">
            <div className="relative">
              <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                placeholder={t('common.searchSkelName')}
                className="pl-8"
              />
            </div>
            <div className="mt-3 flex items-center justify-between text-xs text-muted-foreground">
              <span>
                {t('appConfig.itemCount').replace(
                  '{count}',
                  String(filteredConfigs.length),
                )}
              </span>
              <div className="flex items-center gap-2">
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => void loadList()}
                  disabled={listLoading}
                  className="size-7"
                  title={t('action.refreshList')}
                >
                  {listLoading ? (
                    <Loader2 className="size-3.5 animate-spin" />
                  ) : (
                    <RefreshCw className="size-3.5" />
                  )}
                </Button>
                <Button
                  size="sm"
                  onClick={() => {
                    setCreateDialogOpen(true)
                    setCreateSkelName('')
                    setCreateValue(emptyConfigValue)
                    setCreateSkelNameError(null)
                    setCreateValueError(null)
                    setCreateMessage(null)
                    setCreateMatchedConfig(null)
                    setSelectedKey(null)
                    setSelectedAppConfig(null)
                    setValue('')
                    void navigate({ to: '/app/config' })
                  }}
                  className="h-7 gap-1.5 px-2.5"
                  title={t('action.addConfig')}
                >
                  <Plus className="size-3.5" />
                  {t('action.create')}
                </Button>
              </div>
            </div>
          </div>

          <div
            className="scrollbar-reserved min-h-0 flex-1 overflow-auto py-2 pr-1 pl-2"
            onScroll={handleListScroll}
          >
            {listLoading ? (
              <div className="space-y-2">
                {Array.from({ length: 6 }).map((_, index) => (
                  <Skeleton key={index} className="h-16 w-full" />
                ))}
              </div>
            ) : filteredConfigs.length === 0 && !createDialogOpen ? (
              <div className="flex h-40 items-center justify-center text-sm text-muted-foreground">
                {t('appConfig.empty')}
              </div>
            ) : (
              <div className="space-y-1">
                {createDialogOpen ? (
                  <div className="relative flex w-full flex-col gap-1 rounded-lg border border-dashed border-primary/40 bg-primary/[0.04] px-3 py-2.5 pr-20 text-left">
                    <Badge
                      variant="outline"
                      className="absolute top-2.5 right-3 border-primary/30 bg-background text-primary"
                    >
                      {t('action.create')}
                    </Badge>
                    <span className="truncate text-sm font-medium text-primary">
                      {t('action.createConfig')}
                    </span>
                    <span className="truncate text-xs text-muted-foreground">
                      {createDraftSkelName || t('appConfig.waitingSkelName')}
                    </span>
                  </div>
                ) : null}
                {filteredConfigs.map((config) => {
                  const isSelected = config.key === selectedKey
                  const status = configStatus(config)

                  return (
                    <a
                      key={config.key}
                      id={appConfigListItemDomId(config.key)}
                      href={appConfigPath(config.key)}
                      onClick={(event) => {
                        if (shouldUseBrowserNavigation(event)) {
                          return
                        }
                        event.preventDefault()
                        navigateToConfig(config.key)
                      }}
                      className={cn(
                        'relative flex w-full flex-col gap-1 rounded-lg border px-3 py-2.5 text-left transition-colors',
                        status !== 'NORMAL' && 'pr-24',
                        isSelected
                          ? 'border-primary/30 bg-primary/[0.06]'
                          : 'border-transparent hover:bg-primary/[0.05]',
                      )}
                    >
                      {status === 'UNUSED' ? (
                        <Badge
                          variant="outline"
                          className="absolute top-2.5 right-3 border-amber-400 bg-amber-50 text-amber-700"
                        >
                          {t('status.unused')}
                        </Badge>
                      ) : status === 'UNCONFIGURED' ? (
                        <Badge
                          variant="outline"
                          className="absolute top-2.5 right-3 border-sky-300 bg-sky-50 text-sky-700"
                        >
                          {t('status.unconfigured')}
                        </Badge>
                      ) : status === 'MISMATCH' ? (
                        <Badge
                          variant="outline"
                          className="absolute top-2.5 right-3 border-destructive/40 bg-destructive/5 text-destructive"
                        >
                          {t('status.mismatch')}
                        </Badge>
                      ) : null}
                      <span
                        className={cn(
                          'flex min-w-0 items-center gap-2 text-sm font-medium',
                          isSelected ? 'text-primary' : 'text-foreground',
                        )}
                      >
                        <span className="truncate">{configName(config)}</span>
                        {status !== 'UNUSED' && config.lifecycle ? (
                          <Badge variant="outline" className="shrink-0">
                            {config.lifecycle}
                          </Badge>
                        ) : null}
                      </span>
                      <span className="truncate text-xs text-muted-foreground">
                        {configSkelName(config)}
                      </span>
                    </a>
                  )
                })}
              </div>
            )}
          </div>
          <ResizableListHandle
            defaultWidth={APP_CONFIG_LIST_DEFAULT_WIDTH}
            label={t('appConfig.resizeList')}
            panel={listPanel}
          />
        </aside>

        <main className="min-h-0 overflow-hidden">
          {errorMessage ? (
            <div className="px-6 pt-4">
              <Alert variant="destructive">
                <AlertTitle>{t('appConfig.requestFailed')}</AlertTitle>
                <AlertDescription>{errorMessage}</AlertDescription>
              </Alert>
            </div>
          ) : null}

          {createDialogOpen ? (
            <div className="flex h-full min-h-0 flex-col">
              <div className="border-b border-border/70 px-6 py-4">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <Plus className="size-4 text-primary" />
                      <h2 className="truncate text-base font-semibold text-foreground">
                        {t('action.createConfig')}
                      </h2>
                      <Badge variant="outline">{t('action.draft')}</Badge>
                    </div>
                    <p className="mt-2 text-sm text-muted-foreground">
                      {t('appConfig.createDescription')}
                    </p>
                  </div>
                  <div className="flex flex-wrap items-start justify-end gap-2">
                    <Button
                      type="button"
                      variant="outline"
                      onClick={() => {
                        setCreateDialogOpen(false)
                        setCreateSkelName('')
                        setCreateValue(emptyConfigValue)
                        setCreateSkelNameError(null)
                        setCreateValueError(null)
                        setCreateMessage(null)
                        setCreateMatchedConfig(null)
                      }}
                    >
                      {t('action.cancel')}
                    </Button>
                    <Button
                      type="button"
                      onClick={() => void handleCreateConfig()}
                      disabled={creating || createExistingConfig !== null}
                    >
                      {creating ? (
                        <Loader2 className="size-4 animate-spin" />
                      ) : (
                        <Save className="size-4" />
                      )}
                      {t('action.save')}
                    </Button>
                  </div>
                </div>
              </div>

              <div className="min-h-0 flex-1 overflow-y-auto p-6">
                <div className="grid max-w-3xl gap-4">
                  <label className="grid gap-1.5">
                    <span className="text-sm font-medium text-foreground">
                      skelName
                    </span>
                    <Input
                      value={createSkelName}
                      onChange={(event) => {
                        setCreateSkelName(event.target.value)
                        setCreateSkelNameError(null)
                        setCreateValueError(null)
                        setCreateMessage(null)
                        setCreateMatchedConfig(null)
                      }}
                      aria-invalid={Boolean(createSkelNameError)}
                      className={cn(
                        createSkelNameError &&
                          'border-destructive focus-visible:border-destructive focus-visible:ring-destructive/20',
                      )}
                      placeholder={t('appConfig.exampleSkelName')}
                    />
                    {createSkelNameError ? (
                      <span className="text-xs text-destructive">
                        {createSkelNameError}
                      </span>
                    ) : null}
                  </label>
                  {createExistingConfig ? (
                    <div className="grid gap-2">
                      <div className="rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
                        {t('appConfig.exists')}
                      </div>
                      <div>
                        <Button
                          type="button"
                          size="default"
                          onClick={() => {
                            setCreateDialogOpen(false)
                            navigateToConfig(createExistingConfig.key)
                          }}
                        >
                          {t('action.jump')}
                        </Button>
                      </div>
                    </div>
                  ) : (
                    <label className="grid min-h-0 gap-1.5">
                      <span className="text-sm font-medium text-foreground">
                        JSON
                      </span>
                      <CodeMirror
                        value={createValue}
                        extensions={jsonExtensions}
                        onChange={(nextValue) => {
                          setCreateValue(nextValue)
                          setCreateValueError(null)
                        }}
                        basicSetup={{
                          autocompletion: true,
                          bracketMatching: true,
                          closeBrackets: true,
                          foldGutter: true,
                          highlightActiveLine: true,
                          highlightActiveLineGutter: true,
                          lineNumbers: true,
                        }}
                        minHeight="28rem"
                        className={cn(
                          'overflow-hidden rounded-lg border bg-background text-[13px]',
                          isValidJson(createValue)
                            ? 'border-input'
                            : 'border-destructive',
                        )}
                        theme="light"
                      />
                      {createValueError ? (
                        <span className="text-xs text-destructive">
                          {createValueError}
                        </span>
                      ) : null}
                    </label>
                  )}
                  {createMessage ? (
                    <div className="flex items-center justify-between gap-3 rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
                      <span>{createMessage}</span>
                      {createMatchedConfig ? (
                        <Button
                          type="button"
                          size="default"
                          onClick={() => {
                            setCreateDialogOpen(false)
                            navigateToConfig(createMatchedConfig.key)
                          }}
                        >
                          {t('action.jump')}
                        </Button>
                      ) : null}
                    </div>
                  ) : null}
                </div>
              </div>
            </div>
          ) : !selectedKey && !listLoading ? (
            <div className="flex h-full min-h-[24rem] items-center justify-center text-sm text-muted-foreground">
              {t('appConfig.selectOne')}
            </div>
          ) : detailLoading ? (
            <div className="space-y-4 p-6">
              <Skeleton className="h-8 w-56" />
              <Skeleton className="h-8 w-full" />
              <Skeleton className="h-[28rem] w-full" />
            </div>
          ) : selectedAppConfig ? (
            <div className="flex h-full min-h-0 flex-col">
              <div className="border-b border-border/70 px-6 py-4">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <Braces className="size-4 text-primary" />
                      <h2 className="truncate text-base font-semibold text-foreground">
                        {configName(selectedAppConfig)}
                      </h2>
                      {selectedIsUnused ? (
                        <Badge
                          variant="outline"
                          className="border-amber-400 bg-amber-50 text-amber-700"
                        >
                          {t('status.unused')}
                        </Badge>
                      ) : selectedIsMismatched ? (
                        <>
                          <Badge
                            variant="outline"
                            className="border-destructive/40 bg-destructive/5 text-destructive"
                          >
                            {t('status.mismatch')}
                          </Badge>
                          <Badge variant="outline">
                            {selectedAppConfig.lifecycle}
                          </Badge>
                        </>
                      ) : selectedIsUnconfigured ? (
                        <>
                          <Badge
                            variant="outline"
                            className="border-sky-300 bg-sky-50 text-sky-700"
                          >
                            {t('status.unconfigured')}
                          </Badge>
                          <Badge variant="outline">
                            {selectedAppConfig.lifecycle}
                          </Badge>
                        </>
                      ) : (
                        <Badge variant="outline">
                          {selectedAppConfig.lifecycle}
                        </Badge>
                      )}
                    </div>
                    <p className="mt-2 truncate font-mono text-xs text-muted-foreground">
                      {(() => {
                        const skelName = configSkelName(selectedAppConfig)
                        const { domainPart, restPart } =
                          splitConfigSkelName(skelName)
                        return (
                          <>
                            {domainPart ? (
                              <a
                                href={skeletonDomainPath(domainPart)}
                                className="font-mono text-primary underline-offset-2 hover:underline"
                                onClick={(event) => {
                                  if (shouldUseBrowserNavigation(event)) {
                                    return
                                  }
                                  event.preventDefault()
                                  navigateToDomainDefinition(domainPart)
                                }}
                              >
                                {domainPart}
                              </a>
                            ) : null}
                            {domainPart ? '.' : null}
                            <a
                              href={skeletonConfigPath(skelName)}
                              className="font-mono text-primary underline-offset-2 hover:underline"
                              onClick={(event) => {
                                if (shouldUseBrowserNavigation(event)) {
                                  return
                                }
                                event.preventDefault()
                                navigateToConfigDefinition(skelName)
                              }}
                            >
                              {restPart}
                            </a>
                          </>
                        )
                      })()}
                    </p>
                    {selectedSchema?.description ? (
                      <p className="mt-2 min-w-0 truncate text-sm leading-6 text-muted-foreground">
                        {selectedSchema.description}
                      </p>
                    ) : null}
                  </div>

                  <div className="flex flex-wrap items-start justify-end gap-2">
                    {selectedIsUnused ? (
                      <Button
                        type="button"
                        variant="outline"
                        onClick={() => setDeleteDialogOpen(true)}
                        disabled={removing}
                      >
                        <Trash2 className="size-4" />
                        {t('action.delete')}
                      </Button>
                    ) : null}
                    <Button
                      type="button"
                      variant="outline"
                      onClick={handleResetChanges}
                      disabled={!hasChanges || saving}
                    >
                      <RotateCcw className="size-4" />
                      {t('action.undo')}
                    </Button>
                    <Button
                      onClick={() => void handleSave()}
                      disabled={!canSave}
                    >
                      {saving ? (
                        <Loader2 className="size-4 animate-spin" />
                      ) : (
                        <Save className="size-4" />
                      )}
                      {t('action.save')}
                    </Button>
                    <Dialog
                      open={deleteDialogOpen}
                      onOpenChange={setDeleteDialogOpen}
                    >
                      <DialogContent>
                        <DialogHeader>
                          <DialogTitle>
                            {t('appConfig.deleteUnusedTitle')}
                          </DialogTitle>
                          <DialogDescription>
                            {t('appConfig.deleteUnusedDescription')}
                          </DialogDescription>
                        </DialogHeader>
                        <div className="rounded-md border bg-muted/40 px-3 py-2 font-mono text-sm">
                          {selectedAppConfig.key}
                        </div>
                        <DialogFooter>
                          <DialogClose render={<Button variant="outline" />}>
                            {t('action.cancel')}
                          </DialogClose>
                          <Button
                            type="button"
                            variant="destructive"
                            onClick={() => void handleRemoveConfig()}
                            disabled={removing}
                          >
                            {removing ? (
                              <Loader2 className="size-4 animate-spin" />
                            ) : (
                              <Trash2 className="size-4" />
                            )}
                            {t('action.delete')}
                          </Button>
                        </DialogFooter>
                      </DialogContent>
                    </Dialog>
                  </div>
                </div>
              </div>

              <div className="min-h-0 flex-1 overflow-hidden p-6">
                <div className="flex h-full min-w-0 flex-col gap-4">
                  {selectedIsMismatched ? (
                    <Alert variant="destructive">
                      <AlertTitle>{t('appConfig.mismatchTitle')}</AlertTitle>
                      <AlertDescription>
                        <div className="grid gap-2">
                          <span>{t('appConfig.fieldsUnavailable')}</span>
                          {mismatchIssues.length > 0 ? (
                            <ul className="list-disc space-y-1 pl-4">
                              {mismatchIssues.slice(0, 8).map((issue) => (
                                <li key={issue.text}>{issue.text}</li>
                              ))}
                              {mismatchIssues.length > 8 ? (
                                <li>
                                  {t('appConfig.moreIssues').replace(
                                    '{count}',
                                    String(mismatchIssues.length - 8),
                                  )}
                                </li>
                              ) : null}
                            </ul>
                          ) : null}
                        </div>
                      </AlertDescription>
                    </Alert>
                  ) : null}
                  <Tabs
                    value={configView}
                    onValueChange={(nextView) => {
                      if (
                        (selectedIsUnused || selectedIsMismatched) &&
                        nextView === 'fields'
                      ) {
                        return
                      }
                      setConfigView(nextView)
                    }}
                    className="min-h-0 flex-1"
                  >
                    <div className="flex items-center justify-between gap-3">
                      <div className="flex min-w-0 items-center gap-2">
                        <h3 className="text-sm font-semibold text-foreground">
                          {t('appConfig.fieldsTitle')}
                        </h3>
                        {configView === 'json' ? (
                          <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            className="h-7 px-2 text-xs"
                            onClick={() => void handleCopyConfigJson()}
                          >
                            <Copy />
                            Copy
                          </Button>
                        ) : null}
                      </div>
                      <TabsList>
                        {!selectedIsUnused && !selectedIsMismatched ? (
                          <TabsTrigger value="fields">
                            {t('common.fields')}
                          </TabsTrigger>
                        ) : null}
                        <TabsTrigger value="json">JSON</TabsTrigger>
                      </TabsList>
                    </div>

                    {!selectedIsUnused && !selectedIsMismatched ? (
                      <TabsContent
                        value="fields"
                        className="scrollbar-reserved min-h-0 overflow-y-auto pr-1"
                        onScroll={handleScrollAreaScroll}
                      >
                        {selectedSchema && selectedSchema.fields.length > 0 ? (
                          <div className="grid gap-1">
                            {selectedSchema.fields.map((field) => {
                              const enumItems = field.enumItems ?? []
                              const fieldValue = configObject?.[field.name]
                              const isDirty = dirtyFields.has(field.name)
                              const listValueType = getListValueType(field.type)
                              const isListNumericValue =
                                field.type.startsWith('list<') &&
                                isNumericType(listValueType)
                              const isListBooleanValue =
                                field.type.startsWith('list<') &&
                                listValueType === 'bool'
                              const isMapEnumKey =
                                enumItems.length > 0 &&
                                field.type.startsWith('map<')
                              const mapKeyType = getMapKeyType(field.type)
                              const mapValueType = getMapValueType(field.type)
                              const isMapIntKey =
                                field.type.startsWith('map<') &&
                                mapKeyType === 'int'
                              const isMapNumericValue =
                                field.type.startsWith('map<') &&
                                isNumericType(mapValueType)
                              const isMapBooleanValue =
                                field.type.startsWith('map<') &&
                                mapValueType === 'bool'

                              return (
                                <div
                                  key={field.name}
                                  data-dirty={isDirty}
                                  onPointerDown={(event) =>
                                    event.stopPropagation()
                                  }
                                  className={cn(
                                    'grid gap-3 border-b border-l-2 border-b-border/50 border-l-transparent px-3 py-3 text-left last:border-b-0 md:grid-cols-[minmax(12rem,17rem)_minmax(0,1fr)] md:items-start',
                                    isDirty &&
                                      'rounded-md border-l-amber-400 bg-amber-100/70',
                                  )}
                                >
                                  <div className="min-w-0 text-left">
                                    <div className="truncate text-sm font-medium text-foreground">
                                      {field.name}
                                    </div>
                                    <div className="mt-1 flex min-w-0 items-center gap-2 text-xs leading-5 text-muted-foreground">
                                      {field.type ? (
                                        <span className="inline-flex shrink-0 rounded bg-muted px-1.5 py-0.5 font-mono text-[11px] leading-4 text-muted-foreground">
                                          <ConfigTypeText
                                            type={field.type}
                                            typeIndex={typeIndex}
                                            onTypeClick={
                                              navigateToTypeDefinition
                                            }
                                          />
                                        </span>
                                      ) : null}
                                      <span className="min-w-0 truncate">
                                        {field.description ??
                                          t('appConfig.noFieldDescription')}
                                      </span>
                                    </div>
                                  </div>

                                  {enumItems.length > 0 &&
                                  typeof fieldValue === 'string' ? (
                                    <Select
                                      value={fieldValue}
                                      onValueChange={(nextValue) => {
                                        if (nextValue) {
                                          handleEnumValueChange(
                                            field.name,
                                            nextValue,
                                          )
                                        }
                                      }}
                                    >
                                      <SelectTrigger className="w-full">
                                        <SelectValue
                                          placeholder={t('common.select')}
                                        />
                                      </SelectTrigger>
                                      <SelectContent align="start">
                                        {enumItems.map((item) => (
                                          <SelectItem
                                            key={item.name}
                                            value={item.name}
                                          >
                                            <span className="grid min-w-0 grid-cols-[5.5rem_minmax(0,1fr)] items-center gap-2">
                                              <span className="truncate">
                                                {item.name}
                                              </span>
                                              <span className="truncate text-xs text-muted-foreground">
                                                {item.description ?? ''}
                                              </span>
                                            </span>
                                          </SelectItem>
                                        ))}
                                      </SelectContent>
                                    </Select>
                                  ) : typeof fieldValue === 'boolean' ? (
                                    <BooleanSelect
                                      value={fieldValue}
                                      onChange={(checked) =>
                                        updateConfigField(field.name, checked)
                                      }
                                    />
                                  ) : typeof fieldValue === 'number' ||
                                    (field.type === 'decimal' &&
                                      typeof fieldValue === 'string') ? (
                                    <div className="grid gap-1">
                                      <Input
                                        type="number"
                                        inputMode="decimal"
                                        value={
                                          fieldValueDrafts[field.name] ??
                                          valueToInputText(fieldValue)
                                        }
                                        aria-invalid={Boolean(
                                          fieldValueDrafts[field.name],
                                        )}
                                        className={cn(
                                          fieldValueDrafts[field.name] &&
                                            'border-destructive focus-visible:border-destructive focus-visible:ring-destructive/20',
                                        )}
                                        onChange={(event) => {
                                          const nextValue = event.target.value

                                          if (!isNumberText(nextValue)) {
                                            setFieldValueDrafts((current) => ({
                                              ...current,
                                              [field.name]: nextValue,
                                            }))
                                            return
                                          }

                                          setFieldValueDrafts((current) => {
                                            const next = { ...current }
                                            delete next[field.name]
                                            return next
                                          })
                                          updateConfigField(
                                            field.name,
                                            field.type === 'decimal'
                                              ? nextValue
                                              : Number(nextValue),
                                          )
                                        }}
                                      />
                                      {fieldValueDrafts[field.name] ? (
                                        <div className="text-xs text-destructive">
                                          {t('appConfig.valueMustBeNumber')}
                                        </div>
                                      ) : null}
                                    </div>
                                  ) : typeof fieldValue === 'string' &&
                                    isTimeScalarType(field.type) ? (
                                    <TimeScalarInput
                                      type={field.type}
                                      value={fieldValue}
                                      onChange={(nextValue) =>
                                        updateConfigField(field.name, nextValue)
                                      }
                                    />
                                  ) : typeof fieldValue === 'string' &&
                                    field.type === 'json' ? (
                                    (() => {
                                      const formattedFieldJson =
                                        stringifyFieldJsonValue(
                                          parseJsonString(fieldValue),
                                        )

                                      return (
                                        <CodeMirror
                                          value={formattedFieldJson}
                                          extensions={jsonExtensions}
                                          onChange={(nextValue) => {
                                            if (isValidJson(nextValue)) {
                                              updateConfigField(
                                                field.name,
                                                JSON.stringify(
                                                  JSON.parse(
                                                    nextValue,
                                                  ) as unknown,
                                                ),
                                              )
                                            }
                                          }}
                                          basicSetup={{
                                            autocompletion: true,
                                            bracketMatching: true,
                                            closeBrackets: true,
                                            foldGutter: true,
                                            highlightActiveLine: true,
                                            highlightActiveLineGutter: true,
                                            lineNumbers: true,
                                          }}
                                          height={`${
                                            Math.max(
                                              4,
                                              formattedFieldJson.split('\n')
                                                .length,
                                            ) * 1.5
                                          }rem`}
                                          className="overflow-hidden rounded-md border border-input bg-background text-[13px]"
                                          theme="light"
                                        />
                                      )
                                    })()
                                  ) : typeof fieldValue === 'string' ? (
                                    <Input
                                      value={fieldValue}
                                      onChange={(event) =>
                                        updateConfigField(
                                          field.name,
                                          event.target.value,
                                        )
                                      }
                                    />
                                  ) : Array.isArray(fieldValue) ? (
                                    <div
                                      className="scrollbar-reserved grid max-h-48 gap-2 overflow-y-auto pr-1"
                                      onScroll={handleScrollAreaScroll}
                                    >
                                      {fieldValue.map((item, index) => {
                                        const listDraftKey = `${field.name}:${index}`
                                        const shownListValue =
                                          listValueDrafts[listDraftKey] ??
                                          valueToInputText(item)
                                        const hasInvalidListValue =
                                          isListNumericValue &&
                                          !isNumberText(shownListValue)
                                        const selectedEnumNames = new Set(
                                          fieldValue.filter(
                                            (valueItem, itemIndex) =>
                                              itemIndex !== index &&
                                              typeof valueItem === 'string',
                                          ) as Array<string>,
                                        )

                                        return (
                                          <div
                                            key={`${field.name}-${index}`}
                                            className="grid grid-cols-[minmax(0,1fr)_2rem] gap-2"
                                          >
                                            {enumItems.length > 0 ? (
                                              <Select
                                                value={
                                                  typeof item === 'string'
                                                    ? item
                                                    : undefined
                                                }
                                                onValueChange={(nextValue) => {
                                                  const nextItems = [
                                                    ...fieldValue,
                                                  ]
                                                  nextItems[index] = nextValue
                                                  updateConfigField(
                                                    field.name,
                                                    nextItems,
                                                  )
                                                }}
                                              >
                                                <SelectTrigger className="w-full">
                                                  <SelectValue
                                                    placeholder={t(
                                                      'common.select',
                                                    )}
                                                  />
                                                </SelectTrigger>
                                                <SelectContent align="start">
                                                  {enumItems.map((enumItem) => (
                                                    <SelectItem
                                                      key={enumItem.name}
                                                      value={enumItem.name}
                                                      disabled={selectedEnumNames.has(
                                                        enumItem.name,
                                                      )}
                                                    >
                                                      <span className="grid min-w-0 grid-cols-[5.5rem_minmax(0,1fr)] items-center gap-2">
                                                        <span className="truncate">
                                                          {enumItem.name}
                                                        </span>
                                                        <span className="truncate text-xs text-muted-foreground">
                                                          {enumItem.description ??
                                                            ''}
                                                        </span>
                                                      </span>
                                                    </SelectItem>
                                                  ))}
                                                </SelectContent>
                                              </Select>
                                            ) : isListBooleanValue ? (
                                              <BooleanSelect
                                                value={Boolean(item)}
                                                onChange={(checked) => {
                                                  const nextItems = [
                                                    ...fieldValue,
                                                  ]
                                                  nextItems[index] = checked
                                                  updateConfigField(
                                                    field.name,
                                                    nextItems,
                                                  )
                                                }}
                                              />
                                            ) : (
                                              <Input
                                                value={shownListValue}
                                                type={
                                                  isListNumericValue
                                                    ? 'number'
                                                    : undefined
                                                }
                                                inputMode={
                                                  isListNumericValue
                                                    ? 'decimal'
                                                    : undefined
                                                }
                                                aria-invalid={
                                                  hasInvalidListValue
                                                }
                                                className={cn(
                                                  hasInvalidListValue &&
                                                    'border-destructive focus-visible:border-destructive focus-visible:ring-destructive/20',
                                                )}
                                                onChange={(event) => {
                                                  const nextValue =
                                                    event.target.value

                                                  if (
                                                    isListNumericValue &&
                                                    !isNumberText(nextValue)
                                                  ) {
                                                    setListValueDrafts(
                                                      (current) => ({
                                                        ...current,
                                                        [listDraftKey]:
                                                          nextValue,
                                                      }),
                                                    )
                                                    return
                                                  }

                                                  const nextItems = [
                                                    ...fieldValue,
                                                  ]
                                                  nextItems[index] =
                                                    isListNumericValue
                                                      ? Number(nextValue)
                                                      : nextValue
                                                  setListValueDrafts(
                                                    (current) => {
                                                      const next = {
                                                        ...current,
                                                      }
                                                      delete next[listDraftKey]
                                                      return next
                                                    },
                                                  )
                                                  updateConfigField(
                                                    field.name,
                                                    nextItems,
                                                  )
                                                }}
                                              />
                                            )}
                                            <Button
                                              type="button"
                                              variant="ghost"
                                              size="icon"
                                              className="size-8 shrink-0"
                                              onClick={() => {
                                                updateConfigField(
                                                  field.name,
                                                  fieldValue.filter(
                                                    (_, itemIndex) =>
                                                      itemIndex !== index,
                                                  ),
                                                )
                                                setListValueDrafts(
                                                  (current) => {
                                                    const next = { ...current }
                                                    delete next[listDraftKey]
                                                    return next
                                                  },
                                                )
                                              }}
                                            >
                                              <Trash2 className="size-4" />
                                            </Button>
                                            {hasInvalidListValue ? (
                                              <div className="col-span-2 text-xs text-destructive">
                                                {t(
                                                  'appConfig.valueMustBeNumber',
                                                )}
                                              </div>
                                            ) : null}
                                          </div>
                                        )
                                      })}
                                      <Button
                                        type="button"
                                        variant="outline"
                                        size="sm"
                                        className="justify-start"
                                        disabled={
                                          enumItems.length > 0 &&
                                          !enumItems.some(
                                            (enumItem) =>
                                              !fieldValue.includes(
                                                enumItem.name,
                                              ),
                                          )
                                        }
                                        onClick={() => {
                                          const nextEnumName = enumItems.find(
                                            (enumItem) =>
                                              !fieldValue.includes(
                                                enumItem.name,
                                              ),
                                          )?.name

                                          updateConfigField(field.name, [
                                            ...fieldValue,
                                            enumItems.length > 0
                                              ? (nextEnumName ?? '')
                                              : isListBooleanValue
                                                ? false
                                                : '',
                                          ])
                                        }}
                                      >
                                        <Plus className="size-4" />
                                        {t('action.add')}
                                      </Button>
                                    </div>
                                  ) : isPlainObject(fieldValue) ? (
                                    <div
                                      className="scrollbar-reserved grid max-h-48 gap-2 overflow-y-auto pr-1"
                                      onScroll={handleScrollAreaScroll}
                                    >
                                      {Object.entries(fieldValue).map(
                                        ([itemKey, itemValue], index) => {
                                          const draftKey = `${field.name}:${index}`
                                          const valueDraftKey = `${draftKey}:value`
                                          const shownItemKey =
                                            mapKeyDrafts[draftKey] ?? itemKey
                                          const shownItemValue =
                                            mapValueDrafts[valueDraftKey] ??
                                            valueToInputText(itemValue)
                                          const selectedMapKeys = new Set(
                                            Object.keys(fieldValue).filter(
                                              (_, itemIndex) =>
                                                itemIndex !== index,
                                            ),
                                          )
                                          const hasDuplicateKey =
                                            selectedMapKeys.has(shownItemKey)
                                          const hasInvalidKey =
                                            shownItemKey === '' ||
                                            (isMapIntKey &&
                                              !isIntegerKey(shownItemKey)) ||
                                            (isMapEnumKey &&
                                              !enumItems.some(
                                                (enumItem) =>
                                                  enumItem.name ===
                                                  shownItemKey,
                                              ))
                                          const hasKeyError =
                                            hasDuplicateKey || hasInvalidKey
                                          const hasInvalidValue =
                                            isMapNumericValue &&
                                            !isNumberText(shownItemValue)

                                          return (
                                            <div
                                              key={`${field.name}-${index}`}
                                              className="grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)_2rem] gap-2"
                                            >
                                              {isMapEnumKey ? (
                                                <Select
                                                  value={
                                                    shownItemKey === ''
                                                      ? undefined
                                                      : shownItemKey
                                                  }
                                                  onValueChange={(nextKey) => {
                                                    if (!nextKey) {
                                                      return
                                                    }

                                                    const nextEntries =
                                                      Object.entries(fieldValue)
                                                    nextEntries[index] = [
                                                      nextKey,
                                                      itemValue,
                                                    ]
                                                    setMapKeyDrafts(
                                                      (current) => {
                                                        const next = {
                                                          ...current,
                                                        }
                                                        delete next[draftKey]
                                                        return next
                                                      },
                                                    )
                                                    updateConfigField(
                                                      field.name,
                                                      Object.fromEntries(
                                                        nextEntries,
                                                      ),
                                                    )
                                                  }}
                                                >
                                                  <SelectTrigger className="w-full">
                                                    <SelectValue
                                                      placeholder={t(
                                                        'common.select',
                                                      )}
                                                    />
                                                  </SelectTrigger>
                                                  <SelectContent align="start">
                                                    {enumItems.map(
                                                      (enumItem) => (
                                                        <SelectItem
                                                          key={enumItem.name}
                                                          value={enumItem.name}
                                                          disabled={selectedMapKeys.has(
                                                            enumItem.name,
                                                          )}
                                                        >
                                                          <span className="grid min-w-0 grid-cols-[5.5rem_minmax(0,1fr)] items-center gap-2">
                                                            <span className="truncate">
                                                              {enumItem.name}
                                                            </span>
                                                            <span className="truncate text-xs text-muted-foreground">
                                                              {enumItem.description ??
                                                                ''}
                                                            </span>
                                                          </span>
                                                        </SelectItem>
                                                      ),
                                                    )}
                                                  </SelectContent>
                                                </Select>
                                              ) : (
                                                <Input
                                                  ref={(node) => {
                                                    if (node) {
                                                      mapKeyInputRefs.current.set(
                                                        draftKey,
                                                        node,
                                                      )
                                                    } else {
                                                      mapKeyInputRefs.current.delete(
                                                        draftKey,
                                                      )
                                                    }
                                                  }}
                                                  value={shownItemKey}
                                                  type={
                                                    isMapIntKey
                                                      ? 'number'
                                                      : undefined
                                                  }
                                                  inputMode={
                                                    isMapIntKey
                                                      ? 'numeric'
                                                      : undefined
                                                  }
                                                  aria-invalid={hasKeyError}
                                                  className={cn(
                                                    hasKeyError &&
                                                      'border-destructive focus-visible:border-destructive focus-visible:ring-destructive/20',
                                                  )}
                                                  onChange={(event) => {
                                                    const nextKey =
                                                      event.target.value
                                                    const isInvalidNextKey =
                                                      selectedMapKeys.has(
                                                        nextKey,
                                                      ) ||
                                                      (isMapIntKey &&
                                                        !isIntegerKey(nextKey))

                                                    if (isInvalidNextKey) {
                                                      setMapKeyDrafts(
                                                        (current) => ({
                                                          ...current,
                                                          [draftKey]: nextKey,
                                                        }),
                                                      )
                                                      return
                                                    }

                                                    const nextEntries =
                                                      Object.entries(fieldValue)
                                                    nextEntries[index] = [
                                                      nextKey,
                                                      itemValue,
                                                    ]
                                                    setMapKeyDrafts(
                                                      (current) => {
                                                        const next = {
                                                          ...current,
                                                        }
                                                        delete next[draftKey]
                                                        return next
                                                      },
                                                    )
                                                    updateConfigField(
                                                      field.name,
                                                      Object.fromEntries(
                                                        nextEntries,
                                                      ),
                                                    )
                                                  }}
                                                />
                                              )}
                                              {isMapBooleanValue ? (
                                                <BooleanSelect
                                                  value={Boolean(itemValue)}
                                                  onChange={(checked) => {
                                                    updateConfigField(
                                                      field.name,
                                                      {
                                                        ...fieldValue,
                                                        [itemKey]: checked,
                                                      },
                                                    )
                                                  }}
                                                />
                                              ) : (
                                                <Input
                                                  value={shownItemValue}
                                                  type={
                                                    isMapNumericValue
                                                      ? 'number'
                                                      : undefined
                                                  }
                                                  inputMode={
                                                    isMapNumericValue
                                                      ? 'decimal'
                                                      : undefined
                                                  }
                                                  aria-invalid={hasInvalidValue}
                                                  className={cn(
                                                    hasInvalidValue &&
                                                      'border-destructive focus-visible:border-destructive focus-visible:ring-destructive/20',
                                                  )}
                                                  onChange={(event) => {
                                                    const nextValue =
                                                      event.target.value

                                                    if (
                                                      isMapNumericValue &&
                                                      !isNumberText(nextValue)
                                                    ) {
                                                      setMapValueDrafts(
                                                        (current) => ({
                                                          ...current,
                                                          [valueDraftKey]:
                                                            nextValue,
                                                        }),
                                                      )
                                                      return
                                                    }

                                                    setMapValueDrafts(
                                                      (current) => {
                                                        const next = {
                                                          ...current,
                                                        }
                                                        delete next[
                                                          valueDraftKey
                                                        ]
                                                        return next
                                                      },
                                                    )
                                                    updateConfigField(
                                                      field.name,
                                                      {
                                                        ...fieldValue,
                                                        [itemKey]:
                                                          parseMapInputValue(
                                                            nextValue,
                                                            mapValueType,
                                                          ),
                                                      },
                                                    )
                                                  }}
                                                />
                                              )}
                                              <Button
                                                type="button"
                                                variant="ghost"
                                                size="icon"
                                                className="size-8"
                                                onClick={() => {
                                                  const nextMap = {
                                                    ...fieldValue,
                                                  }
                                                  delete nextMap[itemKey]
                                                  setMapKeyDrafts((current) => {
                                                    const next = { ...current }
                                                    delete next[draftKey]
                                                    return next
                                                  })
                                                  setMapValueDrafts(
                                                    (current) => {
                                                      const next = {
                                                        ...current,
                                                      }
                                                      delete next[valueDraftKey]
                                                      return next
                                                    },
                                                  )
                                                  updateConfigField(
                                                    field.name,
                                                    nextMap,
                                                  )
                                                }}
                                              >
                                                <Trash2 className="size-4" />
                                              </Button>
                                              {hasDuplicateKey ? (
                                                <div className="col-span-3 text-xs text-destructive">
                                                  {t('appConfig.keyExists')}
                                                </div>
                                              ) : null}
                                              {!hasDuplicateKey &&
                                              shownItemKey === '' ? (
                                                <div className="col-span-3 text-xs text-destructive">
                                                  {t('appConfig.keyRequired')}
                                                </div>
                                              ) : null}
                                              {!hasDuplicateKey &&
                                              shownItemKey !== '' &&
                                              hasInvalidKey ? (
                                                <div className="col-span-3 text-xs text-destructive">
                                                  {isMapEnumKey
                                                    ? t(
                                                        'appConfig.keyMustBeEnum',
                                                      )
                                                    : t(
                                                        'appConfig.keyMustBeInteger',
                                                      )}
                                                </div>
                                              ) : null}
                                              {hasInvalidValue ? (
                                                <div className="col-span-3 text-xs text-destructive">
                                                  {t(
                                                    'appConfig.valueMustBeNumber',
                                                  )}
                                                </div>
                                              ) : null}
                                            </div>
                                          )
                                        },
                                      )}
                                      <Button
                                        type="button"
                                        variant="outline"
                                        size="sm"
                                        className="justify-start"
                                        disabled={
                                          isMapEnumKey &&
                                          !enumItems.some(
                                            (enumItem) =>
                                              !Object.prototype.hasOwnProperty.call(
                                                fieldValue,
                                                enumItem.name,
                                              ),
                                          )
                                        }
                                        onClick={() => {
                                          const nextKey = ''

                                          const nextMap = {
                                            ...fieldValue,
                                            [nextKey]: '',
                                          }

                                          updateConfigField(field.name, nextMap)

                                          setMapKeyDrafts((current) => ({
                                            ...current,
                                            [`${field.name}:${Object.keys(nextMap).length - 1}`]:
                                              '',
                                          }))
                                        }}
                                      >
                                        <Plus className="size-4" />
                                        {t('action.add')}
                                      </Button>
                                    </div>
                                  ) : (
                                    <Input
                                      value={valueToInputText(fieldValue)}
                                      onChange={(event) =>
                                        updateConfigField(
                                          field.name,
                                          event.target.value,
                                        )
                                      }
                                    />
                                  )}
                                </div>
                              )
                            })}
                          </div>
                        ) : (
                          <div className="text-sm text-muted-foreground">
                            {t('appConfig.noFieldDescription')}
                          </div>
                        )}
                      </TabsContent>
                    ) : null}

                    <TabsContent
                      value="json"
                      className="scrollbar-reserved grid min-h-0 gap-2 overflow-y-auto pr-1"
                      onScroll={handleScrollAreaScroll}
                    >
                      <CodeMirror
                        value={value}
                        extensions={jsonPreviewExtensions}
                        readOnly={!selectedIsUnused && !selectedIsMismatched}
                        onChange={(nextValue) => {
                          if (selectedIsUnused || selectedIsMismatched) {
                            setValue(nextValue)
                          }
                        }}
                        basicSetup={{
                          autocompletion: true,
                          bracketMatching: true,
                          closeBrackets: true,
                          foldGutter: true,
                          highlightActiveLine: true,
                          highlightActiveLineGutter: true,
                          lineNumbers: true,
                        }}
                        minHeight="28rem"
                        className={cn(
                          'overflow-hidden rounded-lg border bg-background text-[13px]',
                          valueIsValidJson
                            ? 'border-input'
                            : 'border-destructive',
                        )}
                        theme="light"
                      />
                      {!valueIsValidJson ? (
                        <span className="text-sm text-destructive">
                          {t('appConfig.invalidJson')}
                        </span>
                      ) : null}
                    </TabsContent>
                  </Tabs>
                </div>
              </div>
            </div>
          ) : null}
        </main>
      </div>
    </section>
  )
}
