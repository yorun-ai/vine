/**
 * Portal site CORS mode。
 */
export type PortalCorsMode =
  | "UNSPECIFIED"
  | "DISABLED"    // Disable CORS。
  | "SAME_DOMAIN" // Allow origins in the same domain as the entry rule。
  | "STRICT"      // Only allow Origins in the configuration list。
;
/**
 * Portal target site type。
 */
export type PortalSiteType =
  | "UNSPECIFIED"
  | "RPCGW"       // Rpc gateway。
  | "WEBGW"       // Web gateway。
;
/**
 * Configuration creation parameters。
 */
export type AppConfigCreation = {
  /**
   * Configuration Skel name。
   */
  skelName: string;
  /**
   * Configuration JSON。
   */
  value:    string;
}
/**
 * Configuration items。
 */
export type AppConfigItem = {
  /**
   * Configuration ID。
   */
  id:        number;
  /**
   * Configuration key。
   */
  key:       string;
  /**
   * Configuration status。
   */
  status:    string;
  /**
   * Configuration lifecycle。
   */
  lifecycle: string;
  /**
   * Configuration JSON。
   */
  value:     string;
  /**
   * Configuration schema。
   */
  schema:    AppConfigSchema | null;
}
/**
 * Configuration schema items。
 */
export type AppConfigSchema = {
  /**
   * Configuration Skel name。
   */
  skelName:    string;
  /**
   * Configuration name。
   */
  name:        string;
  /**
   * Configuration description。
   */
  description: string | null;
  /**
   * Configuration lifecycle。
   */
  lifecycle:   string;
  /**
   * Configuration field list。
   */
  fields:      Array<AppConfigSchemaField>;
}
/**
 * Configuration schema enumeration options。
 */
export type AppConfigSchemaEnumItem = {
  /**
   * Enum option name。
   */
  name:        string;
  /**
   * Enumeration options description。
   */
  description: string | null;
}
/**
 * Configuration schema fields。
 */
export type AppConfigSchemaField = {
  /**
   * Field name。
   */
  name:        string;
  /**
   * Field type。
   */
  type:        string;
  /**
   * Field description。
   */
  description: string | null;
  /**
   * Enumeration options list。
   */
  enumItems:   Array<AppConfigSchemaEnumItem>;
}
/**
 * Configuration update parameters。
 */
export type AppConfigUpdate = {
  /**
   * Configuration JSON。
   */
  value: string | null;
}
/**
 * Link application instance information registered with Hub。
 */
export type AppRegistration = {
  /**
   * Application name。
   */
  name:            string;
  /**
   * Application instance ID。
   */
  instanceId:      string;
  /**
   * Application version。
   */
  version:         string;
  /**
   * Application access address (e.g. "http://10.1.2.3:23001")。
   */
  endpoint:        string;
  /**
   * List of Rpc service processing capabilities provided by the application。
   */
  serviceHandlers: Array<ServiceHandlerRegistration>;
  /**
   * List of web processing capabilities provided by the application。
   */
  webHandlers:     Array<WebHandlerRegistration>;
  /**
   * List of event listening capabilities provided by the application。
   */
  eventListeners:  Array<EventListenerRegistration>;
  /**
   * List of task execution capabilities provided by the application。
   */
  taskRunners:     Array<TaskRunnerRegistration>;
  /**
   * List of all DomainSchemas registered by the application。
   */
  domainSchemas:   Array<string>;
}
/**
 * Application instance status information, used for heartbeat refresh。
 */
export type AppStatus = {
  /**
   * Application name。
   */
  name:       string;
  /**
   * Application instance ID。
   */
  instanceId: string;
}
/**
 * Application instance status view for Dashboard display。
 */
export type AppStatusView = {
  /**
   * Application name。
   */
  name:            string;
  /**
   * Application instance ID。
   */
  instanceId:      string;
  /**
   * Application version。
   */
  version:         string;
  /**
   * Application access address。
   */
  endpoint:        string;
  /**
   * List of Rpc service processing capabilities provided by the application。
   */
  serviceHandlers: Array<ServiceHandlerRegistration>;
  /**
   * List of web processing capabilities provided by the application。
   */
  webHandlers:     Array<WebHandlerRegistration>;
  /**
   * List of event listening capabilities provided by the application。
   */
  eventListeners:  Array<EventListenerRegistration>;
  /**
   * List of task execution capabilities provided by the application。
   */
  taskRunners:     Array<TaskRunnerRegistration>;
}
/**
 * Default Event Debug send request。
 */
