import * as React from 'react'
import { Navigate, useNavigate } from '@tanstack/react-router'
import {
  CheckCircle2,
  ChevronRight,
  CircleCheck,
  FileUp,
  Loader2,
  MoveRight,
  RefreshCw,
} from 'lucide-react'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { vrpcClient } from '@/config/vrpc-client'
import { useLocale } from '@/i18n'
import { cn } from '@/lib/utils'
import { createMaintenanceService } from '@/skeled'
import type {
  SeedItemSelection,
  SeedPreview,
} from '@/skeled'

const maintenanceService = createMaintenanceService(vrpcClient)
const dataUpdateStorageKey = 'vine.hub.maintenance.dataUpdate'

const kindLabels: Record<string, string> = {
  app_config: 'Config',
  portal_site: 'Portal Site',
  portal_rule: 'Entry Rule',
  portal_cert: 'Entry Cert',
}

type DataUpdateStage = 'upload' | 'preview' | 'result'

type DataUpdateSession = {
  fileName: string
  seedContent: string
  preview: SeedPreview
  selectedKeys: Array<string>
  appliedCount?: number
  appliedAt?: string
}

const stages: Array<{
  id: DataUpdateStage
  title: string
  description: string
}> = [
  {
    id: 'upload',
    title: 'Upload',
    description: 'Select YAML',
  },
  {
    id: 'preview',
    title: 'Confirm',
    description: 'Select Entities',
  },
  {
    id: 'result',
    title: 'Done',
    description: 'View Result',
  },
]

function selectionKey(kind: string, name: string) {
  return `${kind}\u0000${name}`
}

function getErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : 'Request failed'
}

function readSession(): DataUpdateSession | null {
  if (typeof window === 'undefined') {
    return null
  }

  const raw = window.sessionStorage.getItem(dataUpdateStorageKey)
  if (!raw) {
    return null
  }

  try {
    return JSON.parse(raw) as DataUpdateSession
  } catch {
    window.sessionStorage.removeItem(dataUpdateStorageKey)
    return null
  }
}

function writeSession(session: DataUpdateSession) {
  window.sessionStorage.setItem(dataUpdateStorageKey, JSON.stringify(session))
}

function clearSession() {
  window.sessionStorage.removeItem(dataUpdateStorageKey)
}

function useDataUpdateSession() {
  return React.useState<DataUpdateSession | null>(() => readSession())
}

function initialSelection(preview: SeedPreview) {
  const selected = new Set<string>()
  for (const item of preview.items) {
    if (item.fields.some((field) => field.changed)) {
      selected.add(selectionKey(item.kind, item.name))
    }
  }
  return Array.from(selected)
}

function selectedToPayload(selected: Set<string>): Array<SeedItemSelection> {
  return Array.from(selected).map((key) => {
    const [kind, name] = key.split('\u0000')
    return { kind, name }
  })
}

function changedFieldCount(preview: SeedPreview) {
  return preview.items.reduce(
    (sum, item) => sum + item.fields.filter((field) => field.changed).length,
    0,
  )
}

function formatValue(value: string, emptyText: string) {
  if (value === '') {
    return <span className="text-muted-foreground/70">{emptyText}</span>
  }
  return value
}

function PageShell({
  stage,
  children,
}: {
  stage: DataUpdateStage
  children: React.ReactNode
}) {
  return (
    <section className="flex h-[calc(100dvh-3.5rem)] flex-col overflow-hidden bg-white">
      <div className="border-b border-border/70 px-6 py-3">
        <div className="flex w-full">
          <StageRow current={stage} />
        </div>
      </div>
      <div
        className="min-h-0 flex-1 overflow-y-auto p-6"
        style={{ scrollbarGutter: 'stable' }}
      >
        {children}
      </div>
    </section>
  )
}

