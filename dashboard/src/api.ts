export interface VersionResponse {
  version: string;
  commit: string;
}

export interface TenantBranding {
  display_name?: string;
  logo_url?: string;
  accent?: string;
  powered_by?: boolean;
}

export interface TenantRecord {
  id: string;
  slug: string;
  name: string;
  branding?: TenantBranding;
  settings?: Record<string, unknown>;
  member_count?: number;
  project_count?: number;
  user_count?: number;
  email_provider_required?: boolean;
}

export interface ProjectRecord {
  id: string;
  slug: string;
  name: string;
  issuer_url?: string;
  client_id?: string;
  client_secret?: string;
  redirect_uris?: string[];
  allowed_scopes?: string[];
  token_endpoint_auth_method?: "client_secret_basic" | "client_secret_post" | "none";
  rotation_in_progress?: boolean;
}

export interface PersonalAccessTokenRecord {
  id: string;
  name: string;
  scopes: string[];
  created_at: string;
  expires_at?: string;
  last_used_at?: string;
}

export interface UserRecord {
  id: string;
  email: string;
  metadata?: Record<string, unknown>;
  sub?: string;
  state?: "active" | "pending" | "deleting";
  enrolled_methods?: string[];
}

export interface ProviderConfigRecord {
  kind: string;
  configured: boolean;
  healthy: boolean;
  message?: string;
}

export interface EmailProviderConfigInput {
  kind: string;
  from_address: string;
  from_name: string;
  config: string;
}

export interface UpstreamProviderConfigInput {
  client_id: string;
  client_secret: string;
  enabled: boolean;
}

export interface InstanceAdminRecord {
  id: string;
  email: string;
  role: "owner" | "admin";
  created_at: string;
  last_seen_at?: string;
}

export interface InstanceDiagnosticsRecord {
  health: Record<string, boolean>;
  version: VersionResponse & { build_date?: string };
  migrations: { current: number; pending: string[] };
  master_key_rotation: { phase: string; rows_done: number; rows_total: number; eta?: string };
  storage: {
    kind: string;
    bucket?: string;
    endpoint?: string;
    region?: string;
    credentials_present: boolean;
  };
}

export async function getVersion(): Promise<VersionResponse> {
  const response = await fetch("/api/v1/version");
  if (!response.ok) {
    throw new Error("version request failed");
  }
  return (await response.json()) as VersionResponse;
}

export async function getReady(): Promise<boolean> {
  const response = await fetch("/readyz");
  return response.ok;
}

export async function listTenants(): Promise<TenantRecord[]> {
  const response = await fetch("/api/v1/tenants/", {
    headers: { "X-Cypra-Instance-Admin": "true" },
  });
  if (response.status === 403) {
    throw new Error("auth.forbidden");
  }
  if (!response.ok) {
    throw new Error("tenant.list_failed");
  }
  const payload = (await response.json()) as unknown;
  if (!Array.isArray(payload)) {
    throw new Error("tenant.list_invalid");
  }
  return payload as TenantRecord[];
}

export async function saveTenantBranding(
  tenantId: string,
  branding: TenantBranding,
): Promise<TenantBranding> {
  const response = await fetch(`/api/v1/tenants/${tenantId}/branding`, {
    method: "PATCH",
    headers: {
      "Content-Type": "application/json",
      "X-Cypra-Instance-Admin": "true",
    },
    body: JSON.stringify(branding),
  });
  if (!response.ok) {
    const payload = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(payload.error ?? "tenant.branding_failed");
  }
  return (await response.json()) as TenantBranding;
}

export async function listProjects(): Promise<ProjectRecord[]> {
  const response = await fetch("/api/v1/projects/", {
    headers: { "X-Cypra-Tenant-Role": "admin" },
  });
  if (!response.ok) {
    throw new Error(response.status === 403 ? "auth.forbidden" : "project.list_failed");
  }
  const payload = (await response.json()) as unknown;
  if (!Array.isArray(payload)) {
    throw new Error("project.list_invalid");
  }
  return payload as ProjectRecord[];
}

