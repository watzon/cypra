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
