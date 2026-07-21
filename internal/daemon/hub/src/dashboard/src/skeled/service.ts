import type {
  VrpcClient,
  VrpcRequestOptions,
} from '@yorun-ai/vrpc';
import {
  AppConfigServiceSpec,
  AppStatusServiceSpec,
  EventDebugServiceSpec,
  MaintenanceServiceSpec,
  PortalCertServiceSpec,
  PortalEntryServiceSpec,
  PortalRuleServiceSpec,
  PortalSiteServiceSpec,
  ServiceDebugServiceSpec,
  SkeletonServiceSpec,
  TaskDebugServiceSpec,
} from './spec';
import type {
  AppConfigItem,
  AppConfigUpdate,
  AppConfigCreation,
  AppStatusView,
  EventDebugEventItem,
  EventDebugDefaultEmitRequest,
  EventDebugEmitRequest,
  SeedPreview,
  SeedItemSelection,
  PortalCert,
  PortalCertCreation,
  PortalCertUpdate,
  PortalEntry,
  PortalEntryAccessUpdate,
  PortalRule,
  PortalRuleCreation,
  PortalRuleUpdate,
  PortalDashboardAccess,
  PortalSite,
  PortalSiteOptions,
  PortalSiteCreation,
  PortalSiteUpdate,
  ServiceDebugAppInstance,
  ServiceDebugServiceItem,
  ServiceDebugMethodItem,
  ServiceDebugDefaultInvokeRequest,
  ServiceDebugInvokeResponse,
  ServiceDebugInvokeRequest,
  SkeletonDomain,
  SkeletonActorItem,
  SkeletonServiceItem,
  SkeletonResourceItem,
  SkeletonWebItem,
  SkeletonTask,
  SkeletonEventItem,
  SkeletonData,
  SkeletonConfigItem,
  TaskDebugTaskItem,
  TaskDebugTriggerItem,
  TaskDebugDefaultLaunchRequest,
  TaskDebugLaunchRequest,
} from './data';
/**
 * Hub's application configuration service, called by Client
 */
export function createAppConfigService(client: VrpcClient) {
  return {
    /**
     * List configuration items。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns Array<AppConfigItem> - Configuration item list
     */
    list(
      params: null,
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<Array<AppConfigItem>>({
        serviceName: AppConfigServiceSpec.serviceName,
        methodName: AppConfigServiceSpec.methods.list,
        params,
        options,
      });
    },
    /**
     * Read configuration。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns AppConfigItem - Configuration items
     */
    get(
      params: {
        id: number;
      },
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<AppConfigItem>({
        serviceName: AppConfigServiceSpec.serviceName,
        methodName: AppConfigServiceSpec.methods.get,
        params,
        options,
      });
    },
    /**
     * Modify configuration。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns AppConfigItem - Configuration items
     */
    update(
      params: {
        id: number;
        update: AppConfigUpdate;
      },
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<AppConfigItem>({
        serviceName: AppConfigServiceSpec.serviceName,
        methodName: AppConfigServiceSpec.methods.update,
        params,
        options,
      });
    },
    /**
     * Create configuration。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns AppConfigItem - Configuration items
     */
    create(
      params: {
        creation: AppConfigCreation;
      },
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<AppConfigItem>({
        serviceName: AppConfigServiceSpec.serviceName,
        methodName: AppConfigServiceSpec.methods.create,
        params,
        options,
      });
    },
    /**
     * Delete unused configuration。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns boolean - Whether deletion succeeded
     */
    remove(
      params: {
        id: number;
      },
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<boolean>({
        serviceName: AppConfigServiceSpec.serviceName,
        methodName: AppConfigServiceSpec.methods.remove,
        params,
        options,
      });
    },
  };
}
/**
 * Hub Dashboard's application status service
 */
export function createAppStatusService(client: VrpcClient) {
  return {
    /**
     * List application instance statuses currently stored in Redis。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns Array<AppStatusView> -
     */
    list(
      params: null,
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<Array<AppStatusView>>({
        serviceName: AppStatusServiceSpec.serviceName,
        methodName: AppStatusServiceSpec.methods.list,
        params,
        options,
      });
    },
  };
}
/**
 * Hub Dashboard Event Debugging Service
 */