export type EventDebugDefaultEmitRequest = {
  /**
   * Trace ID。
   */
  traceId:   string;
  /**
   * Span ID。
   */
  spanId:    string;
  /**
   * Default event JSON。
   */
  eventJson: string;
}
/**
 * Event Debug send request。
 */
export type EventDebugEmitRequest = {
  /**
   * Event Skel name。
   */
  eventSkelName: string;
  /**
   * Event schema hash。
   */
  schemaHash:    string;
  /**
   * Event JSON。
   */
  eventJson:     string;
  /**
   * Trace ID。
   */
  traceId:       string | null;
  /**
   * Span ID。
   */
  spanId:        string | null;
}
/**
 * Event called by Event Debug。
 */
export type EventDebugEventItem = {
  /**
   * Event name。
   */
  name:          string;
  /**
   * Event Skel name。
   */
  eventSkelName: string;
  /**
   * Event schema hash。
   */
  schemaHash:    string;
  /**
   * Event description。
   */
  description:   string | null;
  /**
   * Field list。
   */
  fields:        Array<SkeletonField>;
}
/**
 * Event listening capability registration information provided by the application。
 */
export type EventListenerRegistration = {
  /**
   * Event Skel name。
   */
  eventSkelName: string;
  /**
   * Event schema hash。
   */
  schemaHash:    string;
  /**
   * Execution timeout, in milliseconds。
   */
  timeoutMs:     number;
  /**
   * Maximum concurrency。
   */
  concurrency:   number;
  /**
   * Whether to disallow retrying after failure。
   */
  noRetry:       boolean;
}
/**
 * Hub information。
 */
export type Info = {
  /**
   * API service port。
   */
  apiPort:    number;
  /**
   * Redis service port。
   */
  redisPort:  number;
  /**
   * NATS service port。
   */
  natsPort:   number;
  /**
   * Standalone MQ service address。
   */
  mqEndpoint: string;
}
/**
 * Portal site certificate。
 */
export type PortalCert = {
  /**
   * Certificate ID。
   */
  id:                   number;
  /**
   * Certificate name。
   */
  name:                 string;
  /**
   * Certificate issuer。
   */
  issuer:               string;
  /**
   * Certificate domain name。
   */
  domains:              Array<string>;
  /**
   * Certificate Base64。
   */
  publicKeyBase64:      string;
  /**
   * Whether the private key has been configured。
   */
  privateKeyConfigured: boolean;
  /**
   * Validity start time。
   */
  validFrom:            string;
  /**
   * Validity end time。
   */
  validTo:              string;
}
/**
 * Portal site certificate creation parameters。
 */
export type PortalCertCreation = {
  /**
   * Certificate name。
   */
  name:             string;
  /**
   * Certificate Base64。
   */
  publicKeyBase64:  string;
  /**
   * Private key Base64。
   */
  privateKeyBase64: string;
}
/**
 * Portal site certificate update parameters。
 */
export type PortalCertUpdate = {
  /**
   * Certificate name。
   */
  name:             string | null;
  /**
   * Certificate Base64。
   */
  publicKeyBase64:  string | null;
  /**
   * Private key Base64。
   */
  privateKeyBase64: string | null;
}
/**
 * Portal site CORS configuration。
 */
export type PortalCors = {
  /**
   * CORS mode: DISABLED/SAME_DOMAIN/STRICT。
   */
  mode:           PortalCorsMode;
  /**
   * List of origins allowed in strict mode。
   */
  allowedOrigins: Array<string>;
}
/**
 * Hub Dashboard access entry。
 */
export type PortalDashboardAccess = {
  /**
   * Entry protocol。
   */
  scheme:     string;
  /**
   * Match Host, empty string means no restriction。
   */
  host:       string;
  /**
   * Entry port。
   */
  port:       number;
  /**
   * Match path prefix。
   */
  pathPrefix: string;
  /**
   * Whether to allow modification of Dashboard access entry。
   */
  canUpdate:  boolean;
}
/**
 * Portal access entry。
 */
export type PortalEntry = {
  /**
   * Entry name。
   */
  name:   string;
  /**
   * Entry protocol。
   */
  scheme: string;
  /**
   * Match Host, empty string means no restriction。
   */
  host:   string;
  /**
   * Entry port。
   */
  port:   number;
  /**
   * Entry rule list。
   */
  rules:  Array<PortalEntryRule>;
}
/**
 * Portal access entry configuration update parameters。
 */
