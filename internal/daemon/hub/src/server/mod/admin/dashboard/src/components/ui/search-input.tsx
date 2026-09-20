import * as React from 'react'
import { Search, X } from 'lucide-react'

import { useLocale } from '@/i18n'
import { cn } from '@/lib/utils'
import { Input } from './input'

type SearchInputProps = Omit<
  React.ComponentProps<typeof Input>,
  'value' | 'defaultValue' | 'onChange'
> & {
  value: string
  onValueChange: (value: string) => void
}

export function SearchInput({
  ref,
  value,
  onValueChange,
  className,
  disabled,
  readOnly,
  ...props
}: SearchInputProps) {
  const inputRef = React.useRef<HTMLInputElement>(null)
  const { t } = useLocale()
  React.useImperativeHandle(ref, () => inputRef.current!, [])

  return (
    <div className="relative">
      <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
      <Input
        {...props}
        ref={inputRef}
        value={value}
        disabled={disabled}
        readOnly={readOnly}
        className={cn('pl-8 pr-8', className)}
        onChange={(event) => onValueChange(event.target.value)}
      />
      {value.length > 0 && !disabled && !readOnly ? (
        <button
          type="button"
          aria-label={t('common.clearSearch')}
          className="absolute top-1/2 right-1 flex size-6 -translate-y-1/2 cursor-pointer items-center justify-center rounded text-muted-foreground hover:bg-accent hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          onPointerDown={(event) => event.preventDefault()}
          onKeyDown={(event) => event.stopPropagation()}
          onClick={(event) => {
            event.stopPropagation()
            onValueChange('')
            inputRef.current?.focus()
          }}
        >
          <X className="size-3.5" aria-hidden="true" />
        </button>
      ) : null}
    </div>
  )
}