function StageRow({ current }: { current: DataUpdateStage }) {
  const navigate = useNavigate()
  const { tText } = useLocale()
  const currentIndex = stages.findIndex((stage) => stage.id === current)

  return (
    <div className="flex w-full flex-row items-center bg-background">
      {stages.map((stage, index) => {
        const active = stage.id === current
        const done = index < currentIndex
        const canNavigateToUpload = stage.id === 'upload' && !active

        return (
          <React.Fragment key={stage.id}>
            <button
              type="button"
              disabled={!canNavigateToUpload}
              className={cn(
                'flex min-w-0 flex-1 items-center justify-center gap-3 px-6 py-3 text-left',
                canNavigateToUpload &&
                  'cursor-pointer rounded-md hover:bg-muted/50',
                !canNavigateToUpload && 'cursor-default',
              )}
              onClick={() => {
                if (canNavigateToUpload) {
                  void navigate({ to: '/maintenance/data-update/upload' })
                }
              }}
            >
              <div
                className={cn(
                  'grid size-7 shrink-0 place-items-center rounded-full border text-xs font-medium',
                  active && 'border-primary bg-primary text-primary-foreground',
                  done && 'border-primary text-primary',
                )}
              >
                {done ? <CheckCircle2 className="size-4" /> : index + 1}
              </div>
              <div className="min-w-0 text-center">
                <div className="text-sm font-medium">{tText(stage.title)}</div>
                <div className="truncate text-xs text-muted-foreground">
                  {tText(stage.description)}
                </div>
              </div>
            </button>
            {index < stages.length - 1 ? (
              <MoveRight className="size-5 shrink-0 text-muted-foreground/60" />
            ) : null}
          </React.Fragment>
        )
      })}
    </div>
  )
}