export type PortalEntryAccessUpdate = {
  /**
   * Entry protocol。
   */
  scheme: string;
  /**
   * Match Host, empty string means no restriction。
   */
  host:   string;
  /**
   * Entry port。
   */
  port:   number;
}
/**
 * Portal access entry rules。
 */
export type PortalEntryRule = {
  /**
   * Entry rules。
   */
  rule: PortalRule;
  /**
   * Target site。
   */
  site: PortalSite | null;
}
/**
 * Portal entry rules。
 */
export type PortalRule = {
  /**
   * Rule ID。
   */
  id:                 number;
  /**
   * Rule name。
   */
  name:               string;
  /**
   * Matching protocol。
   */
  scheme:             string;
  /**
   * Match Host, empty string means no restriction。
   */
  host:               string;
  /**
   * Match port, 0 means no restriction。
   */
  port:               number;
  /**
   * Match path prefix, empty string means match all paths。
   */
  pathPrefix:         string;
  /**
   * Target type。
   */
  targetType:         string;
  /**
   * Site name。
   */
  siteName:           string;
  /**
   * Redirect Pattern。
   */
  redirectionPattern: string;
}
/**
 * Portal entry rule creation parameters。
 */
export type PortalRuleCreation = {
  /**
   * Rule name。
   */
  name:               string;
  /**
   * Matching protocol。
   */
  scheme:             string;
  /**
   * Match Host, empty string means no restriction。
   */
  host:               string;
  /**
   * Match port, 0 means no restriction。
   */
  port:               number;
  /**
   * Match path prefix, empty string means match all paths。
   */
  pathPrefix:         string;
  /**
   * Target type。
   */
  targetType:         string;
  /**
   * Site name。
   */
  siteName:           string;
  /**
   * Redirect Pattern。
   */
  redirectionPattern: string;
}
/**
 * Portal entry rule update parameters。
 */
export type PortalRuleUpdate = {
  /**
   * Rule name。
   */
  name:               string | null;
  /**
   * Matching protocol。
   */
  scheme:             string | null;
  /**
   * Match Host, empty string means no restriction。
   */
  host:               string | null;
  /**
   * Match port, 0 means no restriction。
   */
  port:               number | null;
  /**
   * Match path prefix, empty string means match all paths。
   */
  pathPrefix:         string | null;
  /**
   * Target type。
   */
  targetType:         string | null;
  /**
   * Site name。
   */
  siteName:           string | null;
  /**
   * Redirect Pattern。
   */
  redirectionPattern: string | null;
}
/**
 * Portal target site。
 */
export type PortalSite = {
  /**
   * Target site id。
   */
  id:            number;
  /**
   * Target site name。
   */
  name:          string;
  /**
   * Target site type。
   */
  type:          PortalSiteType;
  /**
   * Actor Skel name。
   */
  actorSkelName: string;
  /**
   * Actor access method。
   */
  actorVia:      string;
  /**
   * Rpc gateway service Skel name list。
   */
  rpcgwServices: Array<string>;
  /**
   * CORS configuration。
   */
  cors:          PortalCors | null;
  /**
   * Web name。
   */
  webName:       string;
}
/**
 * Portal target site Actor options。
 */
export type PortalSiteActorOption = {
  /**
   * Actor name。
   */
  name:      string;
  /**
   * Actor Skel name。
   */
  skelName:  string;
  /**
   * Actor access method list。
   */
  actorVias: Array<string>;
}
/**
 * Portal target site creation parameters。
 */
export type PortalSiteCreation = {
  /**
   * Target site name。
   */
  name:          string;
  /**
   * Target site type。
   */
  type:          PortalSiteType;
  /**
   * Actor Skel name。
   */
  actorSkelName: string;
  /**
   * Actor access method。
   */
  actorVia:      string;
  /**
   * CORS configuration。
   */
  cors:          PortalCors | null;
  /**
   * Web name。
   */
  webName:       string;
}
/**
 * Portal target site form options。
 */
export type PortalSiteOptions = {
  /**
   * Actor options。
   */
  actors:   Array<PortalSiteActorOption>;
  /**
   * Rpc service options。
   */
  services: Array<PortalSiteServiceOption>;
  /**
   * Web options。
   */
  webs:     Array<PortalSiteWebOption>;
}
/**
 * Portal target site service options。
 */
export type PortalSiteServiceOption = {
  /**
   * Service name。
   */
  name:           string;
  /**
   * Service Skel name。
   */
  skelName:       string;
  /**
   * Actor Skel name list。
   */
  actorSkelNames: Array<string>;
}
/**
 * Portal target site update parameters。
 */
