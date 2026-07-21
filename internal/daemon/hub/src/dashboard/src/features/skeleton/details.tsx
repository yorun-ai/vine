import * as React from 'react'
import {
  Braces,
  Globe2,
  KeyRound,
  Server,
  ShieldCheck,
} from 'lucide-react'

import { Badge, badgeVariants } from '@/components/ui/badge'
import { useLocale } from '@/i18n'
import { cn } from '@/lib/utils'
import type {
  SkeletonActorRef,
  SkeletonData,
  SkeletonEnumItem,
  SkeletonField,
  SkeletonMethod,
  SkeletonPermExpr,
  SkeletonResourceAction,
  SkeletonResourceCheck,
  SkeletonServiceItem,
  SkeletonTrigger,
  SkeletonWebItem,
} from '@/skeled'

import type { SkeletonItem, SkeletonKind, TypeDefinitionIndex } from './model'
import {
  findTypeDefinition,
  skeletonActorHref,
  skeletonItemHref,
} from './model'

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

function TypeText({
  type,
  typeIndex,
  domainSchemaHash,
  onTypeClick,
}: {
  type: string
  typeIndex: TypeDefinitionIndex
  domainSchemaHash?: string
  onTypeClick: (item: SkeletonItem) => void
}) {
  if (type === '') {
    return <>void</>
  }

  const parts: Array<React.ReactNode> = []
  const pattern = /[A-Za-z_][A-Za-z0-9_.]*/g
  let cursor = 0
  let match: RegExpExecArray | null

  while ((match = pattern.exec(type)) !== null) {
    const token = match[0]
    const start = match.index
    const definition = findTypeDefinition(typeIndex, token, domainSchemaHash)

    if (start > cursor) {
      parts.push(type.slice(cursor, start))
    }

    if (definition) {
      parts.push(
        <a
          key={`${token}:${start}`}
          href={skeletonItemHref(definition, 'data')}
          className="font-mono text-primary underline-offset-2 hover:underline"
          onClick={(event) => {
            if (shouldUseBrowserNavigation(event)) {
              return
            }
            event.preventDefault()
            onTypeClick(definition)
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

function actorBadges(
  actors: Array<SkeletonActorRef>,
  onActorClick: (skelName: string) => void,
  noneText: string,
) {
  const values = Array.isArray(actors) ? actors : []

  if (values.length === 0) {
    return <span className="text-xs text-muted-foreground">{noneText}</span>
  }
  return (
    <div className="flex flex-wrap gap-1">
      {values.map((actor) => (
        <a
          key={actor.skelName}
          href={skeletonActorHref(actor.skelName)}
          className={cn(
            badgeVariants({ variant: 'outline' }),
            'cursor-pointer hover:border-primary/40 hover:bg-primary/5 hover:text-primary',
          )}
          onClick={(event) => {
            if (shouldUseBrowserNavigation(event)) {
              return
            }
            event.preventDefault()
            onActorClick(actor.skelName)
          }}
        >
          {actor.name}
          {actor.via ? (
            <span className="border-l pl-1.5 text-muted-foreground">
              {actor.via}
            </span>
          ) : null}
        </a>
      ))}
    </div>
  )
}

function textBadges(values: Array<string>, noneText: string) {
  const items = Array.isArray(values) ? values : []
  if (items.length === 0) {
    return <span className="text-xs text-muted-foreground">{noneText}</span>
  }
  return (
    <div className="flex flex-wrap gap-1">
      {items.map((item) => (
        <Badge key={item} variant="secondary">
          {item}
        </Badge>
      ))}
    </div>
  )
}

function FunctionSignature({
  name,
  fields,
  resultType,
  domainSchemaHash,
  typeIndex,
  onTypeClick,
}: {
  fields: Array<SkeletonField>
  name: string
  resultType?: string
  domainSchemaHash?: string
  typeIndex: TypeDefinitionIndex
  onTypeClick: (item: SkeletonItem) => void
}) {
  const args = Array.isArray(fields) ? fields : []
  const hasResult = Boolean(
    resultType && resultType !== '' && resultType !== 'void',
  )

  return (
    <>
      {name}(
      {args.map((field, index) => (
        <React.Fragment key={`${field.name}:${index}`}>
          {index > 0 ? ', ' : null}
          {field.name}:{' '}
          <TypeText
            type={field.type || 'unknown'}
            typeIndex={typeIndex}
            domainSchemaHash={domainSchemaHash}
            onTypeClick={onTypeClick}
          />
        </React.Fragment>
      ))}
      )
      {hasResult ? (
        <>
          {' -> '}
          <TypeText
            type={resultType ?? ''}
            typeIndex={typeIndex}
            domainSchemaHash={domainSchemaHash}
            onTypeClick={onTypeClick}
          />
        </>
      ) : null}
    </>
  )
}

function FieldList({
  fields,
  domainSchemaHash,
  typeIndex,
  onTypeClick,
}: {
  fields: Array<SkeletonField>
  domainSchemaHash?: string
  typeIndex: TypeDefinitionIndex
  onTypeClick: (item: SkeletonItem) => void
}) {
  const values = Array.isArray(fields) ? fields : []
  const { t } = useLocale()

  if (values.length === 0) {
    return (
      <div className="text-sm text-muted-foreground">
        {t('skeleton.noArguments')}
      </div>
    )
  }

  return (
    <div className="grid gap-1.5">
      {values.map((field) => (
        <div
          key={`${field.name}:${field.type}`}
          className="grid gap-1 rounded-md border bg-muted/20 px-2.5 py-2"
        >
          <div className="flex flex-wrap items-center gap-2">
            <code className="text-xs font-medium">{field.name}</code>
            <Badge variant="secondary">
              <TypeText
                type={field.type || 'void'}
                typeIndex={typeIndex}
                domainSchemaHash={domainSchemaHash}
                onTypeClick={onTypeClick}
              />
            </Badge>
          </div>
          {field.description ? (
            <div className="text-xs text-muted-foreground">
              {field.description}
            </div>
          ) : null}
          {field.example ? (
            <code className="text-xs text-muted-foreground">
              example: {field.example}
            </code>
          ) : null}
        </div>
      ))}
    </div>
  )
}

function MethodOutput({
  method,
  domainSchemaHash,
  typeIndex,
  onTypeClick,
}: {
  method: SkeletonMethod
  domainSchemaHash?: string
  typeIndex: TypeDefinitionIndex
  onTypeClick: (item: SkeletonItem) => void
}) {
  if (!method.outputDescription && !method.outputExample) {
    return null
  }

  return (
    <div className="grid gap-1 rounded-md border bg-muted/20 px-2.5 py-2">
      <div className="flex flex-wrap items-center gap-2">
        <code className="text-xs font-medium">return</code>
        {method.resultType ? (
          <Badge variant="secondary">
            <TypeText
              type={method.resultType}
              typeIndex={typeIndex}
              domainSchemaHash={domainSchemaHash}
              onTypeClick={onTypeClick}
            />
          </Badge>
        ) : null}
      </div>
      {method.outputDescription ? (
        <div className="text-xs text-muted-foreground">
          {method.outputDescription}
        </div>
      ) : null}
      {method.outputExample ? (
        <code className="text-xs text-muted-foreground">
          example: {method.outputExample}
        </code>
      ) : null}
    </div>
  )
}

function PermExprView({ expr }: { expr: SkeletonPermExpr | null }) {
  if (!expr) {
    return <span className="text-sm text-muted-foreground">none</span>
  }

  if (expr.mode === 'code' && expr.code) {
    return <Badge variant="secondary">{expr.code}</Badge>
  }

  if (expr.mode === 'check' && expr.check) {
    return (
      <div className="grid gap-1 rounded-md border bg-muted/20 px-2.5 py-2">
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant="outline">check</Badge>
          <code className="text-xs font-medium">
            {expr.check.resourceSkelName}:{expr.check.actionName}:
            {expr.check.checkName}
          </code>
        </div>
        <code className="text-xs text-muted-foreground">
          {expr.check.serviceSkelName}.{expr.check.methodSkelName}
        </code>
        {expr.check.arguments.length > 0 ? (
          <div className="flex flex-wrap gap-1">
            {expr.check.arguments.map((argument) => (
              <Badge key={argument.name} variant="secondary">
                {argument.name}: {argument.jsonPath} · {argument.type}
              </Badge>
            ))}
          </div>
        ) : null}
      </div>
    )
  }

  return (
    <div className="grid gap-1.5 rounded-md border bg-muted/20 px-2.5 py-2">
      <div className="flex items-center gap-2">
        <Badge variant="outline">{expr.mode}</Badge>
      </div>
      <div className="grid gap-1.5 pl-3">
        {expr.children.map((child, index) => (
          <PermExprView key={`${child.mode}:${index}`} expr={child} />
        ))}
      </div>
    </div>
  )
}

function PermRequireBlock({ expr }: { expr: SkeletonPermExpr | null }) {
  if (!expr) {
    return null
  }

  return (
    <div className="grid gap-1.5">
      <div className="text-xs font-medium text-muted-foreground">require</div>
      <PermExprView expr={expr} />
    </div>
  )
}

function collectPermExprBadges(expr: SkeletonPermExpr | null): Array<string> {
  if (!expr) {
    return []
  }
  if (expr.mode === 'code' && expr.code) {
    return [expr.code]
  }
  if (expr.mode === 'check' && expr.check) {
    return [
      `${expr.check.resourceSkelName}:${expr.check.actionName}:${expr.check.checkName}`,
    ]
  }
  const childBadges = expr.children.flatMap((child) =>
    collectPermExprBadges(child),
  )
  return childBadges.length > 0 ? [expr.mode, ...childBadges] : [expr.mode]
}

function PermBadgeSummary({ expr }: { expr: SkeletonPermExpr | null }) {
  const badges = collectPermExprBadges(expr)
  if (badges.length === 0) {
    return null
  }

  return (
    <>
      {badges.map((badge, index) => (
        <Badge
          key={`${badge}:${index}`}
          variant={index === 0 && (badge === 'all' || badge === 'any') ? 'outline' : 'secondary'}
        >
          {badge}
        </Badge>
      ))}
    </>
  )
}

function TypeParameterList({
  typeParameters,
}: {
  typeParameters: Array<string>
}) {
  if (typeParameters.length === 0) {
    return null
  }

  return (
    <div className="grid gap-1.5">
      {typeParameters.map((typeParameter) => (
        <div
          key={typeParameter}
          className="grid gap-1 rounded-md border bg-muted/20 px-2.5 py-2"
        >
          <div className="flex flex-wrap items-center gap-2">
            <code className="text-xs font-medium">{typeParameter}</code>
            <Badge variant="secondary">typeParameter</Badge>
          </div>
        </div>
      ))}
    </div>
  )
}

function EnumItemList({ items }: { items: Array<SkeletonEnumItem> }) {
  const values = Array.isArray(items) ? items : []
  const { t } = useLocale()

  if (values.length === 0) {
    return (
      <div className="text-sm text-muted-foreground">
        {t('skeleton.noEnumItems')}
      </div>
    )
  }

  return (
    <div className="grid gap-1.5">
      {values.map((item) => (
        <div
          key={item.name}
          className="grid gap-1 rounded-md border px-2.5 py-2"
        >
          <code className="text-xs font-medium">{item.name}</code>
          {item.description ? (
            <div className="text-xs text-muted-foreground">
              {item.description}
            </div>
          ) : null}
        </div>
      ))}
    </div>
  )
}

function MethodList({
  methods,
  domainSchemaHash,
  typeIndex,
  onTypeClick,
}: {
  methods: Array<SkeletonMethod>
  domainSchemaHash?: string
  typeIndex: TypeDefinitionIndex
  onTypeClick: (item: SkeletonItem) => void
}) {
  const values = Array.isArray(methods) ? methods : []
  const { t } = useLocale()

  if (values.length === 0) {
    return (
      <div className="text-sm text-muted-foreground">
        {t('skeleton.noMethods')}
      </div>
    )
  }

  return (
    <div className="grid gap-2">
      {values.map((method) => (
        <div key={method.skelName} className="grid gap-2 rounded-md border p-3">
          <div className="flex min-w-0 flex-wrap items-center gap-2">
            <code className="font-mono text-sm font-medium">
              <FunctionSignature
                name={method.name}
                fields={method.arguments}
                resultType={method.resultType}
                typeIndex={typeIndex}
                domainSchemaHash={domainSchemaHash}
                onTypeClick={onTypeClick}
              />
            </code>
            {method.authMode ? (
              <Badge
                variant={method.authMode === 'noauth' ? 'secondary' : 'outline'}
              >
                {method.authMode}
              </Badge>
            ) : null}
            <PermBadgeSummary expr={method.require} />
          </div>
          <code className="text-xs text-muted-foreground">
            {method.skelName}
          </code>
          {method.description ? (
            <div className="text-sm text-muted-foreground">
              {method.description}
            </div>
          ) : null}
          <PermRequireBlock expr={method.require} />
          <FieldList
            fields={method.arguments}
            typeIndex={typeIndex}
            domainSchemaHash={domainSchemaHash}
            onTypeClick={onTypeClick}
          />
          <MethodOutput
            method={method}
            typeIndex={typeIndex}
            domainSchemaHash={domainSchemaHash}
            onTypeClick={onTypeClick}
          />
        </div>
      ))}
    </div>
  )
}

function ResourceCheckList({
  checks,
  emptyText,
  domainSchemaHash,
  typeIndex,
  onTypeClick,
}: {
  checks: Array<SkeletonResourceCheck>
  emptyText: string
  domainSchemaHash?: string
  typeIndex: TypeDefinitionIndex
  onTypeClick: (item: SkeletonItem) => void
}) {
  const values = Array.isArray(checks) ? checks : []

  if (values.length === 0) {
    return <div className="text-sm text-muted-foreground">{emptyText}</div>
  }

  return (
    <div className="grid gap-1.5">
      {values.map((check) => (
        <div key={check.name} className="grid gap-2 rounded-md border p-3">
          <code className="font-mono text-sm font-medium">
            <FunctionSignature
              name={check.methodName}
              fields={check.arguments}
              typeIndex={typeIndex}
              domainSchemaHash={domainSchemaHash}
              onTypeClick={onTypeClick}
            />
          </code>
          <code className="text-xs text-muted-foreground">
            {check.methodSkelName}
          </code>
        </div>
      ))}
    </div>
  )
}

function ResourceActionList({
  actions,
  domainSchemaHash,
  typeIndex,
  onTypeClick,
}: {
  actions: Array<SkeletonResourceAction>
  domainSchemaHash?: string
  typeIndex: TypeDefinitionIndex
  onTypeClick: (item: SkeletonItem) => void
}) {
  const values = Array.isArray(actions) ? actions : []
  const { t } = useLocale()

  if (values.length === 0) {
    return (
      <div className="text-sm text-muted-foreground">
        {t('skeleton.noResourceActions')}
      </div>
    )
  }

  return (
    <div className="grid gap-2">
      {values.map((action) => (
        <div key={action.name} className="grid gap-2 rounded-md border p-3">
          <div className="flex min-w-0 flex-wrap items-center gap-2">
            <code className="font-mono text-sm font-medium">
              {action.name}
            </code>
            <Badge variant="secondary">{action.permissionCode}</Badge>
          </div>
          {action.description ? (
            <div className="text-sm text-muted-foreground">
              {action.description}
            </div>
          ) : null}
          {action.checks.length > 0 ? (
            <div className="grid gap-1.5">
              <div className="text-xs font-medium text-muted-foreground">
                {t('skeleton.resourceChecks')}
              </div>
              <ResourceCheckList
                checks={action.checks}
                emptyText={t('skeleton.noResourceChecks')}
                typeIndex={typeIndex}
                domainSchemaHash={domainSchemaHash}
                onTypeClick={onTypeClick}
              />
            </div>
          ) : null}
        </div>
      ))}
    </div>
  )
}

function TriggerList({
  triggers,
  domainSchemaHash,
  typeIndex,
  onTypeClick,
}: {
  triggers: Array<SkeletonTrigger>
  domainSchemaHash?: string
  typeIndex: TypeDefinitionIndex
  onTypeClick: (item: SkeletonItem) => void
}) {
  const values = Array.isArray(triggers) ? triggers : []
  const { t } = useLocale()

  if (values.length === 0) {
    return (
      <div className="text-sm text-muted-foreground">
        {t('skeleton.noTriggers')}
      </div>
    )
  }

  return (
    <div className="grid gap-2">
      {values.map((trigger) => (
        <div
          key={trigger.skelName}
          className="grid gap-2 rounded-md border p-3"
        >
          <code className="font-mono text-sm font-medium">
            <FunctionSignature
              name={trigger.name}
              fields={trigger.arguments}
              typeIndex={typeIndex}
              domainSchemaHash={domainSchemaHash}
              onTypeClick={onTypeClick}
            />
          </code>
          <code className="text-xs text-muted-foreground">
            {trigger.skelName}
          </code>
          {trigger.description ? (
            <div className="text-sm text-muted-foreground">
              {trigger.description}
            </div>
          ) : null}
          <FieldList
            fields={trigger.arguments}
            typeIndex={typeIndex}
            domainSchemaHash={domainSchemaHash}
            onTypeClick={onTypeClick}
          />
        </div>
      ))}
    </div>
  )
}

function RelatedSkeletonList<T extends { name: string; skelName: string }>({
  items,
  emptyText,
  onItemClick,
  getItemHref,
}: {
  items: Array<T>
  emptyText: string
  onItemClick: (item: T) => void
  getItemHref: (item: T) => string
}) {
  if (items.length === 0) {
    return <div className="text-sm text-muted-foreground">{emptyText}</div>
  }

  return (
    <div className="grid gap-2">
      {items.map((item) => (
        <a
          key={item.skelName}
          href={getItemHref(item)}
          onClick={(event) => {
            if (shouldUseBrowserNavigation(event)) {
              return
            }
            event.preventDefault()
            onItemClick(item)
          }}
          className="grid gap-1 rounded-md border px-3 py-2.5 text-left transition-colors hover:border-primary/30 hover:bg-primary/[0.04]"
        >
          <span className="truncate text-sm font-medium">{item.name}</span>
          <span className="truncate font-mono text-xs text-muted-foreground">
            {item.skelName}
          </span>
        </a>
      ))}
    </div>
  )
}

function DetailSection({
  icon: Icon,
  title,
  children,
}: {
  icon: React.ComponentType<{ className?: string }>
  title: string
  children: React.ReactNode
}) {
  return (
    <div className="grid gap-2">
      <div className="flex items-center gap-2 text-xs font-medium text-muted-foreground">
        <Icon className="size-3.5" />
        {title}
      </div>
      {children}
    </div>
  )
}

function ActorSchemaLink<T extends { name: string; skelName: string }>({
  icon: Icon,
  title,
  hideTitle = false,
  item,
  emptyText,
  onItemClick,
  getItemHref,
}: {
  icon: React.ComponentType<{ className?: string }>
  title: string
  hideTitle?: boolean
  item: T | null
  emptyText: string
  onItemClick: (item: T) => void
  getItemHref: (item: T) => string
}) {
  return (
    <div className="grid min-w-0 gap-2">
      {hideTitle ? null : (
        <div className="flex items-center gap-2 text-xs font-medium text-muted-foreground">
          <Icon className="size-3.5" />
          {title}
        </div>
      )}
      {item ? (
        <a
          href={getItemHref(item)}
          onClick={(event) => {
            if (shouldUseBrowserNavigation(event)) {
              return
            }
            event.preventDefault()
            onItemClick(item)
          }}
          className="grid min-w-0 gap-1 rounded-md border px-3 py-2.5 text-left transition-colors hover:border-primary/30 hover:bg-primary/[0.04]"
        >
          <span className="truncate text-sm font-medium">{item.name}</span>
          <span className="truncate font-mono text-xs text-muted-foreground">
            {item.skelName}
          </span>
        </a>
      ) : (
        <div className="text-sm text-muted-foreground">{emptyText}</div>
      )}
    </div>
  )
}

export function SkeletonItemBadges({
  item,
  showVersion = true,
}: {
  item: SkeletonItem
  showVersion?: boolean
}) {
  const { t } = useLocale()
  const actorVias =
    'actorVias' in item && Array.isArray(item.actorVias)
      ? item.actorVias
      : []

  return (
    <div className="flex flex-wrap justify-end gap-1">
      {'pub' in item && item.pub ? (
        <Badge variant="outline">public</Badge>
      ) : null}
      {'authMode' in item && item.authMode ? (
        <Badge variant={item.authMode === 'noauth' ? 'secondary' : 'outline'}>
          {item.authMode}
        </Badge>
      ) : null}
      {'authEnabled' in item && item.authEnabled ? (
        <Badge variant="outline">auth</Badge>
      ) : null}
      {'permEnabled' in item && item.permEnabled ? (
        <Badge variant="outline">perm</Badge>
      ) : null}
      {'require' in item && item.require ? <Badge variant="outline">perm</Badge> : null}
      {'enum' in item && item.enum ? (
        <Badge variant="outline">Enum</Badge>
      ) : null}
      {'lifecycle' in item && item.lifecycle ? (
        <Badge variant="outline">{item.lifecycle}</Badge>
      ) : null}
      {showVersion && !item.isMain ? (
        <Badge
          variant="outline"
          className="border-amber-300 bg-amber-50 text-amber-700"
        >
          {t('skeleton.version')}
        </Badge>
      ) : showVersion && item.isMultiVersion ? (
        <Badge variant="outline">{t('version.multiple')}</Badge>
      ) : null}
      {actorVias.map((actorVia) => (
        <Badge key={actorVia} variant="secondary">
          {actorVia}
        </Badge>
      ))}
    </div>
  )
}

export function SkeletonItemDetails({
  item,
  kind,
  typeIndex,
  relatedServices,
  relatedWebs,
  onTypeClick,
  onActorClick,
  onServiceClick,
  onDataClick,
  onWebClick,
}: {
  item: SkeletonItem
  kind: SkeletonKind
  typeIndex: TypeDefinitionIndex
  relatedServices: Array<SkeletonServiceItem>
  relatedWebs: Array<SkeletonWebItem>
  onTypeClick: (item: SkeletonItem) => void
  onActorClick: (skelName: string, domainSchemaHash?: string) => void
  onServiceClick: (item: SkeletonServiceItem) => void
  onDataClick: (item: SkeletonData) => void
  onWebClick: (item: SkeletonWebItem) => void
}) {
  const { t } = useLocale()
  const isActor = kind === 'actors'
  const hasActors = 'actors' in item
  const hasMethods = kind === 'services' && 'methods' in item
  const hasServiceRequire = kind === 'services' && 'require' in item
  const isResource = kind === 'resources' && 'actions' in item
  const hasConfigFields = kind === 'configs' && 'fields' in item
  const hasTriggers = kind === 'tasks' && 'triggers' in item
  const hasFields = kind === 'events' && 'fields' in item
  const hasDataFields =
    kind === 'data' && 'enum' in item && 'fields' in item && !item.enum
  const dataTypeParameters =
    kind === 'data' && 'typeParameters' in item ? item.typeParameters : []
  const hasEnumItems =
    kind === 'data' && 'enum' in item && 'enumItems' in item && item.enum

  return (
    <div className="grid gap-4">
      {isActor ? (
        <>
          <div className="grid gap-2">
            <div className="text-xs font-medium text-muted-foreground">
              {t('skeleton.actorVia')}
            </div>
            {textBadges(
              'actorVias' in item ? item.actorVias : [],
              t('common.none'),
            )}
          </div>
          <DetailSection icon={KeyRound} title={t('skeleton.authGroup')}>
            <div className="grid gap-3 md:grid-cols-3">
              <ActorSchemaLink
                icon={KeyRound}
                title={t('skeleton.actorCredential')}
                hideTitle
                item={'credential' in item ? item.credential : null}
                emptyText={t('skeleton.noActorCredential')}
                onItemClick={onDataClick}
                getItemHref={(data) => skeletonItemHref(data, 'data')}
              />
              <ActorSchemaLink
                icon={Braces}
                title={t('skeleton.actorInfo')}
                hideTitle
                item={'info' in item ? item.info : null}
                emptyText={t('skeleton.noActorInfo')}
                onItemClick={onDataClick}
                getItemHref={(data) => skeletonItemHref(data, 'data')}
              />
              <ActorSchemaLink
                icon={Server}
                title={t('skeleton.actorAuthService')}
                hideTitle
                item={'authService' in item ? item.authService : null}
                emptyText={t('skeleton.noActorAuthService')}
                onItemClick={onServiceClick}
                getItemHref={(service) => skeletonItemHref(service, 'services')}
              />
            </div>
          </DetailSection>
          <DetailSection
            icon={ShieldCheck}
            title={t('skeleton.permissionGroup')}
          >
            <div className="grid gap-3">
              <ActorSchemaLink
                icon={ShieldCheck}
                title={t('skeleton.actorPermService')}
                hideTitle
                item={'permService' in item ? item.permService : null}
                emptyText={t('skeleton.noActorPermService')}
                onItemClick={onServiceClick}
                getItemHref={(service) => skeletonItemHref(service, 'services')}
              />
            </div>
          </DetailSection>
          <DetailSection
            icon={Server}
            title={t('skeleton.accessibleService')}
          >
            <RelatedSkeletonList
              items={relatedServices}
              emptyText={t('skeleton.noAccessibleService')}
              onItemClick={onServiceClick}
              getItemHref={(service) => skeletonItemHref(service, 'services')}
            />
          </DetailSection>
          <DetailSection
            icon={Globe2}
            title={t('skeleton.accessibleWeb')}
          >
            <RelatedSkeletonList
              items={relatedWebs}
              emptyText={t('skeleton.noAccessibleWeb')}
              onItemClick={onWebClick}
              getItemHref={(web) => skeletonItemHref(web, 'webs')}
            />
          </DetailSection>
        </>
      ) : null}
      {hasActors ? (
        <div className="grid gap-2">
          <div className="text-xs font-medium text-muted-foreground">
            {t('skeleton.accessibleActor')}
          </div>
          {actorBadges(
            item.actors,
            (skelName) => onActorClick(skelName),
            t('common.none'),
          )}
        </div>
      ) : null}
      {hasServiceRequire ? <PermRequireBlock expr={item.require} /> : null}
      {hasMethods ? (
        <div className="grid gap-2">
          <div className="text-xs font-medium text-muted-foreground">
            {t('skeleton.methods')}
          </div>
          <MethodList
            methods={item.methods}
            typeIndex={typeIndex}
            domainSchemaHash={item.domainSchemaHash}
            onTypeClick={onTypeClick}
          />
        </div>
      ) : null}
      {isResource ? (
        <>
          <div className="grid gap-3">
            <ActorSchemaLink
              icon={Server}
              title={t('skeleton.resourceCheckService')}
              item={item.checkService}
              emptyText={t('skeleton.noResourceCheckService')}
              onItemClick={onServiceClick}
              getItemHref={(service) => skeletonItemHref(service, 'services')}
            />
          </div>
          <div className="grid gap-2">
            <div className="flex items-center gap-2 text-xs font-medium text-muted-foreground">
              <ShieldCheck className="size-3.5" />
              {t('skeleton.resourceChecks')}
            </div>
            <ResourceCheckList
              checks={item.checks}
              emptyText={t('skeleton.noResourceChecks')}
              typeIndex={typeIndex}
              domainSchemaHash={item.domainSchemaHash}
              onTypeClick={onTypeClick}
            />
          </div>
          <div className="grid gap-2">
            <div className="text-xs font-medium text-muted-foreground">
              {t('skeleton.resourceActions')}
            </div>
            <ResourceActionList
              actions={item.actions}
              typeIndex={typeIndex}
              domainSchemaHash={item.domainSchemaHash}
              onTypeClick={onTypeClick}
            />
          </div>
        </>
      ) : null}
      {hasConfigFields ? (
        <div className="grid gap-2">
          <div className="text-xs font-medium text-muted-foreground">
            {t('common.fields')}
          </div>
          <FieldList
            fields={item.fields}
            typeIndex={typeIndex}
            domainSchemaHash={item.domainSchemaHash}
            onTypeClick={onTypeClick}
          />
        </div>
      ) : null}
      {hasTriggers ? (
        <div className="grid gap-2">
          <div className="text-xs font-medium text-muted-foreground">
            {t('skeleton.triggers')}
          </div>
          <TriggerList
            triggers={item.triggers}
            typeIndex={typeIndex}
            domainSchemaHash={item.domainSchemaHash}
            onTypeClick={onTypeClick}
          />
        </div>
      ) : null}
      {hasFields ? (
        <div className="grid gap-2">
          <div className="text-xs font-medium text-muted-foreground">
            {t('common.fields')}
          </div>
          <FieldList
            fields={item.fields}
            typeIndex={typeIndex}
            domainSchemaHash={item.domainSchemaHash}
            onTypeClick={onTypeClick}
          />
        </div>
      ) : null}
      {hasDataFields ? (
        <div className="grid gap-2">
          {dataTypeParameters.length > 0 ? (
            <div className="grid gap-1.5">
              <div className="text-xs font-medium text-muted-foreground">
                {t('skeleton.typeParameters')}
              </div>
              <TypeParameterList typeParameters={dataTypeParameters} />
            </div>
          ) : null}
          <div className="text-xs font-medium text-muted-foreground">
            {t('common.fields')}
          </div>
          <FieldList
            fields={item.fields}
            typeIndex={typeIndex}
            domainSchemaHash={item.domainSchemaHash}
            onTypeClick={onTypeClick}
          />
        </div>
      ) : null}
      {hasEnumItems ? (
        <div className="grid gap-2">
          <div className="text-xs font-medium text-muted-foreground">
            {t('skeleton.enumItems')}
          </div>
          <EnumItemList items={item.enumItems} />
        </div>
      ) : null}
      {!hasActors &&
      !isActor &&
      !hasMethods &&
      !hasServiceRequire &&
      !isResource &&
      !hasConfigFields &&
      !hasTriggers &&
      !hasFields &&
      !hasDataFields &&
      !hasEnumItems ? (
        <div className="flex h-40 items-center justify-center rounded-lg border text-sm text-muted-foreground">
          {t('skeleton.noDetails')}
        </div>
      ) : null}
    </div>
  )
}
