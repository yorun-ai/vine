import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { formatConfigYaml, normalizeConfigYaml } from './config-yaml-document'
import { ConfigJsonEditor } from './config-json-editor'
import { normalizeConfigJson, getFreeConfigJsonErrors } from './config-json-document'
import { useConfigAccess } from '@/lib/config-access'
import { DomainFilter } from '@/components/domain-filter'
import { SkelName } from '@/components/skel-name'
import { SearchInput } from '@/components/ui/search-input'
import * as React from 'react'
import { useNavigate, useRouterState } from '@tanstack/react-router'
import CodeMirror from '@uiw/react-codemirror'
import { json } from '@codemirror/lang-json'
import {
  Braces,
  Copy,
  Replace,
  Loader2,
  Plus,
  RefreshCw,
  RotateCcw,
  Save,
  Trash2,
} from 'lucide-react'
import { toast } from 'sonner'

import {
  DeprecatedBadge,
  DeprecatedNotice,
} from '@/components/deprecated'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
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
import { ListDetailLayout } from '@/components/ui/list-detail-layout'
import { Skeleton } from '@/components/ui/skeleton'
import { vrpcClient } from '@/config/vrpc-client'
import { copyTextToClipboard } from '@/lib/clipboard'
import { useLocale } from '@/i18n'
import { cn } from '@/lib/utils'
import {
  createAppConfigApiService,
  createSkeletonApiService,
} from '@/skeled/admin'
import type {
  AppConfigItem,
  AppConfigSchema,
  SkeletonData,
} from '@/skeled/admin'

const appConfigService = createAppConfigApiService(vrpcClient)
const skeletonService = createSkeletonApiService(vrpcClient)
const jsonExtensions = [json()]
const APP_CONFIG_LIST_DEFAULT_WIDTH = 352
const configFormatStorageKey = 'vine.hub.config.editorFormat'

interface AppConfigPageProps {
  routeKey?: string
}

type TypeDefinitionIndex = Map<string, SkeletonData>
type AppConfigStatus = 'NORMAL' | 'UNUSED' | 'UNCONFIGURED' | 'MISMATCH'

const emptyConfigValue = '{}'

interface ConfigMismatchIssue {
  repair?: 'add' | 'remove' | 'reset'
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

function getErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : 'Request failed'
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return value !== null && !Array.isArray(value) && typeof value === 'object'
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
    return [{ text: t('appConfig.valueMustBeObject'), repair: 'reset' as const }]
  }

  const issues: Array<ConfigMismatchIssue> = []
  const fieldsByName = new Map(
    schema.fields.map((field) => [field.name, field]),
  )

  for (const field of schema.fields) {
    if (!Object.prototype.hasOwnProperty.call(parsed, field.name)) {
      issues.push({
        fieldName: field.name,
        repair: 'add',
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
        repair: 'remove',
        text: t('appConfig.unknownField').replace('{field}', key),
      })
    }
  }

  return issues
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