export type PortalSiteUpdate = {
  /**
   * Target site name。
   */
  name:          string | null;
  /**
   * Target site type。
   */
  type:          PortalSiteType | null;
  /**
   * Actor Skel name。
   */
  actorSkelName: string | null;
  /**
   * Actor access method。
   */
  actorVia:      string | null;
  /**
   * CORS configuration。
   */
  cors:          PortalCors | null;
  /**
   * Web name。
   */
  webName:       string | null;
}
/**
 * Portal target site web options。
 */
export type PortalSiteWebOption = {
  /**
   * Web name。
   */
  name:           string;
  /**
   * Web Skel name。
   */
  skelName:       string;
  /**
   * Actor Skel name list。
   */
  actorSkelNames: Array<string>;
}
/**
 * Seed entity differences。
 */
export type SeedEntityDiff = {
  /**
   * Entity type。
   */
  kind:   string;
  /**
   * Entity name。
   */
  name:   string;
  /**
   * Whether the entity currently exists。
   */
  exists: boolean;
  /**
   * Field differences。
   */
  fields: Array<SeedFieldDiff>;
}
/**
 * Seed field differences。
 */
export type SeedFieldDiff = {
  /**
   * Field name。
   */
  name:         string;
  /**
   * Current value。
   */
  currentValue: string;
  /**
   * Seed value。
   */
  seedValue:    string;
  /**
   * Whether the values differ。
   */
  changed:      boolean;
}
/**
 * Seed entity selection。
 */
export type SeedItemSelection = {
  /**
   * Entity type。
   */
  kind: string;
  /**
   * Entity name。
   */
  name: string;
}
/**
 * Seed preview。
 */
export type SeedPreview = {
  /**
   * Entity differences。
   */
  items: Array<SeedEntityDiff>;
}
/**
 * Service Debug Actor options。
 */
export type ServiceDebugActorItem = {
  /**
   * Actor name。
   */
  name:          string;
  /**
   * Actor Skel name。
   */
  skelName:      string;
  /**
   * Actor Info Skel name。
   */
  infoSkelName:  string;
  /**
   * Default Actor Info JSON。
   */
  actorInfoJson: string;
}
/**
 * Application instance called by Service Debug。
 */
export type ServiceDebugAppInstance = {
  /**
   * Application name。
   */
  appName:       string;
  /**
   * Application instance ID。
   */
  appInstanceId: string;
  /**
   * Application version。
   */
  appVersion:    string;
  /**
   * Application access address。
   */
  endpoint:      string;
}
/**
 * Service Debug default call request。
 */
export type ServiceDebugDefaultInvokeRequest = {
  /**
   * Trace ID。
   */
  traceId:       string;
  /**
   * Span ID。
   */
  spanId:        string;
  /**
   * Actor options。
   */
  actors:        Array<ServiceDebugActorItem>;
  /**
   * Default Actor Skel name。
   */
  actorSkelName: string | null;
  /**
   * Default Actor Info JSON。
   */
  actorInfoJson: string;
  /**
   * Default request parameters JSON。
   */
  paramsJson:    string;
}
/**
 * Service Debug call request。
 */
export type ServiceDebugInvokeRequest = {
  /**
   * Application name。
   */
  appName:         string | null;
  /**
   * Application instance ID。
   */
  appInstanceId:   string | null;
  /**
   * Service Skel name。
   */
  serviceSkelName: string;
  /**
   * Service schema hash。
   */
  schemaHash:      string;
  /**
   * Method Skel name。
   */
  methodSkelName:  string;
  /**
   * Request parameters JSON。
   */
  paramsJson:      string;
  /**
   * Call timeout, in seconds。
   */
  timeoutSeconds:  number;
  /**
   * Trace ID。
   */
  traceId:         string | null;
  /**
   * Span ID。
   */
  spanId:          string | null;
  /**
   * Actor Skel name。
   */
  actorSkelName:   string | null;
  /**
   * Actor Info JSON。
   */
  actorInfoJson:   string;
}
/**
 * Service Debug call response。
 */
export type ServiceDebugInvokeResponse = {
  /**
   * HTTP status code。
   */
  httpStatus:  number;
  /**
   * Rpc status code。
   */
  rpcStatus:   string;
  /**
   * Response header JSON。
   */
  headersJson: string;
  /**
   * Response body JSON。
   */
  bodyJson:    string;
}
/**
 * Method called by Service Debug。
 */