export function createEventDebugService(client: VrpcClient) {
  return {
    /**
     * List the events monitored by the application instance。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns Array<EventDebugEventItem> -
     */
    listEvents(
      params: null,
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<Array<EventDebugEventItem>>({
        serviceName: EventDebugServiceSpec.serviceName,
        methodName: EventDebugServiceSpec.methods.listEvents,
        params,
        options,
      });
    },
    /**
     * Generate a default Event send request。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns EventDebugDefaultEmitRequest -
     */
    buildDefaultEmitRequest(
      params: {
        eventSkelName: string;
        schemaHash: string;
      },
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<EventDebugDefaultEmitRequest>({
        serviceName: EventDebugServiceSpec.serviceName,
        methodName: EventDebugServiceSpec.methods.buildDefaultEmitRequest,
        params,
        options,
      });
    },
    /**
     * Send Event。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     */
    emitEvent(
      params: {
        request: EventDebugEmitRequest;
      },
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<void>({
        serviceName: EventDebugServiceSpec.serviceName,
        methodName: EventDebugServiceSpec.methods.emitEvent,
        params,
        options,
      });
    },
  };
}
/**
 * Hub maintenance service
 */
export function createMaintenanceService(client: VrpcClient) {
  return {
    /**
     * Preview Seed YAML differences。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns SeedPreview - Seed preview
     */
    previewSeedYaml(
      params: {
        content: string;
      },
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<SeedPreview>({
        serviceName: MaintenanceServiceSpec.serviceName,
        methodName: MaintenanceServiceSpec.methods.previewSeedYaml,
        params,
        options,
      });
    },
    /**
     * Apply Seed YAML entity updates。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns SeedPreview - Updated Seed preview
     */
    applySeedYaml(
      params: {
        content: string;
        selections: Array<SeedItemSelection>;
      },
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<SeedPreview>({
        serviceName: MaintenanceServiceSpec.serviceName,
        methodName: MaintenanceServiceSpec.methods.applySeedYaml,
        params,
        options,
      });
    },
  };
}
/**
 * Hub's Portal site certificate service, called by the Portal management client
 */
export function createPortalCertService(client: VrpcClient) {
  return {
    /**
     * List Portal site certificates。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns Array<PortalCert> - Portal site certificate list
     */
    list(
      params: null,
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<Array<PortalCert>>({
        serviceName: PortalCertServiceSpec.serviceName,
        methodName: PortalCertServiceSpec.methods.list,
        params,
        options,
      });
    },
    /**
     * Read the Portal site certificate。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns PortalCert - Portal site certificate
     */
    get(
      params: {
        id: number;
      },
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<PortalCert>({
        serviceName: PortalCertServiceSpec.serviceName,
        methodName: PortalCertServiceSpec.methods.get,
        params,
        options,
      });
    },
    /**
     * Create Portal site certificate。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns PortalCert - Portal site certificate
     */
    create(
      params: {
        creation: PortalCertCreation;
      },
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<PortalCert>({
        serviceName: PortalCertServiceSpec.serviceName,
        methodName: PortalCertServiceSpec.methods.create,
        params,
        options,
      });
    },
    /**
     * Modify Portal site certificate。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns PortalCert - Portal site certificate
     */
    update(
      params: {
        id: number;
        update: PortalCertUpdate;
      },
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<PortalCert>({
        serviceName: PortalCertServiceSpec.serviceName,
        methodName: PortalCertServiceSpec.methods.update,
        params,
        options,
      });
    },
    /**
     * Delete Portal site certificate。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     */
    remove(
      params: {
        id: number;
      },
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<void>({
        serviceName: PortalCertServiceSpec.serviceName,
        methodName: PortalCertServiceSpec.methods.remove,
        params,
        options,
      });
    },
  };
}
/**
 * Hub's Portal access entry service, called by the Portal management client
 */
