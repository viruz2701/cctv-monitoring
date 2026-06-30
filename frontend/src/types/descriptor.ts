// ═══════════════════════════════════════════════════════════════════════
// Protocol Descriptor Types — PROTO-06
//
// Types for declarative protocol descriptors used to define
// vendor-specific camera/NVR protocol integrations.
//
// Compliance:
//   - OWASP ASVS V5 (Input validation via Zod in components)
//   - JSON Schema validation for descriptor structure
// ═══════════════════════════════════════════════════════════════════════

/** HTTP method supported by endpoint */
export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';

/** Auth type for the descriptor */
export type AuthType = 'none' | 'basic' | 'bearer' | 'digest' | 'custom';

/** Parameter type */
export type ParamType = 'string' | 'number' | 'boolean' | 'object';

/** Parser type */
export type ParserType = 'jsonpath' | 'regex' | 'xpath' | 'custom';

// ─── Parameter Definition ───────────────────────────────────────────

export interface ParamDef {
  name: string;
  type: ParamType;
  required: boolean;
  default?: unknown;
  description?: string;
  enum?: string[];
  min?: number;
  max?: number;
}

// ─── Parser Definition ──────────────────────────────────────────────

export interface ParserDef {
  type: ParserType;
  expression?: string;
  script?: string;
  /** Path to extract from response (JSONPath) */
  resultPath?: string;
}

// ─── Auth Configuration ─────────────────────────────────────────────

export interface AuthConfig {
  type: AuthType;
  /** For 'basic' auth */
  usernameField?: string;
  passwordField?: string;
  /** For 'bearer' auth — field name in response to extract token */
  tokenField?: string;
  /** For 'custom' auth — header name for the token */
  headerName?: string;
  /** Auth endpoint path (relative) */
  authEndpoint?: string;
  /** Body template for auth request */
  authBody?: Record<string, unknown>;
}

// ─── Endpoint Definition ────────────────────────────────────────────

export interface ProtocolEndpoint {
  id: string;
  method: HttpMethod;
  path: string;
  name: string;
  description?: string;
  headers?: Record<string, string>;
  queryParams?: ParamDef[];
  body?: Record<string, unknown>;
  bodyParams?: ParamDef[];
  parser?: ParserDef;
  successCodes?: number[];
  /** Timeout in seconds */
  timeout?: number;
}

// ─── Protocol Descriptor ────────────────────────────────────────────

export interface ProtocolDescriptor {
  vendor: string;
  version: string;
  description?: string;
  endpoints: ProtocolEndpoint[];
  auth?: AuthConfig;
  metadata?: Record<string, unknown>;
  /** Minimum firmware version required */
  minFirmware?: string;
  /** Category (camera, nvr, dvr, etc.) */
  category?: string;
  /** Icon/logo URL */
  icon?: string;
  createdAt?: string;
  updatedAt?: string;
}

// ─── Descriptor List Item ───────────────────────────────────────────

export interface DescriptorListItem {
  vendor: string;
  version: string;
  description?: string;
  category?: string;
  endpointCount: number;
  updatedAt?: string;
}

// ─── Test Request ───────────────────────────────────────────────────

export interface DescriptorTestRequest {
  vendor: string;
  endpointId: string;
  baseUrl: string;
  credentials?: {
    username?: string;
    password?: string;
    token?: string;
  };
  params?: Record<string, unknown>;
}

export interface DescriptorTestResponse {
  success: boolean;
  statusCode?: number;
  headers?: Record<string, string>;
  body?: unknown;
  parsedResult?: unknown;
  durationMs?: number;
  error?: string;
}

// ─── Store State ────────────────────────────────────────────────────

export type EditorMode = 'list' | 'create' | 'edit';

export type DescriptorViewTab = 'form' | 'preview' | 'test';