export type ServiceDebugMethodItem = {
  /**
   * Method name。
   */
  name:              string;
  /**
   * Method Skel name。
   */
  skelName:          string;
  /**
   * Method description。
   */
  description:       string | null;
  /**
   * Input description。
   */
  inputDescription:  string | null;
  /**
   * Output description。
   */
  outputDescription: string | null;
  /**
   * Input example。
   */
  example:           string | null;
  /**
   * Output example。
   */
  outputExample:     string | null;
  /**
   * Parameter list。
   */
  arguments:         Array<SkeletonField>;
  /**
   * Return type。
   */
  resultType:        string;
}
/**
 * Service called by Service Debug。
 */
export type ServiceDebugServiceItem = {
  /**
   * Service Skel name。
   */
  serviceSkelName: string;
  /**
   * Service schema hash。
   */
  schemaHash:      string;
}
/**
 * Rpc service processing capability registration information provided by the application。
 */
export type ServiceHandlerRegistration = {
  /**
   * Service Skel name。
   */
  serviceSkelName: string;
  /**
   * Service schema hash。
   */
  schemaHash:      string;
  /**
   * Service agent access address。
   */
  endpoint:        string;
}
/**
 * SkeletonActor。
 */
export type SkeletonActorItem = {
  /**
   * Domain。
   */
  domain:           string;
  /**
   * Skeleton item hash。
   */
  schemaHash:       string;
  /**
   * Primary skeleton item hash。
   */
  mainSchemaHash:   string;
  /**
   * Whether there are multiple valid versions of the skeleton item。
   */
  isMultiVersion:   boolean;
  /**
   * Whether it is the main version of the skeleton item。
   */
  isMain:           boolean;
  /**
   * Owning DomainSchema hash。
   */
  domainSchemaHash: string;
  /**
   * Actor name。
   */
  name:             string;
  /**
   * Actor Skel name。
   */
  skelName:         string;
  /**
   * Actor description。
   */
  description:      string | null;
  /**
   * Actor access method list。
   */
  actorVias:        Array<string>;
  /**
   * Whether to enable authentication。
   */
  authEnabled:      boolean;
  /**
   * Authentication credentials。
   */
  credential:       SkeletonData | null;
  /**
   * Authentication information。
   */
  info:             SkeletonData | null;
  /**
   * Authentication services。
   */
  authService:      SkeletonServiceItem | null;
  /**
   * Whether to enable permissions。
   */
  permEnabled:      boolean;
  /**
   * Permission service。
   */
  permService:      SkeletonServiceItem | null;
  /**
   * Permission method。
   */
  permMethod:       SkeletonMethod | null;
  /**
   * Accessible Service List。
   */
  services:         Array<SkeletonServiceItem>;
  /**
   * Accessible web list。
   */
  webs:             Array<SkeletonWebItem>;
}
/**
 * Skeleton Actor Reference。
 */
export type SkeletonActorRef = {
  /**
   * Actor name。
   */
  name:     string;
  /**
   * Actor Skel name。
   */
  skelName: string;
  /**
   * Access method。
   */
  via:      string | null;
}
/**
 * SkeletonConfig。
 */
export type SkeletonConfigItem = {
  /**
   * Domain。
   */
  domain:           string;
  /**
   * Skeleton item hash。
   */
  schemaHash:       string;
  /**
   * Primary skeleton item hash。
   */
  mainSchemaHash:   string;
  /**
   * Whether there are multiple valid versions of the skeleton item。
   */
  isMultiVersion:   boolean;
  /**
   * Whether it is the main version of the skeleton item。
   */
  isMain:           boolean;
  /**
   * Owning DomainSchema hash。
   */
  domainSchemaHash: string;
  /**
   * Config name。
   */
  name:             string;
  /**
   * Config Skel name。
   */
  skelName:         string;
  /**
   * Config description。
   */
  description:      string | null;
  /**
   * Whether the item is public。
   */
  pub:              boolean;
  /**
   * Config lifecycle。
   */
  lifecycle:        string;
  /**
   * Field list。
   */
  fields:           Array<SkeletonField>;
}
/**
 * SkeletonData。
 */