export function DataUpdateUploadPage() {
  const navigate = useNavigate()
  const { t } = useLocale()
  const inputRef = React.useRef<HTMLInputElement>(null)
  const [selectedFile, setSelectedFile] = React.useState<File | null>(null)
  const [dragging, setDragging] = React.useState(false)
  const [loading, setLoading] = React.useState(false)

  const uploadFile = React.useCallback(
    async (file: File | undefined) => {
      if (!file || loading) {
        return
      }

      setSelectedFile(file)
      setLoading(true)
      try {
        const content = await file.text()
        const preview = await maintenanceService.previewSeedYaml({
          content,
        })
        writeSession({
          fileName: file.name,
          seedContent: content,
          preview,
          selectedKeys: initialSelection(preview),
        })
        await navigate({ to: '/maintenance/data-update/preview' })
      } catch (error) {
        toast.error(getErrorMessage(error))
      } finally {
        setLoading(false)
      }
    },
    [loading, navigate],
  )

  const handleFileChange = React.useCallback(
    (event: React.ChangeEvent<HTMLInputElement>) => {
      const input = event.currentTarget
      void uploadFile(event.target.files?.[0])
      input.value = ''
    },
    [uploadFile],
  )

  const handleDrop = React.useCallback(
    (event: React.DragEvent<HTMLDivElement>) => {
      event.preventDefault()
      setDragging(false)
      void uploadFile(event.dataTransfer.files?.[0])
    },
    [uploadFile],
  )

  const openFilePicker = React.useCallback(() => {
    if (!loading) {
      inputRef.current?.click()
    }
  }, [loading])

  const uploadText = React.useMemo(() => {
    if (loading) {
      return t('dataUpdate.uploading')
    }
    if (selectedFile) {
      return selectedFile.name
    }
    return t('dataUpdate.selectYaml')
  }, [loading, selectedFile, t])

  return (
    <PageShell stage="upload">
      <div className="grid h-full min-h-[24rem] place-items-center">
        <div
          className={cn(
            'w-full max-w-sm cursor-pointer rounded-lg border border-dashed bg-muted/20 text-center transition-colors hover:border-primary/50 hover:bg-primary/5',
            dragging && 'border-primary bg-primary/5',
            loading && 'cursor-default opacity-80',
          )}
          style={{ padding: '64px 32px' }}
          role="button"
          tabIndex={0}
          onClick={openFilePicker}
          onDragEnter={(event) => {
            event.preventDefault()
            setDragging(true)
          }}
          onDragOver={(event) => event.preventDefault()}
          onDragLeave={() => setDragging(false)}
          onDrop={handleDrop}
          onKeyDown={(event) => {
            if (event.key === 'Enter' || event.key === ' ') {
              event.preventDefault()
              openFilePicker()
            }
          }}
        >
          <div
            className="inline-flex items-center justify-center rounded-xl bg-primary/10 text-primary"
            style={{ height: 48, marginBottom: 20, width: 48 }}
          >
            {loading ? (
              <Loader2
                className="animate-spin"
                style={{ height: 24, width: 24 }}
              />
            ) : (
              <FileUp style={{ height: 24, width: 24 }} />
            )}
          </div>
          <div
            className={cn(
              'mx-auto max-w-full text-sm font-medium text-foreground',
              selectedFile &&
                'truncate font-mono text-xs text-muted-foreground',
            )}
          >
            {uploadText}
          </div>
          <Input
            ref={inputRef}
            type="file"
            accept=".yaml,.yml"
            className="hidden"
            disabled={loading}
            onChange={handleFileChange}
          />
        </div>
      </div>
    </PageShell>
  )
}
export function DataUpdatePreviewPage() {
  const navigate = useNavigate()
  const { t } = useLocale()
  const [session, setSession] = useDataUpdateSession()
  const [applying, setApplying] = React.useState(false)
  const [selectedItems, setSelectedItems] = React.useState<Set<string>>(
    () => new Set(session?.selectedKeys ?? []),
  )

  React.useEffect(() => {
    setSelectedItems(new Set(session?.selectedKeys ?? []))
  }, [session])

  if (!session) {
    return <Navigate to="/maintenance/data-update/upload" replace />
  }

  const selectedCount = selectedItems.size
  const preview = session.preview
  const changedItemKeys = preview.items
    .filter((item) => item.fields.some((field) => field.changed))
    .map((item) => selectionKey(item.kind, item.name))
  const allChangedItemsSelected =
    changedItemKeys.length > 0 &&
    changedItemKeys.every((key) => selectedItems.has(key))
  const hasChangedItems = changedItemKeys.length > 0

  const toggleItem = (kind: string, name: string, changed: boolean) => {
    if (!changed) {
      return
    }

    const key = selectionKey(kind, name)
    setSelectedItems((current) => {
      const next = new Set(current)
      if (next.has(key)) {
        next.delete(key)
      } else {
        next.add(key)
      }
      const nextSession = { ...session, selectedKeys: Array.from(next) }
      writeSession(nextSession)
      setSession(nextSession)
      return next
    })
  }

  const toggleAllChangedItems = (checked: boolean) => {
    const next = checked ? new Set(changedItemKeys) : new Set<string>()
    const nextSession = { ...session, selectedKeys: Array.from(next) }
    setSelectedItems(next)
    writeSession(nextSession)
    setSession(nextSession)
  }

  const handleApply = async () => {
    if (selectedItems.size === 0) {
      return
    }

    setApplying(true)
    try {
      const appliedCount = selectedItems.size
      const nextPreview = await maintenanceService.applySeedYaml({
        content: session.seedContent,
        selections: selectedToPayload(selectedItems),
      })
      const nextSession = {
        ...session,
        preview: nextPreview,
        selectedKeys: initialSelection(nextPreview),
        appliedCount,
        appliedAt: new Date().toISOString(),
      }
      writeSession(nextSession)
      setSession(nextSession)
      toast.success(t('dataUpdate.applied'))
      await navigate({ to: '/maintenance/data-update/result' })
    } catch (error) {
      toast.error(getErrorMessage(error))
    } finally {
      setApplying(false)
    }
  }

  return (
    <PageShell stage="preview">
      <div className="grid gap-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex min-w-0 items-center gap-2 text-sm text-muted-foreground">
            <CheckCircle2 className="h-4 w-4 shrink-0 text-foreground" />
            <span className="truncate">
              {t('dataUpdate.previewSummary')
                .replace('{fileName}', session.fileName)
                .replace('{items}', String(preview.items.length))
                .replace('{fields}', String(changedFieldCount(preview)))
                .replace('{selected}', String(selectedCount))}
            </span>
          </div>

          <div className="flex shrink-0 flex-wrap items-center justify-end gap-3">
            {hasChangedItems ? (
              <>
                <label className="flex cursor-pointer items-center gap-2 text-sm text-muted-foreground">
                  <Checkbox
                    checked={allChangedItemsSelected}
                    disabled={applying}
                    onCheckedChange={(checked) =>
                      toggleAllChangedItems(checked === true)
                    }
                  />
                  {t('dataUpdate.selectAll')}
                </label>
                <Button
                  type="button"
                  disabled={selectedCount === 0 || applying}
                  onClick={() => void handleApply()}
                >
                  {applying ? (
                    <Loader2 className="animate-spin" />
                  ) : (
                    <RefreshCw />
                  )}
                  {t('dataUpdate.applySelected')}
                </Button>
              </>
            ) : (
              <Button
                type="button"
                onClick={() => {
                  clearSession()
                  void navigate({ to: '/maintenance/data-update/upload' })
                }}
              >
                {t('dataUpdate.back')}
              </Button>
            )}
          </div>
        </div>

        <SeedDiffList
          preview={preview}
          applying={applying}
          selectedItems={selectedItems}
          onToggleItem={toggleItem}
          showUnchangedToggle
        />
      </div>
    </PageShell>
  )
}