export function AppConfigPage({ routeKey }: AppConfigPageProps) {
  const { readOnly } = useConfigAccess()
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
  const [value, setValue] = React.useState('')
  const [rawReplacement, setRawReplacement] = React.useState(false)
  const [editorFormat, setEditorFormat] = React.useState<'json5' | 'yaml'>(() => {
    try {
      return window.localStorage.getItem(configFormatStorageKey) === 'yaml' ? 'yaml' : 'json5'
    } catch {
      return 'json5'
    }
  })
  const [replaceFormat, setReplaceFormat] = React.useState<'json5' | 'yaml'>('json5')
  const [replaceDialogOpen, setReplaceDialogOpen] = React.useState(false)
  const [replaceDraft, setReplaceDraft] = React.useState('')
  const [jsonDraftInvalid, setJsonDraftInvalid] = React.useState(false)
  const [jsonEditorRevision, setJsonEditorRevision] = React.useState(0)
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
  const configDomains = React.useMemo(
    () =>
      Array.from(
        new Set(
          appConfigs
            .map((config) => splitConfigSkelName(configSkelName(config)).domainPart)
            .filter(Boolean),
        ),
      ).sort((left, right) => left.localeCompare(right)),
    [appConfigs],
  )

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

  const valueIsValidJson = React.useMemo(() => isValidJson(value) && getFreeConfigJsonErrors(value).length === 0, [value])
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
  const mismatchMessages = React.useMemo(
    () => new Map(mismatchIssues.flatMap((issue) =>
      issue.fieldName ? [[issue.fieldName, issue.text] as const] : [],
    )),
    [mismatchIssues],
  )
  const visibleMismatchMessages = React.useMemo(
    () => jsonDraftInvalid || !valueIsValidJson ? new Map<string, string>() : mismatchMessages,
    [jsonDraftInvalid, valueIsValidJson, mismatchMessages],
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
  const hasChanges =
    selectedAppConfig !== null &&
    (selectedIsUnconfigured || jsonDraftInvalid || value !== selectedSavedValue)
  const canSave =
    !readOnly &&
    selectedAppConfig !== null &&
    hasChanges &&
    valueIsValidJson &&
    !jsonDraftInvalid &&
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
    try {
      window.localStorage.setItem(configFormatStorageKey, editorFormat)
    } catch {
      return
    }
  }, [editorFormat])

  React.useEffect(() => {
    setReplaceDialogOpen(false)
    setReplaceDraft('')
    setRawReplacement(false)
  }, [selectedKey])

  React.useEffect(() => {
    routeKeyRef.current = effectiveRouteKey
    setSelectedKey(effectiveRouteKey ?? null)
  }, [effectiveRouteKey])

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

    setRawReplacement(false)
    setValue(selectedSavedValue)
    setJsonEditorRevision((revision) => revision + 1)
  }

  function replaceConfigJson(text: string) {
    if (readOnly) {
      return
    }
    let nextValue = text
    let invalid = false
    try {
      nextValue = replaceFormat === 'yaml' ? normalizeConfigYaml(text) : normalizeConfigJson(text)
    } catch {
      invalid = true
    }
    setRawReplacement(invalid)
    setEditorFormat(replaceFormat)
    setValue(nextValue)
    setJsonEditorRevision((revision) => revision + 1)
    setReplaceDialogOpen(false)
    setReplaceDraft('')
  }

  async function handleCopyConfigJson() {
    try {
      await copyTextToClipboard(editorFormat === 'yaml' ? formatConfigYaml(value) : value)
      toast.success(t('appConfig.configCopied'))
    } catch (error) {
      toast.error(getErrorMessage(error))
    }
  }

  return (
    <ListDetailLayout
      defaultWidth={APP_CONFIG_LIST_DEFAULT_WIDTH}
      resizeLabel={t('appConfig.resizeList')}
      listFooter={t('appConfig.itemCount').replace(
        '{count}',
        String(filteredConfigs.length),
      )}
      listHeader={
        <>
          <div className="relative">
            <SearchInput
              value={query}
              onValueChange={setQuery}
              placeholder={t('common.searchSkelName')}
            />
          </div>
          <div className="mt-3 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
            <DomainFilter
              domains={configDomains}
              query={query}
              onQueryChange={setQuery}
              loading={listLoading}
            />
            <div className="ml-auto flex shrink-0 items-center gap-2">
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
                disabled={readOnly}
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
        </>
      }
      list={
        listLoading ? (
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
                    <SkelName skelName={configSkelName(config)} />
                  </span>
                </a>
              )
            })}
          </div>
        )
      }
    >
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
                  disabled={readOnly || creating || createExistingConfig !== null}
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
                  <DeprecatedBadge
                    deprecated={Boolean(selectedSchema?.deprecated)}
                  />
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
                          className="font-mono text-muted-foreground underline-offset-2 hover:underline"
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
                <DeprecatedNotice
                  deprecated={Boolean(selectedSchema?.deprecated)}
                  deprecatedReason={selectedSchema?.deprecatedReason}
                  className="mt-3"
                />
              </div>

              <div className="flex flex-wrap items-start justify-end gap-2">
                {selectedIsUnused ? (
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => setDeleteDialogOpen(true)}
                    disabled={readOnly || removing}
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
                        disabled={readOnly || removing}
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
              <Tabs
                value={editorFormat}
                onValueChange={(format) => {
                  if (!jsonDraftInvalid && valueIsValidJson && (format === 'json5' || format === 'yaml')) {
                    setEditorFormat(format)
                  }
                }}
                className="min-h-0 flex-1 gap-0 overflow-hidden rounded-lg border border-input bg-background"
              >
                <div className="flex h-14 shrink-0 items-center justify-between gap-3 border-b border-border bg-muted/20 pr-4">
                  <div className="flex h-full items-center gap-3">
                    <TabsList className="h-full rounded-none bg-transparent p-0 group-data-horizontal/tabs:h-full" aria-label={t('appConfig.format')}>
                    {(['json5', 'yaml'] as const).map((format) => (
                      <TabsTrigger
                        key={format}
                        value={format}
                        className="h-full min-w-20 rounded-none border-0 border-b-2 border-b-transparent px-4 after:hidden data-active:border-b-primary data-active:bg-primary/5 data-active:text-primary data-active:shadow-none dark:data-active:bg-primary/5 dark:data-active:text-primary"
                        disabled={jsonDraftInvalid || !valueIsValidJson}
                      >
                        {format.toUpperCase()}
                      </TabsTrigger>
                    ))}
                    </TabsList>
                  </div>
                  <div className="ml-auto flex shrink-0 items-center gap-2">
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      className="h-7 px-2 text-xs"
                      disabled={jsonDraftInvalid || !valueIsValidJson}
                      onClick={() => void handleCopyConfigJson()}
                    >
                      <Copy />
                      {t('appConfig.copy')}
                    </Button>
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      className="h-7 px-2 text-xs"
                      disabled={readOnly || saving}
                      onClick={() => {
                        setReplaceDraft('')
                        setReplaceFormat(editorFormat)
                        setReplaceDialogOpen(true)
                      }}
                    >
                      <Replace />
                      {t('appConfig.replace')}
                    </Button>
                    <Dialog open={replaceDialogOpen} onOpenChange={setReplaceDialogOpen}>
                      <DialogContent>
                        <DialogHeader>
                          <DialogTitle>{t('appConfig.replaceTitle')}</DialogTitle>
                          <DialogDescription>{t('appConfig.replaceDescription')}</DialogDescription>
                        </DialogHeader>
                        <select
                          aria-label={t('appConfig.format')}
                          className="h-8 rounded-md border border-input bg-background px-2 text-sm"
                          value={replaceFormat}
                          onChange={(event) => setReplaceFormat(event.target.value as 'json5' | 'yaml')}
                        >
                          <option value="json5">JSON5</option>
                          <option value="yaml">YAML</option>
                        </select>
                        <textarea
                          autoFocus
                          aria-label={t('appConfig.replaceTitle')}
                          className="h-64 w-full resize-y rounded-md border border-input p-3 font-mono text-sm"
                          value={replaceDraft}
                          onChange={(event) => setReplaceDraft(event.target.value)}
                        />
                        <DialogFooter>
                          <DialogClose render={<Button variant="outline" />}>
                            {t('action.cancel')}
                          </DialogClose>
                          <Button disabled={readOnly || saving} onClick={() => replaceConfigJson(replaceDraft)}>
                            {t('appConfig.replaceSubmit')}
                          </Button>
                        </DialogFooter>
                      </DialogContent>
                    </Dialog>
                  </div>

                </div>

                <TabsContent
                  value={editorFormat}
                  className="scrollbar-reserved min-h-0 flex-1 overflow-y-auto"
                  onScroll={handleScrollAreaScroll}
                >
                  <ConfigJsonEditor
                    key={`${selectedKey}:${jsonEditorRevision}:${editorFormat}`}
                    format={editorFormat}
                    rawValue={rawReplacement}
                    value={value}
                    fields={selectedSchema?.fields ?? []}
                    lockKeys={!rawReplacement && valueIsValidJson && selectedSchema !== null && !selectedIsUnused && configObject !== null}
                    mismatchMessages={visibleMismatchMessages}
                    dirtyFields={dirtyFields}
                    typeIndex={typeIndex}
                    onTypeClick={navigateToTypeDefinition}
                    readOnly={readOnly}
                    onChange={(nextValue) => {
                      setRawReplacement(false)
                      setValue(nextValue)
                    }}
                    onInvalidChange={setJsonDraftInvalid}
                  />
                </TabsContent>
              </Tabs>
              <div className="h-28 shrink-0 overflow-y-auto" aria-live="polite">
                {!valueIsValidJson || jsonDraftInvalid || mismatchIssues.length > 0 ? (
                <Alert variant="destructive" className="min-h-full">
                  <AlertTitle>
                    {t(!valueIsValidJson || jsonDraftInvalid ? 'appConfig.formatErrorTitle' : 'appConfig.mismatchTitle')}
                  </AlertTitle>
                  <AlertDescription>
                    {!valueIsValidJson || jsonDraftInvalid ? (
                      <span>{t(editorFormat === 'yaml' ? 'appConfig.invalidYaml' : 'appConfig.invalidJson5')}</span>
                    ) : (
                    <ul className="grid gap-2">
                      {mismatchIssues.map((issue) => (
                        <li key={issue.text} className="flex items-center justify-between gap-3">
                          <span>{issue.text}</span>
                          {issue.repair ? (
                            <Button
                              type="button"
                              variant="outline"
                              size="sm"
                              disabled={readOnly || jsonDraftInvalid}
                              onClick={() => {
                                const next = { ...configObject }
                                if (issue.repair === 'reset') {
                                  setValue(defaultConfigValue(selectedSchema))
                                } else {
                                  if (issue.repair === 'add') {
                                    next[issue.fieldName!] = defaultConfigObject(selectedSchema)[issue.fieldName!]
                                  } else {
                                    delete next[issue.fieldName!]
                                  }
                                  setValue(stringifyConfigObject(next))
                                }
                                setJsonEditorRevision((revision) => revision + 1)
                              }}
                            >
                              {t(issue.repair === 'add'
                                ? 'appConfig.addMissingField'
                                : issue.repair === 'remove'
                                  ? 'appConfig.removeUnknownField'
                                  : 'appConfig.resetInvalidObject')}
                            </Button>
                          ) : null}
                        </li>
                      ))}
                    </ul>
                    )}
                  </AlertDescription>
                </Alert>
                ) : null}
              </div>
            </div>
          </div>
        </div>
      ) : null}
    </ListDetailLayout>
  )
}