export type SkeletonData = {
  /**
   * Domain。
   */
  domain:           string;
  /**
   * Skeleton item hash。
   */
  schemaHash:       string;
  /**
   * Primary skeleton item hash。
   */
  mainSchemaHash:   string;
  /**
   * Whether there are multiple valid versions of the skeleton item。
   */
  isMultiVersion:   boolean;
  /**
   * Whether it is the main version of the skeleton item。
   */
  isMain:           boolean;
  /**
   * Owning DomainSchema hash。
   */
  domainSchemaHash: string;
  /**
   * Data name。
   */
  name:             string;
  /**
   * Data Skel name。
   */
  skelName:         string;
  /**
   * Data description。
   */
  description:      string | null;
  /**
   * Whether it is Enum。
   */
  enum:             boolean;
  /**
   * Type parameter list。
   */
  typeParameters:   Array<string>;
  /**
   * Field list。
   */
  fields:           Array<SkeletonField>;
  /**
   * List of enumeration items。
   */
  enumItems:        Array<SkeletonEnumItem>;
}
/**
 * Domain skeleton version。
 */
export type SkeletonDomain = {
  /**
   * Domain name。
   */
  domain:         string;
  /**
   * DomainSchema hash。
   */
  schemaHash:     string;
  /**
   * Primary DomainSchema hash。
   */
  mainSchemaHash: string;
  /**
   * Whether multiple active versions exist。
   */
  isMultiVersion: boolean;
  /**
   * Whether this is the primary version。
   */
  isMain:         boolean;
  /**
   * Total number of skeleton items。
   */
  total:          number;
  /**
   * Actor list。
   */
  actors:         Array<SkeletonActorItem>;
  /**
   * Service list。
   */
  services:       Array<SkeletonServiceItem>;
  /**
   * Resource list。
   */
  resources:      Array<SkeletonResourceItem>;
  /**
   * Data list。
   */
  data:           Array<SkeletonData>;
  /**
   * Config list。
   */
  configs:        Array<SkeletonConfigItem>;
  /**
   * Web list。
   */
  webs:           Array<SkeletonWebItem>;
  /**
   * Task list。
   */
  tasks:          Array<SkeletonTask>;
  /**
   * Event list。
   */
  events:         Array<SkeletonEventItem>;
}
/**
 * Skeleton enumeration items。
 */
export type SkeletonEnumItem = {
  /**
   * Enumeration item name。
   */
  name:        string;
  /**
   * Enumeration item description。
   */
  description: string | null;
}
/**
 * Skeleton event。
 */
export type SkeletonEventItem = {
  /**
   * Domain。
   */
  domain:           string;
  /**
   * Skeleton item hash。
   */
  schemaHash:       string;
  /**
   * Primary skeleton item hash。
   */
  mainSchemaHash:   string;
  /**
   * Whether there are multiple valid versions of the skeleton item。
   */
  isMultiVersion:   boolean;
  /**
   * Whether it is the main version of the skeleton item。
   */
  isMain:           boolean;
  /**
   * Owning DomainSchema hash。
   */
  domainSchemaHash: string;
  /**
   * Event name。
   */
  name:             string;
  /**
   * Event Skel name。
   */
  skelName:         string;
  /**
   * Event description。
   */
  description:      string | null;
  /**
   * Whether the item is public。
   */
  pub:              boolean;
  /**
   * Field list。
   */
  fields:           Array<SkeletonField>;
}
/**
 * Skeleton field。
 */
export type SkeletonField = {
  /**
   * Field name。
   */
  name:        string;
  /**
   * Field type。
   */
  type:        string;
  /**
   * Field description。
   */
  description: string | null;
  /**
   * Field example。
   */
  example:     string | null;
}
/**
 * Skeleton method。
 */
export type SkeletonMethod = {
  /**
   * Method name。
   */
  name:              string;
  /**
   * Method Skel name。
   */
  skelName:          string;
  /**
   * Method description。
   */
  description:       string | null;
  /**
   * Input description。
   */
  inputDescription:  string | null;
  /**
   * Output description。
   */
  outputDescription: string | null;
  /**
   * Input example。
   */
  example:           string | null;
  /**
   * Authentication mode。
   */
  authMode:          string;
  /**
   * Permission requirements。
   */
  require:           SkeletonPermExpr | null;
  /**
   * Output example。
   */
  outputExample:     string | null;
  /**
   * Parameter list。
   */
  arguments:         Array<SkeletonField>;
  /**
   * Return type。
   */
  resultType:        string;
}
/**
 * Skeleton permission verification call。
 */
