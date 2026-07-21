import {
  Activity,
  BadgeCheck,
  Boxes,
  Braces,
  CalendarClock,
  Compass,
  GitBranch,
  Globe2,
  LayoutDashboard,
  PanelsTopLeft,
  Radio,
  RefreshCw,
  Server,
  SlidersHorizontal,
  Terminal,
  Users,
  Wrench,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import type { TranslationKey } from '@/i18n'
import { cn as cnDictionary } from '@/i18n/dictionaries/cn'

export interface AppNavItem {
  id: string
  label: string
  labelKey: TranslationKey
  description: string
  descriptionKey: TranslationKey
  to: string
  icon: LucideIcon
}

export interface AppNavGroup {
  id: string
  title: string | null
  titleKey: TranslationKey | null
  items: Array<AppNavItem>
}

export interface AppScene {
  id: string
  label: string
  labelKey: TranslationKey
  description: string
  descriptionKey: TranslationKey
  category: string
  categoryKey: TranslationKey
  icon: LucideIcon
  defaultTo: string
  groups: Array<AppNavGroup>
}

export interface AppSceneCategory {
  titleKey: TranslationKey
  sceneIds: Array<string>
}

function createScene(
  scene: Omit<AppScene, 'groups' | 'label' | 'description' | 'category'> & {
    groups?: Array<AppNavGroup>
    groupTitleKey?: TranslationKey
    sidebarItems?: Array<AppNavItem>
  },
): AppScene {
  return {
    ...scene,
    label: cnDictionary[scene.labelKey],
    description: cnDictionary[scene.descriptionKey],
    category: cnDictionary[scene.categoryKey],
    groups: scene.groups ?? [
      {
        id: `${scene.id}-group`,
        title: cnDictionary[scene.groupTitleKey ?? scene.labelKey],
        titleKey: scene.groupTitleKey ?? scene.labelKey,
        items: scene.sidebarItems ?? [
          {
            id: scene.id,
            label: cnDictionary[scene.labelKey],
            labelKey: scene.labelKey,
            description: cnDictionary[scene.descriptionKey],
            descriptionKey: scene.descriptionKey,
            to: scene.defaultTo,
            icon: scene.icon,
          },
        ],
      },
    ],
  }
}

export const APP_SCENES: Array<AppScene> = [
  createScene({
    id: 'vine-hub',
    labelKey: 'scene.vineHub.label',
    descriptionKey: 'scene.vineHub.description',
    categoryKey: 'scene.category.vineHub',
    icon: LayoutDashboard,
    defaultTo: '/app/config',
    groups: [
      {
        id: 'app-group',
        titleKey: 'nav.group.app',
        title: cnDictionary['nav.group.app'],
        items: [
          {
            id: 'app-config',
            label: cnDictionary['nav.appConfig.label'],
            labelKey: 'nav.appConfig.label',
            description: cnDictionary['nav.appConfig.description'],
            descriptionKey: 'nav.appConfig.description',
            to: '/app/config',
            icon: PanelsTopLeft,
          },
        ],
      },
      {
        id: 'portal-group',
        titleKey: 'nav.group.portal',
        title: cnDictionary['nav.group.portal'],
        items: [
          {
            id: 'portal-entry',
            label: cnDictionary['nav.portalEntry.label'],
            labelKey: 'nav.portalEntry.label',
            description: cnDictionary['nav.portalEntry.description'],
            descriptionKey: 'nav.portalEntry.description',
            to: '/portal/entry',
            icon: Compass,
          },
          {
            id: 'portal-rule',
            label: cnDictionary['nav.portalRule.label'],
            labelKey: 'nav.portalRule.label',
            description: cnDictionary['nav.portalRule.description'],
            descriptionKey: 'nav.portalRule.description',
            to: '/portal/rule',
            icon: GitBranch,
          },
          {
            id: 'portal-site',
            label: cnDictionary['nav.portalSite.label'],
            labelKey: 'nav.portalSite.label',
            description: cnDictionary['nav.portalSite.description'],
            descriptionKey: 'nav.portalSite.description',
            to: '/portal/site',
            icon: Globe2,
          },
          {
            id: 'portal-cert',
            label: cnDictionary['nav.portalCert.label'],
            labelKey: 'nav.portalCert.label',
            description: cnDictionary['nav.portalCert.description'],
            descriptionKey: 'nav.portalCert.description',
            to: '/portal/cert',
            icon: BadgeCheck,
          },
        ],
      },
      {
        id: 'app-status-group',
        titleKey: 'nav.group.status',
        title: cnDictionary['nav.group.status'],
        items: [
          {
            id: 'app-status',
            label: cnDictionary['nav.appStatus.label'],
            labelKey: 'nav.appStatus.label',
            description: cnDictionary['nav.appStatus.description'],
            descriptionKey: 'nav.appStatus.description',
            to: '/status/app',
            icon: Activity,
          },
        ],
      },
      {
        id: 'debug-group',
        titleKey: 'nav.group.debug',
        title: cnDictionary['nav.group.debug'],
        items: [
          {
            id: 'debug-service-client',
            label: cnDictionary['nav.serviceClient.label'],
            labelKey: 'nav.serviceClient.label',
            description: cnDictionary['nav.serviceClient.description'],
            descriptionKey: 'nav.serviceClient.description',
            to: '/debug/service-client',
            icon: Terminal,
          },
          {
            id: 'debug-task-launcher',
            label: cnDictionary['nav.taskLauncher.label'],
            labelKey: 'nav.taskLauncher.label',
            description: cnDictionary['nav.taskLauncher.description'],
            descriptionKey: 'nav.taskLauncher.description',
            to: '/debug/task-launcher',
            icon: CalendarClock,
          },
          {
            id: 'debug-event-emitter',
            label: cnDictionary['nav.eventEmitter.label'],
            labelKey: 'nav.eventEmitter.label',
            description: cnDictionary['nav.eventEmitter.description'],
            descriptionKey: 'nav.eventEmitter.description',
            to: '/debug/event-emitter',
            icon: Radio,
          },
        ],
      },
      {
        id: 'skeleton-group',
        titleKey: 'nav.group.skeleton',
        title: cnDictionary['nav.group.skeleton'],
        items: [
          {
            id: 'skeleton-domain',
            label: cnDictionary['nav.skeletonDomain.label'],
            labelKey: 'nav.skeletonDomain.label',
            description: cnDictionary['nav.skeletonDomain.description'],
            descriptionKey: 'nav.skeletonDomain.description',
            to: '/skeleton/domain',
            icon: Boxes,
          },
          {
            id: 'skeleton-actor',
            label: cnDictionary['nav.skeletonActor.label'],
            labelKey: 'nav.skeletonActor.label',
            description: cnDictionary['nav.skeletonActor.description'],
            descriptionKey: 'nav.skeletonActor.description',
            to: '/skeleton/actor',
            icon: Users,
          },
          {
            id: 'skeleton-config',
            label: cnDictionary['nav.skeletonConfig.label'],
            labelKey: 'nav.skeletonConfig.label',
            description: cnDictionary['nav.skeletonConfig.description'],
            descriptionKey: 'nav.skeletonConfig.description',
            to: '/skeleton/config',
            icon: SlidersHorizontal,
          },
          {
            id: 'skeleton-resource',
            label: cnDictionary['nav.skeletonResource.label'],
            labelKey: 'nav.skeletonResource.label',
            description: cnDictionary['nav.skeletonResource.description'],
            descriptionKey: 'nav.skeletonResource.description',
            to: '/skeleton/resource',
            icon: BadgeCheck,
          },
          {
            id: 'skeleton-data',
            label: cnDictionary['nav.skeletonData.label'],
            labelKey: 'nav.skeletonData.label',
            description: cnDictionary['nav.skeletonData.description'],
            descriptionKey: 'nav.skeletonData.description',
            to: '/skeleton/data',
            icon: Braces,
          },
          {
            id: 'skeleton-service',
            label: cnDictionary['nav.skeletonService.label'],
            labelKey: 'nav.skeletonService.label',
            description: cnDictionary['nav.skeletonService.description'],
            descriptionKey: 'nav.skeletonService.description',
            to: '/skeleton/service',
            icon: Server,
          },
          {
            id: 'skeleton-task',
            label: cnDictionary['nav.skeletonTask.label'],
            labelKey: 'nav.skeletonTask.label',
            description: cnDictionary['nav.skeletonTask.description'],
            descriptionKey: 'nav.skeletonTask.description',
            to: '/skeleton/task',
            icon: CalendarClock,
          },
          {
            id: 'skeleton-event',
            label: cnDictionary['nav.skeletonEvent.label'],
            labelKey: 'nav.skeletonEvent.label',
            description: cnDictionary['nav.skeletonEvent.description'],
            descriptionKey: 'nav.skeletonEvent.description',
            to: '/skeleton/event',
            icon: Radio,
          },
          {
            id: 'skeleton-web',
            label: cnDictionary['nav.skeletonWeb.label'],
            labelKey: 'nav.skeletonWeb.label',
            description: cnDictionary['nav.skeletonWeb.description'],
            descriptionKey: 'nav.skeletonWeb.description',
            to: '/skeleton/web',
            icon: Globe2,
          },
        ],
      },
    ],
  }),
  createScene({
    id: 'settings',
    labelKey: 'scene.settings.label',
    descriptionKey: 'scene.settings.description',
    categoryKey: 'scene.category.vineHub',
    icon: Wrench,
    defaultTo: '/settings/dashboard-port',
    groups: [
      {
        id: 'settings-group',
        titleKey: 'nav.group.settings',
        title: cnDictionary['nav.group.settings'],
        items: [
          {
            id: 'settings-dashboard-port',
            label: cnDictionary['nav.dashboardPort.label'],
            labelKey: 'nav.dashboardPort.label',
            description: cnDictionary['nav.dashboardPort.description'],
            descriptionKey: 'nav.dashboardPort.description',
            to: '/settings/dashboard-port',
            icon: LayoutDashboard,
          },
        ],
      },
      {
        id: 'maintenance-group',
        titleKey: 'nav.group.maintenance',
        title: cnDictionary['nav.group.maintenance'],
        items: [
          {
            id: 'maintenance',
            label: cnDictionary['nav.dataUpdate.label'],
            labelKey: 'nav.dataUpdate.label',
            description: cnDictionary['nav.dataUpdate.description'],
            descriptionKey: 'nav.dataUpdate.description',
            to: '/maintenance/data-update',
            icon: RefreshCw,
          },
        ],
      },
    ],
  }),
]

export const APP_SCENE_CATEGORIES: Array<AppSceneCategory> = [
  {
    titleKey: 'scene.category.vineHub',
    sceneIds: ['vine-hub', 'settings'],
  },
]

export const MOCK_USER = {
  name: 'Vinekeeper',
  role: '',
  email: 'vine@yorun.ai',
}

export function getSceneById(sceneId: string) {
  return APP_SCENES.find((scene) => scene.id === sceneId) ?? null
}
