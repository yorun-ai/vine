import { APP_SCENE_CATEGORIES, APP_SCENES, getSceneById } from './nav-config'
import type { AppNavItem, AppScene } from './nav-config'
import { translate, type Locale } from '@/i18n'

export interface BreadcrumbItem {
  label: string
  href?: string | null
}

export type SidebarDisplayState = 'expanded' | 'collapsed'

export function normalizePath(pathname: string) {
  if (!pathname) {
    return '/'
  }

  if (pathname.length > 1 && pathname.endsWith('/')) {
    return pathname.slice(0, -1)
  }

  return pathname
}

function getMatchedPrefixLength(pathname: string, target: string) {
  const normalizedPathname = normalizePath(pathname)
  const normalizedTarget = normalizePath(target)

  if (normalizedPathname === normalizedTarget) {
    return normalizedTarget.length
  }

  if (
    normalizedTarget !== '/' &&
    normalizedPathname.startsWith(`${normalizedTarget}/`)
  ) {
    return normalizedTarget.length
  }

  return -1
}

function localizeNavItem(item: AppNavItem, locale: Locale): AppNavItem {
  return {
    ...item,
    label: translate(locale, item.labelKey),
    description: translate(locale, item.descriptionKey),
  }
}

function localizeScene(scene: AppScene, locale: Locale): AppScene {
  return {
    ...scene,
    label: translate(locale, scene.labelKey),
    description: translate(locale, scene.descriptionKey),
    category: translate(locale, scene.categoryKey),
    groups: scene.groups.map((group) => ({
      ...group,
      title: group.titleKey ? translate(locale, group.titleKey) : group.title,
      items: group.items.map((item) => localizeNavItem(item, locale)),
    })),
  }
}

function getActiveMatch(pathname: string, locale: Locale = 'cn') {
  let activeScene: AppScene | null = null
  let activeItem: AppNavItem | null = null
  let bestLength = -1

  for (const scene of APP_SCENES) {
    for (const group of scene.groups) {
      for (const item of group.items) {
        const matchLength = getMatchedPrefixLength(pathname, item.to)

        if (matchLength > bestLength) {
          bestLength = matchLength
          activeScene = scene
          activeItem = item
        }
      }
    }
  }

  return {
    activeScene: localizeScene(
      activeScene ?? getSceneById('vine-hub') ?? APP_SCENES[0],
      locale,
    ),
    activeItem: activeItem ? localizeNavItem(activeItem, locale) : activeItem,
  }
}

export function getActiveScene(pathname: string, locale: Locale = 'cn') {
  return getActiveMatch(pathname, locale).activeScene
}

export function getActiveNavItem(pathname: string, locale: Locale = 'cn') {
  return getActiveMatch(pathname, locale).activeItem
}

export function getBreadcrumbItems(
  pathname: string,
  sidebarState: SidebarDisplayState,
  locale: Locale = 'cn',
): Array<BreadcrumbItem> {
  const { activeItem, activeScene } = getActiveMatch(pathname, locale)

  if (!activeItem || activeItem.label === activeScene.label) {
    return [{ label: activeScene.label, href: activeScene.defaultTo }]
  }

  if (sidebarState === 'expanded') {
    return [{ label: activeItem.label, href: activeItem.to }]
  }

  return [
    { label: activeScene.label, href: activeScene.defaultTo },
    { label: activeItem.label, href: activeItem.to },
  ]
}

export function getCategorizedScenes(locale: Locale = 'cn') {
  return APP_SCENE_CATEGORIES.map((category) => ({
    title: translate(locale, category.titleKey),
    scenes: category.sceneIds
      .map((sceneId) => getSceneById(sceneId))
      .map((scene) => (scene ? localizeScene(scene, locale) : scene))
      .filter((scene): scene is AppScene => scene !== null),
  }))
}