export type SkeletonPermCheck = {
  /**
   * Resource Skel name。
   */
  resourceSkelName: string;
  /**
   * Action name。
   */
  actionName:       string;
  /**
   * Check name。
   */
  checkName:        string;
  /**
   * Check Service Skel name。
   */
  serviceSkelName:  string;
  /**
   * Check Method Skel name。
   */
  methodSkelName:   string;
  /**
   * Parameter list。
   */
  arguments:        Array<SkeletonPermCheckArgument>;
}
/**
 * Skeleton permission verification parameters。
 */
export type SkeletonPermCheckArgument = {
  /**
   * Parameter name。
   */
  name:     string;
  /**
   * Parameter JSON path。
   */
  jsonPath: string;
  /**
   * Parameter type。
   */
  type:     string;
}
/**
 * Skeleton permission expression。
 */
export type SkeletonPermExpr = {
  /**
   * Permission expression pattern。
   */
  mode:     string;
  /**
   * Permission code。
   */
  code:     string | null;
  /**
   * Permission verification call。
   */
  check:    SkeletonPermCheck | null;
  /**
   * Subexpression。
   */
  children: Array<SkeletonPermExpr>;
}
/**
 * SkeletonResource Action。
 */
export type SkeletonResourceAction = {
  /**
   * Action name。
   */
  name:           string;
  /**
   * Permission code。
   */
  permissionCode: string;
  /**
   * Action description。
   */
  description:    string | null;
  /**
   * Check list。
   */
  checks:         Array<SkeletonResourceCheck>;
}
/**
 * SkeletonResource Check。
 */
export type SkeletonResourceCheck = {
  /**
   * Check name。
   */
  name:           string;
  /**
   * Check method name。
   */
  methodName:     string;
  /**
   * Check method Skel name。
   */
  methodSkelName: string;
  /**
   * Parameter list。
   */
  arguments:      Array<SkeletonField>;
}
/**
 * Skeleton Resource item。
 */
export type SkeletonResourceItem = {
  /**
   * Domain。
   */
  domain:           string;
  /**
   * Skeleton item hash。
   */
  schemaHash:       string;
  /**
   * Primary skeleton item hash。
   */
  mainSchemaHash:   string;
  /**
   * Whether there are multiple valid versions of the skeleton item。
   */
  isMultiVersion:   boolean;
  /**
   * Whether it is the main version of the skeleton item。
   */
  isMain:           boolean;
  /**
   * Owning DomainSchema hash。
   */
  domainSchemaHash: string;
  /**
   * Resource name。
   */
  name:             string;
  /**
   * Resource Skel name。
   */
  skelName:         string;
  /**
   * Resource description。
   */
  description:      string | null;
  /**
   * Resource level Check list。
   */
  checks:           Array<SkeletonResourceCheck>;
  /**
   * Action list。
   */
  actions:          Array<SkeletonResourceAction>;
  /**
   * Check service。
   */
  checkService:     SkeletonServiceItem | null;
}
/**
 * Skeleton service items。
 */
export type SkeletonServiceItem = {
  /**
   * Domain。
   */
  domain:           string;
  /**
   * Skeleton item hash。
   */
  schemaHash:       string;
  /**
   * Primary skeleton item hash。
   */
  mainSchemaHash:   string;
  /**
   * Whether there are multiple valid versions of the skeleton item。
   */
  isMultiVersion:   boolean;
  /**
   * Whether it is the main version of the skeleton item。
   */
  isMain:           boolean;
  /**
   * Owning DomainSchema hash。
   */
  domainSchemaHash: string;
  /**
   * Service name。
   */
  name:             string;
  /**
   * Service Skel name。
   */
  skelName:         string;
  /**
   * Service Description。
   */
  description:      string | null;
  /**
   * Whether the item is public。
   */
  pub:              boolean;
  /**
   * Authentication mode。
   */
  authMode:         string;
  /**
   * Permission requirements。
   */
  require:          SkeletonPermExpr | null;
  /**
   * Accessible Actor List。
   */
  actors:           Array<SkeletonActorRef>;
  /**
   * Method list。
   */
  methods:          Array<SkeletonMethod>;
}
/**
 * Skeleton task。
 */
export type SkeletonTask = {
  /**
   * Domain。
   */
  domain:           string;
  /**
   * Skeleton item hash。
   */
  schemaHash:       string;
  /**
   * Primary skeleton item hash。
   */
  mainSchemaHash:   string;
  /**
   * Whether there are multiple valid versions of the skeleton item。
   */
  isMultiVersion:   boolean;
  /**
   * Whether it is the main version of the skeleton item。
   */
  isMain:           boolean;
  /**
   * Owning DomainSchema hash。
   */
  domainSchemaHash: string;
  /**
   * Task name。
   */
  name:             string;
  /**
   * Task Skel name。
   */
  skelName:         string;
  /**
   * Task description。
   */
  description:      string | null;
  /**
   * Trigger list。
   */
  triggers:         Array<SkeletonTrigger>;
}
/**
 * Skeleton task trigger。
 */