export function createPortalEntryService(client: VrpcClient) {
  return {
    /**
     * List Portal access entries。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns Array<PortalEntry> - Portal access entry list
     */
    list(
      params: null,
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<Array<PortalEntry>>({
        serviceName: PortalEntryServiceSpec.serviceName,
        methodName: PortalEntryServiceSpec.methods.list,
        params,
        options,
      });
    },
    /**
     * Modify Portal access configuration。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns PortalEntry - Portal access entry
     */
    updateAccess(
      params: {
        scheme: string;
        host: string;
        port: number;
        update: PortalEntryAccessUpdate;
      },
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<PortalEntry>({
        serviceName: PortalEntryServiceSpec.serviceName,
        methodName: PortalEntryServiceSpec.methods.updateAccess,
        params,
        options,
      });
    },
  };
}
/**
 * Hub's Portal entry rule service, called by the Portal management client
 */
export function createPortalRuleService(client: VrpcClient) {
  return {
    /**
     * List Portal entry rules。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns Array<PortalRule> - Portal entry rule list
     */
    list(
      params: null,
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<Array<PortalRule>>({
        serviceName: PortalRuleServiceSpec.serviceName,
        methodName: PortalRuleServiceSpec.methods.list,
        params,
        options,
      });
    },
    /**
     * Read Portal entry rules。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns PortalRule - Portal entry rules
     */
    get(
      params: {
        id: number;
      },
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<PortalRule>({
        serviceName: PortalRuleServiceSpec.serviceName,
        methodName: PortalRuleServiceSpec.methods.get,
        params,
        options,
      });
    },
    /**
     * Create Portal entry rules。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns PortalRule - Portal entry rules
     */
    create(
      params: {
        creation: PortalRuleCreation;
      },
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<PortalRule>({
        serviceName: PortalRuleServiceSpec.serviceName,
        methodName: PortalRuleServiceSpec.methods.create,
        params,
        options,
      });
    },
    /**
     * Modify Portal entry rules。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns PortalRule - Portal entry rules
     */
    update(
      params: {
        id: number;
        update: PortalRuleUpdate;
      },
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<PortalRule>({
        serviceName: PortalRuleServiceSpec.serviceName,
        methodName: PortalRuleServiceSpec.methods.update,
        params,
        options,
      });
    },
    /**
     * Delete Portal entry rules。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     */
    remove(
      params: {
        id: number;
      },
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<void>({
        serviceName: PortalRuleServiceSpec.serviceName,
        methodName: PortalRuleServiceSpec.methods.remove,
        params,
        options,
      });
    },
    /**
     * Get the Hub Dashboard access entry。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns PortalDashboardAccess - Hub Dashboard access entry
     */
    getDashboardAccess(
      params: null,
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<PortalDashboardAccess>({
        serviceName: PortalRuleServiceSpec.serviceName,
        methodName: PortalRuleServiceSpec.methods.getDashboardAccess,
        params,
        options,
      });
    },
    /**
     * Modify Hub Dashboard access entry。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns Array<PortalRule> - Hub Dashboard entry rules
     */
    updateDashboardAccess(
      params: {
        scheme: string;
        host: string;
        port: number;
        pathPrefix: string;
      },
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<Array<PortalRule>>({
        serviceName: PortalRuleServiceSpec.serviceName,
        methodName: PortalRuleServiceSpec.methods.updateDashboardAccess,
        params,
        options,
      });
    },
  };
}
/**
 * Hub's Portal target site service, called by the Portal management client
 */
export function createPortalSiteService(client: VrpcClient) {
  return {
    /**
     * List Portal target sites。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns Array<PortalSite> - Portal target site list
     */
    list(
      params: null,
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<Array<PortalSite>>({
        serviceName: PortalSiteServiceSpec.serviceName,
        methodName: PortalSiteServiceSpec.methods.list,
        params,
        options,
      });
    },
    /**
     * List Portal target site form options。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns PortalSiteOptions - Portal target site form options
     */
    listOptions(
      params: null,
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<PortalSiteOptions>({
        serviceName: PortalSiteServiceSpec.serviceName,
        methodName: PortalSiteServiceSpec.methods.listOptions,
        params,
        options,
      });
    },
    /**
     * Read the Portal target site。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns PortalSite - Portal target site
     */
    get(
      params: {
        id: number;
      },
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<PortalSite>({
        serviceName: PortalSiteServiceSpec.serviceName,
        methodName: PortalSiteServiceSpec.methods.get,
        params,
        options,
      });
    },
    /**
     * Create Portal target site。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns PortalSite - Portal target site
     */
    create(
      params: {
        creation: PortalSiteCreation;
      },
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<PortalSite>({
        serviceName: PortalSiteServiceSpec.serviceName,
        methodName: PortalSiteServiceSpec.methods.create,
        params,
        options,
      });
    },
    /**
     * Modify Portal target site。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns PortalSite - Portal target site
     */
    update(
      params: {
        id: number;
        update: PortalSiteUpdate;
      },
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<PortalSite>({
        serviceName: PortalSiteServiceSpec.serviceName,
        methodName: PortalSiteServiceSpec.methods.update,
        params,
        options,
      });
    },
    /**
     * Delete Portal target site。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     */
    remove(
      params: {
        id: number;
      },
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<void>({
        serviceName: PortalSiteServiceSpec.serviceName,
        methodName: PortalSiteServiceSpec.methods.remove,
        params,
        options,
      });
    },
  };
}
/**
 * Hub Dashboard Service debugging service
 */