export function DataUpdateResultPage() {
  const navigate = useNavigate()
  const { t } = useLocale()
  const [session] = useDataUpdateSession()

  if (!session) {
    return <Navigate to="/maintenance/data-update/upload" replace />
  }

  return (
    <PageShell stage="result">
      <div className="grid h-full min-h-[24rem] place-items-center">
        <div className="grid justify-items-center gap-4 text-center">
          <CircleCheck
            className="mx-auto text-primary"
            strokeWidth={1.8}
            style={{ height: 80, width: 80 }}
          />
          <div>
            <div className="text-lg font-semibold">
              {t('dataUpdate.success')}
            </div>
            <div className="mt-1 text-sm text-muted-foreground">
              {t('dataUpdate.resultSummary')
                .replace('{fileName}', session.fileName)
                .replace('{count}', String(session.appliedCount ?? 0))}
            </div>
          </div>
          <Button
            type="button"
            onClick={() => {
              clearSession()
              void navigate({ to: '/maintenance/data-update/upload' })
            }}
          >
            {t('dataUpdate.done')}
          </Button>
        </div>
      </div>
    </PageShell>
  )
}

function SeedDiffList({
  preview,
  applying,
  selectedItems,
  onToggleItem,
  readOnly = false,
  showUnchangedToggle = false,
}: {
  preview: SeedPreview
  applying: boolean
  selectedItems: Set<string>
  onToggleItem: (kind: string, name: string, changed: boolean) => void
  readOnly?: boolean
  showUnchangedToggle?: boolean
}) {
  const { t } = useLocale()
  const [showUnchangedItems, setShowUnchangedItems] = React.useState(false)
  const changedItems = preview.items.filter((item) =>
    item.fields.some((field) => field.changed),
  )
  const unchangedItems = preview.items.filter(
    (item) => !item.fields.some((field) => field.changed),
  )
  const items = showUnchangedToggle ? changedItems : preview.items

  const renderItem = (item: SeedPreview['items'][number]) => {
    const changedCount = item.fields.filter((field) => field.changed).length
    const changed = changedCount > 0
    const key = selectionKey(item.kind, item.name)
    const selected = selectedItems.has(key)
    const selectable = changed && !readOnly && !applying

    return (
      <section
        key={`${item.kind}:${item.name}`}
        role={selectable ? 'button' : undefined}
        tabIndex={selectable ? 0 : undefined}
        className={cn(
          'overflow-hidden rounded-lg border bg-card transition-colors',
          selected
            ? 'bg-primary/[0.03] shadow-[inset_3px_0_0_#0f766e,0_0_0_1px_#0f766e]'
            : 'border-border',
          selectable &&
            (selected
              ? 'cursor-pointer hover:bg-primary/[0.05]'
              : 'cursor-pointer hover:border-primary/50 hover:bg-primary/[0.02]'),
        )}
        style={
          selected
            ? {
                borderColor: '#0f766e',
              }
            : undefined
        }
        onClick={() => {
          if (selectable) {
            onToggleItem(item.kind, item.name, changed)
          }
        }}
        onKeyDown={(event) => {
          if (selectable && (event.key === 'Enter' || event.key === ' ')) {
            event.preventDefault()
            onToggleItem(item.kind, item.name, changed)
          }
        }}
      >
        <div
          className={cn(
            'flex flex-wrap items-center justify-between gap-2 border-b px-4 py-3',
            selected ? 'border-[#0f766e]' : 'bg-muted/20',
          )}
          style={
            selected
              ? {
                  backgroundColor: '#dff3ef',
                }
              : undefined
          }
        >
          <div className="flex min-w-0 items-center gap-2">
            <Checkbox
              checked={selected}
              disabled={readOnly || !changed || applying}
              onClick={(event) => event.stopPropagation()}
              onCheckedChange={() =>
                onToggleItem(item.kind, item.name, changed)
              }
            />
            <Badge variant={item.exists ? 'secondary' : 'outline'}>
              {kindLabels[item.kind] ?? item.kind}
            </Badge>
            <span className="truncate font-mono text-sm font-medium">
              {item.name}
            </span>
            {!item.exists ? <Badge variant="outline">New</Badge> : null}
          </div>
          <div className="text-xs text-muted-foreground">
            {changedCount} changed
          </div>
        </div>

        <div className={cn(selected && 'bg-primary/[0.03]')}>
          <Table>
            <TableHeader className={cn(selected && 'bg-primary/[0.04]')}>
              <TableRow
                className={cn(
                  'hover:bg-transparent',
                  selected && 'bg-primary/[0.04]',
                )}
              >
                <TableHead
                  className={cn(selected && 'bg-primary/[0.04]', 'w-44')}
                >
                  {t('dataUpdate.field')}
                </TableHead>
                <TableHead className={cn(selected && 'bg-primary/[0.04]')}>
                  {t('dataUpdate.currentValue')}
                </TableHead>
                <TableHead className={cn(selected && 'bg-primary/[0.04]')}>
                  {t('dataUpdate.seedValue')}
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {item.fields.map((field) => (
                <TableRow
                  key={field.name}
                  className={cn(
                    selected && 'bg-primary/[0.04] hover:bg-primary/[0.04]',
                    field.changed &&
                      (selected
                        ? 'bg-primary/10 hover:bg-primary/10'
                        : 'bg-amber-50/60 hover:bg-amber-50/60'),
                  )}
                >
                  <TableCell className="font-mono text-xs">
                    {field.name}
                  </TableCell>
                  <TableCell className="max-w-[20rem] whitespace-normal break-all font-mono text-xs text-muted-foreground">
                    {formatValue(
                      field.currentValue,
                      t('dataUpdate.emptyValue'),
                    )}
                  </TableCell>
                  <TableCell className="max-w-[20rem] whitespace-normal break-all font-mono text-xs">
                    {formatValue(field.seedValue, t('dataUpdate.emptyValue'))}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </section>
    )
  }

  return (
    <div className="grid gap-4">
      {items.map(renderItem)}
      {showUnchangedToggle && unchangedItems.length > 0 ? (
        <section className="grid gap-4">
          <button
            type="button"
            className="flex w-full items-center justify-center gap-2 px-4 py-2 text-sm text-muted-foreground transition-colors hover:bg-muted/30 hover:text-foreground"
            onClick={() => setShowUnchangedItems((current) => !current)}
          >
            <ChevronRight
              className={cn(
                'h-4 w-4 transition-transform',
                showUnchangedItems && 'rotate-90',
              )}
            />
            {showUnchangedItems
              ? t('dataUpdate.collapseUnchanged').replace(
                  '{count}',
                  String(unchangedItems.length),
                )
              : t('dataUpdate.expandUnchanged').replace(
                  '{count}',
                  String(unchangedItems.length),
                )}
          </button>
          {showUnchangedItems ? unchangedItems.map(renderItem) : null}
        </section>
      ) : null}
    </div>
  )
}
