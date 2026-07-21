import * as React from 'react'

import { cn, type TranslationKey } from './dictionaries/cn'
import { en } from './dictionaries/en'

export type Locale = 'cn' | 'en'
export type { TranslationKey }

const LOCALE_STORAGE_KEY = 'vinehub_locale'
const dictionaries = { cn, en } satisfies Record<
  Locale,
  Record<TranslationKey, string>
>

interface LocaleContextValue {
  locale: Locale
  setLocale: (locale: Locale) => void
  t: (key: TranslationKey) => string
  tText: (text: string) => string
}

const fallbackTextTranslations: Record<string, string> = {
  打开侧边栏: en['topbar.openSidebar'],
  展开侧边栏: en['topbar.expandSidebar'],
  收起侧边栏: en['topbar.collapseSidebar'],
  账号菜单: en['account.menu'],
  调整侧边栏宽度: en['sidebar.resize.label'],
  '拖拽调整侧边栏宽度，双击恢复默认': en['sidebar.resize.title'],
  '切换工作场景，当前为': en['sidebar.sceneSwitch.current'],
}

const LocaleContext = React.createContext<LocaleContextValue | null>(null)

function getInitialLocale(): Locale {
  if (typeof window === 'undefined') {
    return 'cn'
  }

  return window.localStorage.getItem(LOCALE_STORAGE_KEY) === 'en' ? 'en' : 'cn'
}

export function translate(locale: Locale, key: TranslationKey) {
  return dictionaries[locale][key]
}

export function translateText(locale: Locale, text: string) {
  const matchedKey = (Object.keys(cn) as Array<TranslationKey>).find(
    (key) => cn[key] === text || en[key] === text,
  )

  if (matchedKey) {
    return dictionaries[locale][matchedKey]
  }

  return locale === 'en' ? (fallbackTextTranslations[text] ?? text) : text
}

export function LocaleProvider({ children }: { children: React.ReactNode }) {
  const [locale, setLocaleState] = React.useState<Locale>(getInitialLocale)

  const setLocale = React.useCallback((nextLocale: Locale) => {
    setLocaleState(nextLocale)
    window.localStorage.setItem(LOCALE_STORAGE_KEY, nextLocale)
  }, [])

  const value = React.useMemo<LocaleContextValue>(
    () => ({
      locale,
      setLocale,
      t: (key) => translate(locale, key),
      tText: (text) => translateText(locale, text),
    }),
    [locale, setLocale],
  )

  return (
    <LocaleContext.Provider value={value}>{children}</LocaleContext.Provider>
  )
}

export function useLocale() {
  const value = React.useContext(LocaleContext)
  if (!value) {
    throw new Error('useLocale must be used inside LocaleProvider')
  }
  return value
}