export function createServiceDebugService(client: VrpcClient) {
  return {
    /**
     * List application instances。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns Array<ServiceDebugAppInstance> -
     */
    listAppInstances(
      params: null,
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<Array<ServiceDebugAppInstance>>({
        serviceName: ServiceDebugServiceSpec.serviceName,
        methodName: ServiceDebugServiceSpec.methods.listAppInstances,
        params,
        options,
      });
    },
    /**
     * List the services provided by the application instance。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns Array<ServiceDebugServiceItem> -
     */
    listServices(
      params: null,
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<Array<ServiceDebugServiceItem>>({
        serviceName: ServiceDebugServiceSpec.serviceName,
        methodName: ServiceDebugServiceSpec.methods.listServices,
        params,
        options,
      });
    },
    /**
     * List application instances that provide the specified service。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns Array<ServiceDebugAppInstance> -
     */
    listServiceAppInstances(
      params: {
        serviceSkelName: string;
        schemaHash: string;
      },
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<Array<ServiceDebugAppInstance>>({
        serviceName: ServiceDebugServiceSpec.serviceName,
        methodName: ServiceDebugServiceSpec.methods.listServiceAppInstances,
        params,
        options,
      });
    },
    /**
     * List Service methods。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns Array<ServiceDebugMethodItem> -
     */
    listMethods(
      params: {
        serviceSkelName: string;
        schemaHash: string;
      },
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<Array<ServiceDebugMethodItem>>({
        serviceName: ServiceDebugServiceSpec.serviceName,
        methodName: ServiceDebugServiceSpec.methods.listMethods,
        params,
        options,
      });
    },
    /**
     * Generate default Service call request。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns ServiceDebugDefaultInvokeRequest -
     */
    buildDefaultInvokeRequest(
      params: {
        serviceSkelName: string;
        schemaHash: string;
        methodSkelName: string;
      },
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<ServiceDebugDefaultInvokeRequest>({
        serviceName: ServiceDebugServiceSpec.serviceName,
        methodName: ServiceDebugServiceSpec.methods.buildDefaultInvokeRequest,
        params,
        options,
      });
    },
    /**
     * Call Service method。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns ServiceDebugInvokeResponse -
     */
    invokeService(
      params: {
        request: ServiceDebugInvokeRequest;
      },
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<ServiceDebugInvokeResponse>({
        serviceName: ServiceDebugServiceSpec.serviceName,
        methodName: ServiceDebugServiceSpec.methods.invokeService,
        params,
        options,
      });
    },
  };
}
/**
 * Hub's skeleton service, called by the Portal management client
 */