export async function listPersonalAccessTokens(): Promise<PersonalAccessTokenRecord[]> {
  const response = await fetch("/api/v1/pats/", {
    headers: { "X-Cypra-Tenant-Role": "admin" },
  });
  if (!response.ok) {
    throw new Error(response.status === 403 ? "auth.forbidden" : "pat.list_failed");
  }
  const payload = (await response.json()) as unknown;
  if (!Array.isArray(payload)) {
    throw new Error("pat.list_invalid");
  }
  return payload as PersonalAccessTokenRecord[];
}

export async function listUsers(): Promise<UserRecord[]> {
  const response = await fetch("/api/v1/users/", {
    headers: { "X-Cypra-Tenant-Role": "admin" },
  });
  if (!response.ok) {
    throw new Error(response.status === 403 ? "auth.forbidden" : "user.list_failed");
  }
  const payload = (await response.json()) as unknown;
  if (!Array.isArray(payload)) {
    throw new Error("user.list_invalid");
  }
  return payload as UserRecord[];
}

export async function getEmailProviderConfig(): Promise<ProviderConfigRecord> {
  return fetchJSON<ProviderConfigRecord>("/api/v1/provider-config/email", "provider.email_failed");
}

export async function saveEmailProviderConfig(
  input: EmailProviderConfigInput,
): Promise<ProviderConfigRecord> {
  return putJSON<ProviderConfigRecord>(
    "/api/v1/provider-config/email",
    input,
    "provider.email_save_failed",
    { "X-Cypra-Tenant-Role": "admin" },
  );
}

export async function getUpstreamProviderConfig(): Promise<ProviderConfigRecord> {
  return fetchJSON<ProviderConfigRecord>(
    "/api/v1/provider-config/upstream",
    "provider.upstream_failed",
  );
}

export async function saveUpstreamProviderConfig(
  input: UpstreamProviderConfigInput,
): Promise<ProviderConfigRecord> {
  return putJSON<ProviderConfigRecord>(
    "/api/v1/provider-config/upstream",
    input,
    "provider.upstream_save_failed",
    { "X-Cypra-Tenant-Role": "admin" },
  );
}

export async function listInstanceAdmins(): Promise<InstanceAdminRecord[]> {
  return fetchJSON<InstanceAdminRecord[]>("/api/v1/instance/admins", "instance.admins_failed", {
    "X-Cypra-Instance-Admin": "true",
  });
}

export async function demoteInstanceAdmin(id: string): Promise<void> {
  const response = await fetch(`/api/v1/instance/admins/${id}`, {
    method: "DELETE",
    headers: { "X-Cypra-Instance-Admin": "true" },
  });
  if (!response.ok) {
    const payload = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(payload.error ?? "instance_admin.demote_failed");
  }
}

export async function getInstanceDiagnostics(): Promise<InstanceDiagnosticsRecord> {
  return fetchJSON<InstanceDiagnosticsRecord>(
    "/api/v1/instance/diagnostics",
    "instance.diagnostics_failed",
    { "X-Cypra-Instance-Admin": "true" },
  );
}

async function fetchJSON<T>(url: string, fallbackError: string, headers?: HeadersInit): Promise<T> {
  const response = await fetch(url, { headers });
  if (!response.ok) {
    throw new Error(response.status === 403 ? "auth.forbidden" : fallbackError);
  }
  return (await response.json()) as T;
}

async function putJSON<T>(
  url: string,
  body: unknown,
  fallbackError: string,
  headers?: HeadersInit,
): Promise<T> {
  const requestHeaders = new Headers(headers);
  requestHeaders.set("Content-Type", "application/json");
  const response = await fetch(url, {
    method: "PUT",
    headers: requestHeaders,
    body: JSON.stringify(body),
  });
  if (!response.ok) {
    const payload = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(payload.error ?? fallbackError);
  }
  return (await response.json()) as T;
}