export type SkeletonTrigger = {
  /**
   * Trigger name。
   */
  name:             string;
  /**
   * Trigger Skel name。
   */
  skelName:         string;
  /**
   * Trigger description。
   */
  description:      string | null;
  /**
   * Input description。
   */
  inputDescription: string | null;
  /**
   * Input example。
   */
  example:          string | null;
  /**
   * Parameter list。
   */
  arguments:        Array<SkeletonField>;
}
/**
 * Skeleton web page。
 */
export type SkeletonWebItem = {
  /**
   * Domain。
   */
  domain:           string;
  /**
   * Skeleton item hash。
   */
  schemaHash:       string;
  /**
   * Primary skeleton item hash。
   */
  mainSchemaHash:   string;
  /**
   * Whether there are multiple valid versions of the skeleton item。
   */
  isMultiVersion:   boolean;
  /**
   * Whether it is the main version of the skeleton item。
   */
  isMain:           boolean;
  /**
   * Owning DomainSchema hash。
   */
  domainSchemaHash: string;
  /**
   * Web page name。
   */
  name:             string;
  /**
   * Web Skel name。
   */
  skelName:         string;
  /**
   * Web page description。
   */
  description:      string | null;
  /**
   * Accessible Actor List。
   */
  actors:           Array<SkeletonActorRef>;
}
/**
 * Task Debug initiates a request by default。
 */
export type TaskDebugDefaultLaunchRequest = {
  /**
   * Trace ID。
   */
  traceId:       string;
  /**
   * Span ID。
   */
  spanId:        string;
  /**
   * Default task parameters JSON。
   */
  argumentsJson: string;
}
/**
 * Task Debug initiates a request。
 */
export type TaskDebugLaunchRequest = {
  /**
   * Task Skel name。
   */
  taskSkelName:    string;
  /**
   * Task schema hash。
   */
  schemaHash:      string;
  /**
   * Trigger Skel name。
   */
  triggerSkelName: string;
  /**
   * Task parameters JSON。
   */
  argumentsJson:   string;
  /**
   * Trace ID。
   */
  traceId:         string | null;
  /**
   * Span ID。
   */
  spanId:          string | null;
}
/**
 * Task called by Task Debug。
 */
export type TaskDebugTaskItem = {
  /**
   * Task name。
   */
  name:         string;
  /**
   * Task Skel name。
   */
  taskSkelName: string;
  /**
   * Task schema hash。
   */
  schemaHash:   string;
  /**
   * Task description。
   */
  description:  string | null;
}
/**
 * Trigger called by Task Debug。
 */
export type TaskDebugTriggerItem = {
  /**
   * Trigger name。
   */
  name:             string;
  /**
   * Trigger Skel name。
   */
  skelName:         string;
  /**
   * Trigger description。
   */
  description:      string | null;
  /**
   * Input description。
   */
  inputDescription: string | null;
  /**
   * Input example。
   */
  example:          string | null;
  /**
   * Parameter list。
   */
  arguments:        Array<SkeletonField>;
}
/**
 * Task execution Cron schedule。
 */
export type TaskRunnerCronScheduler = {
  /**
   * Trigger Skel name。
   */
  triggerSkelName: string;
  /**
   * Cron expression。
   */
  cronExpr:        string;
}
/**
 * Task execution capability registration information provided by the application。
 */
export type TaskRunnerRegistration = {
  /**
   * Task Skel name。
   */
  taskSkelName:   string;
  /**
   * Task schema hash。
   */
  schemaHash:     string;
  /**
   * Execution timeout, in milliseconds。
   */
  timeoutMs:      number;
  /**
   * Maximum concurrency。
   */
  concurrency:    number;
  /**
   * Whether to disallow retrying after failure。
   */
  noRetry:        boolean;
  /**
   * Cron schedule list。
   */
  cronSchedulers: Array<TaskRunnerCronScheduler>;
}
/**
 * Web processing capability registration information provided by the application。
 */
export type WebHandlerRegistration = {
  /**
   * Web Skel name。
   */
  webSkelName: string;
  /**
   * Web schema hash。
   */
  schemaHash:  string;
  /**
   * Web proxy access address。
   */
  endpoint:    string;
}
export {};