export function createSkeletonService(client: VrpcClient) {
  return {
    /**
     * List Domain skeleton。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns Array<SkeletonDomain> - Domain skeleton list
     */
    listDomains(
      params: null,
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<Array<SkeletonDomain>>({
        serviceName: SkeletonServiceSpec.serviceName,
        methodName: SkeletonServiceSpec.methods.listDomains,
        params,
        options,
      });
    },
    /**
     * List Actor Skeleton。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns Array<SkeletonActorItem> - Actor skeleton list
     */
    listActors(
      params: null,
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<Array<SkeletonActorItem>>({
        serviceName: SkeletonServiceSpec.serviceName,
        methodName: SkeletonServiceSpec.methods.listActors,
        params,
        options,
      });
    },
    /**
     * List Service skeleton。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns Array<SkeletonServiceItem> - Service skeleton list
     */
    listServices(
      params: null,
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<Array<SkeletonServiceItem>>({
        serviceName: SkeletonServiceSpec.serviceName,
        methodName: SkeletonServiceSpec.methods.listServices,
        params,
        options,
      });
    },
    /**
     * List Resource skeleton。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns Array<SkeletonResourceItem> - Resource skeleton list
     */
    listResources(
      params: null,
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<Array<SkeletonResourceItem>>({
        serviceName: SkeletonServiceSpec.serviceName,
        methodName: SkeletonServiceSpec.methods.listResources,
        params,
        options,
      });
    },
    /**
     * List Web Skeletons。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns Array<SkeletonWebItem> - Web skeleton list
     */
    listWebs(
      params: null,
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<Array<SkeletonWebItem>>({
        serviceName: SkeletonServiceSpec.serviceName,
        methodName: SkeletonServiceSpec.methods.listWebs,
        params,
        options,
      });
    },
    /**
     * List Task skeleton。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns Array<SkeletonTask> - Task skeleton list
     */
    listTasks(
      params: null,
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<Array<SkeletonTask>>({
        serviceName: SkeletonServiceSpec.serviceName,
        methodName: SkeletonServiceSpec.methods.listTasks,
        params,
        options,
      });
    },
    /**
     * List Event skeletons。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns Array<SkeletonEventItem> - Event skeleton list
     */
    listEvents(
      params: null,
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<Array<SkeletonEventItem>>({
        serviceName: SkeletonServiceSpec.serviceName,
        methodName: SkeletonServiceSpec.methods.listEvents,
        params,
        options,
      });
    },
    /**
     * List Data skeleton。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns Array<SkeletonData> - Data skeleton list, including Enum
     */
    listData(
      params: null,
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<Array<SkeletonData>>({
        serviceName: SkeletonServiceSpec.serviceName,
        methodName: SkeletonServiceSpec.methods.listData,
        params,
        options,
      });
    },
    /**
     * List Config skeleton。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns Array<SkeletonConfigItem> - Config skeleton list
     */
    listConfigs(
      params: null,
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<Array<SkeletonConfigItem>>({
        serviceName: SkeletonServiceSpec.serviceName,
        methodName: SkeletonServiceSpec.methods.listConfigs,
        params,
        options,
      });
    },
  };
}
/**
 * Hub Dashboard Task Debugging Service
 */
export function createTaskDebugService(client: VrpcClient) {
  return {
    /**
     * List the tasks provided by the application instance。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns Array<TaskDebugTaskItem> -
     */
    listTasks(
      params: null,
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<Array<TaskDebugTaskItem>>({
        serviceName: TaskDebugServiceSpec.serviceName,
        methodName: TaskDebugServiceSpec.methods.listTasks,
        params,
        options,
      });
    },
    /**
     * List Task triggers。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns Array<TaskDebugTriggerItem> -
     */
    listTriggers(
      params: {
        taskSkelName: string;
        schemaHash: string;
      },
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<Array<TaskDebugTriggerItem>>({
        serviceName: TaskDebugServiceSpec.serviceName,
        methodName: TaskDebugServiceSpec.methods.listTriggers,
        params,
        options,
      });
    },
    /**
     * Generate a default Task launch request。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     * @returns TaskDebugDefaultLaunchRequest -
     */
    buildDefaultLaunchRequest(
      params: {
        taskSkelName: string;
        schemaHash: string;
        triggerSkelName: string;
      },
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<TaskDebugDefaultLaunchRequest>({
        serviceName: TaskDebugServiceSpec.serviceName,
        methodName: TaskDebugServiceSpec.methods.buildDefaultLaunchRequest,
        params,
        options,
      });
    },
    /**
     * Initiate Task。
     * @param params - Request parameters, or null for methods without input
     * @param options - Optional invocation options
     */
    launchTask(
      params: {
        request: TaskDebugLaunchRequest;
      },
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<void>({
        serviceName: TaskDebugServiceSpec.serviceName,
        methodName: TaskDebugServiceSpec.methods.launchTask,
        params,
        options,
      });
    },
  };
}
