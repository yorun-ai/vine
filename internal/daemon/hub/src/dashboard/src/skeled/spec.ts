export const AppConfigServiceSpec = {
  serviceName: 'vine.hub.AppConfigService',
  methods: {
    list: 'list',
    get: 'get',
    update: 'update',
    create: 'create',
    remove: 'remove',
  },
} as const;
export const AppStatusServiceSpec = {
  serviceName: 'vine.hub.AppStatusService',
  methods: {
    list: 'list',
  },
} as const;
export const EventDebugServiceSpec = {
  serviceName: 'vine.hub.EventDebugService',
  methods: {
    listEvents: 'listEvents',
    buildDefaultEmitRequest: 'buildDefaultEmitRequest',
    emitEvent: 'emitEvent',
  },
} as const;
export const MaintenanceServiceSpec = {
  serviceName: 'vine.hub.MaintenanceService',
  methods: {
    previewSeedYaml: 'previewSeedYaml',
    applySeedYaml: 'applySeedYaml',
  },
} as const;
export const PortalCertServiceSpec = {
  serviceName: 'vine.hub.PortalCertService',
  methods: {
    list: 'list',
    get: 'get',
    create: 'create',
    update: 'update',
    remove: 'remove',
  },
} as const;
export const PortalEntryServiceSpec = {
  serviceName: 'vine.hub.PortalEntryService',
  methods: {
    list: 'list',
    updateAccess: 'updateAccess',
  },
} as const;
export const PortalRuleServiceSpec = {
  serviceName: 'vine.hub.PortalRuleService',
  methods: {
    list: 'list',
    get: 'get',
    create: 'create',
    update: 'update',
    remove: 'remove',
    getDashboardAccess: 'getDashboardAccess',
    updateDashboardAccess: 'updateDashboardAccess',
  },
} as const;
export const PortalSiteServiceSpec = {
  serviceName: 'vine.hub.PortalSiteService',
  methods: {
    list: 'list',
    listOptions: 'listOptions',
    get: 'get',
    create: 'create',
    update: 'update',
    remove: 'remove',
  },
} as const;
export const ServiceDebugServiceSpec = {
  serviceName: 'vine.hub.ServiceDebugService',
  methods: {
    listAppInstances: 'listAppInstances',
    listServices: 'listServices',
    listServiceAppInstances: 'listServiceAppInstances',
    listMethods: 'listMethods',
    buildDefaultInvokeRequest: 'buildDefaultInvokeRequest',
    invokeService: 'invokeService',
  },
} as const;
export const SkeletonServiceSpec = {
  serviceName: 'vine.hub.SkeletonService',
  methods: {
    listDomains: 'listDomains',
    listActors: 'listActors',
    listServices: 'listServices',
    listResources: 'listResources',
    listWebs: 'listWebs',
    listTasks: 'listTasks',
    listEvents: 'listEvents',
    listData: 'listData',
    listConfigs: 'listConfigs',
  },
} as const;
export const TaskDebugServiceSpec = {
  serviceName: 'vine.hub.TaskDebugService',
  methods: {
    listTasks: 'listTasks',
    listTriggers: 'listTriggers',
    buildDefaultLaunchRequest: 'buildDefaultLaunchRequest',
    launchTask: 'launchTask',
  },
} as const;
