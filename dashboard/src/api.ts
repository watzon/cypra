export interface VersionResponse {
  version: string;
  commit: string;
}

export interface CompleteSetupInput {
  token: string;
  email: string;
  displayName: string;
  ceremonyID: string;
  response: unknown;
}

export interface BeginSetupPasskeyInput {
  token: string;
  email: string;
  displayName: string;
}

export interface BeginSetupPasskeyResult {
  admin_id: string;
  ceremony_id: string;
  options: unknown;
}

export interface CompleteSetupResult {
  admin_id: string;
  backup_codes: string[];
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
  created_at?: string;
  member_count?: number;
  project_count?: number;
  user_count?: number;
  email_provider_required?: boolean;
}

export interface MeRecord {
  kind: "user" | "instance_admin";
  id: string;
  sub: string;
  email: string;
  display_name: string;
  tenant_id: string | null;
  created_at: string;
  metadata: Record<string, unknown>;
}

export interface PasskeyRecord {
  id: string;
  label: string;
  aaguid?: string;
  transports: string[];
  created_at: string;
  last_used_at?: string;
  this_device: boolean;
}

export interface SessionRecord {
  id: string;
  created_at: string;
  last_seen_at: string;
  expires_at: string;
  ip?: string;
  user_agent?: string;
  auth_kind: "cookie" | "pat";
  current: boolean;
}

export interface MFAFactorsRecord {
  totp: { enrolled: boolean; confirmed_at?: string } | null;
  backup_codes: { remaining: number; total: number };
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
  last4?: string;
  scopes: string[];
  created_at: string;
  expires_at?: string;
  last_used_at?: string;
  revoked_at?: string;
}

export interface UserRecord {
  id: string;
  email: string;
  metadata?: Record<string, unknown>;
  sub?: string;
  state?: "active" | "pending" | "deleting";
  enrolled_methods?: string[];
}

export interface UserSessionRecord {
  id: string;
  created_at: string;
  last_seen_at: string;
  expires_at: string;
  revoked: boolean;
  revoked_at?: string;
  ip?: string;
  user_agent?: string;
}

export interface UserConsentRecord {
  id: string;
  client_id: string;
  scopes: string[];
  granted_at: string;
  revoked: boolean;
  revoked_at?: string;
}

export interface UserAuditRecord {
  id: string;
  occurred_at: string;
  action: string;
  resource_kind: string;
  state_after: Record<string, unknown>;
  redacted: boolean;
}

export interface UserDetailRecord {
  user: UserRecord;
  auth_methods: string[];
  sessions: UserSessionRecord[];
  consents: UserConsentRecord[];
  audit: UserAuditRecord[];
  metadata: Record<string, unknown>;
}

export interface ProviderConfigRecord {
  kind: string;
  configured: boolean;
  healthy: boolean;
  message?: string;
}

export type AuthProviderMethod =
  | "password"
  | "magic_link"
  | "passkey"
  | "totp"
  | "google"
  | "oidc_upstream";

export interface AuthProviderRecord {
  method: AuthProviderMethod;
  enabled: boolean;
  enrolled_count: number;
  config: Record<string, unknown>;
}

export interface AuthProviderInput {
  enabled: boolean;
  config: Record<string, unknown>;
}

// DefaultAuthMethod can be a built-in method (password / magic_link / passkey
// / totp / google / oidc_upstream), a social SSO connection ("social:google",
// "social:microsoft", ...), or an enterprise OIDC connection ("oidc:<slug>").
export type DefaultAuthMethodValue =
  | AuthProviderMethod
  | `social:${SocialProviderKind}`
  | `oidc:${string}`;

export interface DefaultAuthMethodRecord {
  method?: DefaultAuthMethodValue;
}

export type SignupMode = "open" | "restricted" | "closed";

export interface RegistrationSettingsRecord {
  mode: SignupMode;
  allowlist: string[];
  invites_enabled: boolean;
}

export interface AuthMethodRecord {
  method: AuthProviderMethod;
  enabled: boolean;
  enrolled_count: number;
}

export interface SigningKeyRecord {
  kid: string;
  algorithm: string;
  state: "active" | "overlap" | "retired" | "sunsetting";
  activated_at: string;
  retires_at?: string;
  sunset_until?: string;
}

export interface AuditEntryRecord {
  id: string;
  occurred_at: string;
  actor_kind: string;
  actor_id?: string;
  action: string;
  resource_kind: string;
  resource_id?: string;
  state_before: Record<string, unknown>;
  state_after: Record<string, unknown>;
  metadata: Record<string, unknown>;
  redacted_at?: string;
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

export interface PendingInviteRecord {
  id: string;
  email: string;
  role: string;
  expires_at: string;
  created_at: string;
}

export interface TenantMemberRecord {
  id: string;
  user_id: string;
  email: string;
  role: "owner" | "admin" | "member";
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

export interface DashboardSummaryRecord {
  tenants: number;
  projects: number;
  users: number;
  active_signing_key: boolean;
  recent_audit: { action: string; resource_id: string }[];
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

export async function verifySetupToken(token: string): Promise<void> {
  const response = await fetch("/api/v1/setup/verify", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ token }),
  });
  if (!response.ok) {
    const payload = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(payload.error ?? "setup.token_invalid");
  }
}

export async function beginSetupPasskey(
  input: BeginSetupPasskeyInput,
): Promise<BeginSetupPasskeyResult> {
  const response = await fetch("/api/v1/setup/passkey/begin", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      token: input.token,
      email: input.email,
      display_name: input.displayName,
    }),
  });
  if (!response.ok) {
    const payload = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(payload.error ?? "setup.passkey_begin_failed");
  }
  return (await response.json()) as BeginSetupPasskeyResult;
}

export async function completeSetup(input: CompleteSetupInput): Promise<CompleteSetupResult> {
  const response = await fetch("/api/v1/setup/complete", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      token: input.token,
      email: input.email,
      display_name: input.displayName,
      ceremony_id: input.ceremonyID,
      response: input.response,
    }),
  });
  if (!response.ok) {
    const payload = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(payload.error ?? "setup.complete_failed");
  }
  return (await response.json()) as CompleteSetupResult;
}

export async function signOut(): Promise<void> {
  try {
    await fetch("/api/v1/auth/logout", { method: "POST" });
  } catch {
    // ignore — we navigate away regardless
  }
  window.location.assign("/");
}

export async function revokeOtherSessions(): Promise<void> {
  const response = await fetch("/api/v1/auth/sessions/revoke-others", { method: "POST" });
  if (!response.ok) {
    const payload = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(payload.error ?? "auth.session_revoke_failed");
  }
}

export class AuthError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
    this.name = "AuthError";
  }
}

export async function getMe(): Promise<MeRecord> {
  const response = await fetch("/api/v1/users/me");
  if (response.status === 401) {
    throw new AuthError(401, "auth.unauthorized");
  }
  if (!response.ok) {
    throw new Error(response.status === 403 ? "auth.forbidden" : "users.me.fetch_failed");
  }
  return (await response.json()) as MeRecord;
}

export async function listPasskeys(): Promise<PasskeyRecord[]> {
  return fetchJSON<PasskeyRecord[]>("/api/v1/auth/passkeys", "passkey.list_failed");
}

export async function removePasskey(id: string): Promise<void> {
  const response = await fetch(`/api/v1/auth/passkeys/${id}`, { method: "DELETE" });
  if (!response.ok) {
    const payload = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(payload.error ?? "passkey.delete_failed");
  }
}

export async function listSessions(): Promise<SessionRecord[]> {
  return fetchJSON<SessionRecord[]>("/api/v1/auth/sessions", "session.list_failed");
}

export async function revokeSession(id: string): Promise<void> {
  const response = await fetch(`/api/v1/auth/sessions/${id}`, { method: "DELETE" });
  if (!response.ok) {
    const payload = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(payload.error ?? "session.revoke_failed");
  }
}

export async function getMFAFactors(): Promise<MFAFactorsRecord> {
  return fetchJSON<MFAFactorsRecord>("/api/v1/auth/mfa/factors", "mfa.fetch_failed");
}

export interface CreatePATResult {
  id: string;
  token: string;
  scopes?: string[];
  name?: string;
}

export async function createPAT(name: string, scopes: string[]): Promise<CreatePATResult> {
  const response = await fetch("/api/v1/pats/", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ name, scopes }),
  });
  if (!response.ok) {
    const payload = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(payload.error ?? "pat.create_failed");
  }
  return (await response.json()) as CreatePATResult;
}

export async function revokePAT(id: string): Promise<void> {
  const response = await fetch(`/api/v1/pats/${id}`, {
    method: "DELETE",
  });
  if (!response.ok && response.status !== 204) {
    const payload = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(payload.error ?? "pat.revoke_failed");
  }
}

export async function regenerateBackupCodes(userID: string): Promise<{ codes: string[] }> {
  const response = await fetch("/api/v1/auth/backup-codes/regenerate", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ user_id: userID }),
  });
  if (!response.ok) {
    const payload = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(payload.error ?? "auth.backup_codes_failed");
  }
  return (await response.json()) as { codes: string[] };
}

export async function registerPasskey(input: {
  userID: string;
  credentialID: string;
  publicKey: string;
}): Promise<void> {
  const response = await fetch("/api/v1/auth/passkey/register", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      user_id: input.userID,
      credential_id: input.credentialID,
      public_key: input.publicKey,
    }),
  });
  if (!response.ok) {
    const payload = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(payload.error ?? "passkey.register_failed");
  }
}

export interface CreateTenantInput {
  slug: string;
  name: string;
}

export async function createTenant(input: CreateTenantInput): Promise<TenantRecord> {
  const response = await fetch("/api/v1/tenants/", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(input),
  });
  if (!response.ok) {
    const payload = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(payload.error ?? "tenant.create_failed");
  }
  return (await response.json()) as TenantRecord;
}

export interface CreateProjectInput {
  slug: string;
  name: string;
}

export interface UpdateProjectInput {
  id: string;
  name: string;
  redirect_uris: string[];
  allowed_scopes: string[];
  token_endpoint_auth_method: "client_secret_basic" | "client_secret_post" | "none";
}

export async function createProject(input: CreateProjectInput): Promise<ProjectRecord> {
  const response = await fetch("/api/v1/projects/", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(input),
  });
  if (!response.ok) {
    const payload = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(payload.error ?? "project.create_failed");
  }
  return (await response.json()) as ProjectRecord;
}

export async function updateProject(input: UpdateProjectInput): Promise<ProjectRecord> {
  return putJSON<ProjectRecord>(
    `/api/v1/projects/${input.id}`,
    {
      name: input.name,
      redirect_uris: input.redirect_uris,
      allowed_scopes: input.allowed_scopes,
      token_endpoint_auth_method: input.token_endpoint_auth_method,
    },
    "project.update_failed",
  );
}

export async function rotateProjectSecret(id: string): Promise<ProjectRecord> {
  return postJSON<ProjectRecord>(
    `/api/v1/projects/${id}/rotate-secret`,
    {},
    "project.secret_rotate_failed",
  );
}

export async function deleteProject(id: string): Promise<void> {
  const response = await fetch(`/api/v1/projects/${id}`, {
    method: "DELETE",
  });
  if (!response.ok) {
    const payload = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(payload.error ?? "project.delete_failed");
  }
}

export interface InviteUserInput {
  email: string;
  role: "owner" | "admin" | "member";
  redirectURL?: string;
}

export async function inviteUser(input: InviteUserInput): Promise<void> {
  const response = await fetch("/api/v1/admin/invite", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      email: input.email,
      role: input.role,
      redirect_url: input.redirectURL,
    }),
  });
  if (!response.ok) {
    const payload = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(payload.error ?? "invite.issue_failed");
  }
}

export const resendInvite = (email: string, role: "owner" | "admin" | "member" = "member") =>
  inviteUser({ email, role });

export async function listTenantInvites(): Promise<PendingInviteRecord[]> {
  return fetchJSON<PendingInviteRecord[]>("/api/v1/admin/invites", "invite.list_failed");
}

export async function revokeTenantInvite(id: string): Promise<void> {
  const response = await fetch(`/api/v1/admin/invites/${id}`, {
    method: "DELETE",
  });
  if (!response.ok) {
    const payload = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(payload.error ?? "invite.revoke_failed");
  }
}

export interface AuditFilters {
  range?: string;
  action?: string;
  actor?: string;
  resourceKind?: string;
}

export function auditExportURL(filters: AuditFilters = {}): string {
  const params = new URLSearchParams();
  if (filters.range) params.set("range", filters.range);
  if (filters.action) params.set("action", filters.action);
  if (filters.actor) params.set("actor", filters.actor);
  if (filters.resourceKind) params.set("resource_kind", filters.resourceKind);
  const qs = params.toString();
  return qs ? `/api/v1/audit/export?${qs}` : "/api/v1/audit/export";
}

export function auditListURL(scope: "tenant" | "instance", filters: AuditFilters = {}): string {
  const params = new URLSearchParams();
  if (filters.range) params.set("range", filters.range);
  if (filters.action) params.set("action", filters.action);
  if (filters.actor) params.set("actor", filters.actor);
  if (filters.resourceKind) params.set("resource_kind", filters.resourceKind);
  const base = scope === "instance" ? "/api/v1/instance/audit" : "/api/v1/audit/";
  const qs = params.toString();
  return qs ? `${base}?${qs}` : base;
}

export async function listAuditEntries(
  scope: "tenant" | "instance",
  filters: AuditFilters = {},
): Promise<AuditEntryRecord[]> {
  return fetchJSON<AuditEntryRecord[]>(auditListURL(scope, filters), "audit.list_failed");
}

export async function listTenants(): Promise<TenantRecord[]> {
  const response = await fetch("/api/v1/tenants/");
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
    },
    body: JSON.stringify(branding),
  });
  if (!response.ok) {
    const payload = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(payload.error ?? "tenant.branding_failed");
  }
  return (await response.json()) as TenantBranding;
}

export interface TenantLogoUploadResult {
  object_id: string;
  content_type: string;
  byte_size: number;
  logo_url: string;
}

export async function uploadTenantLogo(
  tenantId: string,
  file: File,
): Promise<TenantLogoUploadResult> {
  const body = new FormData();
  body.append("file", file);
  const response = await fetch(`/api/v1/tenants/${tenantId}/logo`, {
    method: "PUT",
    body,
  });
  if (!response.ok) {
    const payload = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(payload.error ?? "tenant.logo_failed");
  }
  return (await response.json()) as TenantLogoUploadResult;
}

export async function deleteTenantLogo(tenantId: string): Promise<void> {
  const response = await fetch(`/api/v1/tenants/${tenantId}/logo`, { method: "DELETE" });
  if (!response.ok && response.status !== 204) {
    const payload = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(payload.error ?? "tenant.logo_failed");
  }
}

export async function suspendTenant(tenantId: string): Promise<void> {
  await postJSON(`/api/v1/tenants/${tenantId}/suspend`, {}, "tenant.suspend_failed");
}

export async function resumeTenant(tenantId: string): Promise<void> {
  await postJSON(`/api/v1/tenants/${tenantId}/resume`, {}, "tenant.resume_failed");
}

export async function scheduleTenantDeletion(tenantId: string, slug: string): Promise<void> {
  await postJSON(`/api/v1/tenants/${tenantId}/delete`, { slug }, "tenant.delete_failed");
}

export async function cancelTenantDeletion(tenantId: string): Promise<void> {
  await postJSON(`/api/v1/tenants/${tenantId}/delete/cancel`, {}, "tenant.delete_cancel_failed");
}

export async function listProjects(): Promise<ProjectRecord[]> {
  const response = await fetch("/api/v1/projects/");
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
  const response = await fetch("/api/v1/pats/");
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
  const response = await fetch("/api/v1/users/");
  if (!response.ok) {
    throw new Error(response.status === 403 ? "auth.forbidden" : "user.list_failed");
  }
  const payload = (await response.json()) as unknown;
  if (!Array.isArray(payload)) {
    throw new Error("user.list_invalid");
  }
  return payload as UserRecord[];
}

export async function getUserDetail(id: string): Promise<UserDetailRecord> {
  return fetchJSON<UserDetailRecord>(`/api/v1/users/${id}`, "user.detail_failed");
}

export async function resetUserPassword(id: string): Promise<void> {
  await postJSON(`/api/v1/users/${id}/reset-password`, {}, "user.password_reset_failed");
}

export async function reinviteUser(id: string): Promise<void> {
  await postJSON(`/api/v1/users/${id}/reinvite`, {}, "user.reinvite_failed");
}

export async function disableUserMFA(id: string): Promise<void> {
  await postJSON(`/api/v1/users/${id}/disable-mfa`, {}, "user.mfa_disable_failed");
}

export async function enrollUserFactor(id: string): Promise<void> {
  await postJSON(`/api/v1/users/${id}/enroll-factor`, {}, "user.factor_enroll_failed");
}

export async function deleteUserDSR(id: string): Promise<void> {
  const response = await fetch(`/api/v1/users/${id}`, {
    method: "DELETE",
  });
  if (!response.ok) {
    const payload = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(payload.error ?? "user.delete_failed");
  }
}

export function userExportURL(id: string): string {
  return `/api/v1/users/${id}/export`;
}

export async function listAuthProviders(): Promise<AuthProviderRecord[]> {
  return fetchJSON<AuthProviderRecord[]>("/api/v1/auth-providers/", "auth_providers.list_failed");
}

export async function getAuthProvider(method: AuthProviderMethod): Promise<AuthProviderRecord> {
  return fetchJSON<AuthProviderRecord>(
    `/api/v1/auth-providers/${method}`,
    "auth_providers.load_failed",
  );
}

export async function saveAuthProvider(
  method: AuthProviderMethod,
  input: AuthProviderInput,
): Promise<AuthProviderRecord> {
  return putJSON<AuthProviderRecord>(
    `/api/v1/auth-providers/${method}`,
    input,
    "auth_providers.update_failed",
  );
}

export async function getDefaultAuthMethod(): Promise<DefaultAuthMethodRecord> {
  return fetchJSON<DefaultAuthMethodRecord>(
    "/api/v1/auth-providers/default",
    "auth_providers.default_load_failed",
  );
}

export async function setDefaultAuthMethod(
  method: DefaultAuthMethodValue | null,
): Promise<DefaultAuthMethodRecord> {
  return putJSON<DefaultAuthMethodRecord>(
    "/api/v1/auth-providers/default",
    { method },
    "auth_providers.default_save_failed",
  );
}

export async function getRegistrationSettings(): Promise<RegistrationSettingsRecord> {
  return fetchJSON<RegistrationSettingsRecord>(
    "/api/v1/auth-providers/registration",
    "auth_providers.registration_load_failed",
  );
}

export async function saveRegistrationSettings(
  input: RegistrationSettingsRecord,
): Promise<RegistrationSettingsRecord> {
  return putJSON<RegistrationSettingsRecord>(
    "/api/v1/auth-providers/registration",
    input,
    "auth_providers.registration_save_failed",
  );
}

export type SocialProviderKind = "google" | "microsoft" | "apple" | "github" | "discord";

export interface SocialConnectionRecord {
  kind: SocialProviderKind;
  label: string;
  icon_url: string;
  configured: boolean;
  enabled: boolean;
  client_id_set: boolean;
  allowed_domains: string[];
  allow_signup: boolean | null;
  config: Record<string, unknown>;
  updated_at?: string;
}

export interface SocialConnectionInput {
  enabled: boolean;
  client_id: string;
  client_secret: string;
  allowed_domains: string[];
  allow_signup: boolean | null;
}

export interface OIDCConnectionRecord {
  id: string;
  slug: string;
  display_name: string;
  issuer_url: string;
  scopes: string[];
  enabled: boolean;
  client_id_set: boolean;
  allowed_domains: string[];
  allow_signup: boolean | null;
  updated_at?: string;
}

export interface OIDCConnectionInput {
  slug?: string;
  display_name: string;
  issuer_url: string;
  client_id?: string;
  client_secret?: string;
  scopes: string[];
  enabled: boolean;
  allowed_domains: string[];
  allow_signup: boolean | null;
}

export async function listSocialConnections(): Promise<SocialConnectionRecord[]> {
  const payload = await fetchJSON<{ connections: SocialConnectionRecord[] }>(
    "/api/v1/auth-providers/social/",
    "auth_providers.social_list_failed",
  );
  return payload.connections ?? [];
}

export async function getSocialConnection(kind: SocialProviderKind): Promise<SocialConnectionRecord> {
  return fetchJSON<SocialConnectionRecord>(
    `/api/v1/auth-providers/social/${kind}`,
    "auth_providers.social_load_failed",
  );
}

export async function saveSocialConnection(
  kind: SocialProviderKind,
  input: SocialConnectionInput,
): Promise<SocialConnectionRecord> {
  return putJSON<SocialConnectionRecord>(
    `/api/v1/auth-providers/social/${kind}`,
    input,
    "auth_providers.social_save_failed",
  );
}

export async function deleteSocialConnection(kind: SocialProviderKind): Promise<void> {
  const response = await fetch(`/api/v1/auth-providers/social/${kind}`, { method: "DELETE" });
  if (!response.ok && response.status !== 204) {
    const payload = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(payload.error ?? "auth_providers.social_delete_failed");
  }
}

export async function listOIDCConnections(): Promise<OIDCConnectionRecord[]> {
  const payload = await fetchJSON<{ connections: OIDCConnectionRecord[] }>(
    "/api/v1/auth-providers/oidc/",
    "auth_providers.oidc_list_failed",
  );
  return payload.connections ?? [];
}

export async function createOIDCConnection(input: OIDCConnectionInput): Promise<OIDCConnectionRecord> {
  return postJSON<OIDCConnectionRecord>(
    "/api/v1/auth-providers/oidc/",
    input,
    "auth_providers.oidc_save_failed",
  );
}

export async function getOIDCConnection(slug: string): Promise<OIDCConnectionRecord> {
  return fetchJSON<OIDCConnectionRecord>(
    `/api/v1/auth-providers/oidc/${slug}`,
    "auth_providers.oidc_load_failed",
  );
}

export async function updateOIDCConnection(
  slug: string,
  input: OIDCConnectionInput,
): Promise<OIDCConnectionRecord> {
  return putJSON<OIDCConnectionRecord>(
    `/api/v1/auth-providers/oidc/${slug}`,
    input,
    "auth_providers.oidc_save_failed",
  );
}

export async function deleteOIDCConnection(slug: string): Promise<void> {
  const response = await fetch(`/api/v1/auth-providers/oidc/${slug}`, { method: "DELETE" });
  if (!response.ok && response.status !== 204) {
    const payload = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(payload.error ?? "auth_providers.oidc_delete_failed");
  }
}

// Back-compat shim used by overview tiles. Prefer listAuthProviders.
export async function listAuthMethods(): Promise<AuthMethodRecord[]> {
  const records = await listAuthProviders();
  return records.map((record) => ({
    method: record.method,
    enabled: record.enabled,
    enrolled_count: record.enrolled_count,
  }));
}

export async function listSigningKeys(): Promise<SigningKeyRecord[]> {
  return fetchJSON<SigningKeyRecord[]>("/api/v1/signing-keys/", "signing_keys.list_failed");
}

export async function listTenantMembers(): Promise<TenantMemberRecord[]> {
  return fetchJSON<TenantMemberRecord[]>("/api/v1/admin/members", "members.list_failed");
}

export async function updateTenantMemberRole(
  id: string,
  role: TenantMemberRecord["role"],
): Promise<TenantMemberRecord> {
  return putJSON<TenantMemberRecord>(
    `/api/v1/admin/members/${id}`,
    { role },
    "members.update_failed",
  );
}

export async function removeTenantMember(id: string): Promise<void> {
  const response = await fetch(`/api/v1/admin/members/${id}`, {
    method: "DELETE",
  });
  if (!response.ok) {
    const payload = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(payload.error ?? "members.remove_failed");
  }
}

export async function forceRotateSigningKey(kid: string): Promise<SigningKeyRecord[]> {
  return postJSON<SigningKeyRecord[]>(
    "/api/v1/signing-keys/rotate",
    { kid },
    "signing_keys.rotate_failed",
  );
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
  );
}

export async function testEmailProviderConfig(): Promise<ProviderConfigRecord> {
  return postJSON<ProviderConfigRecord>(
    "/api/v1/provider-config/email/test",
    {},
    "provider.email_test_failed",
  );
}

export async function getUpstreamProviderConfig(): Promise<ProviderConfigRecord> {
  return fetchJSON<ProviderConfigRecord>(
    "/api/v1/provider-config/upstream",
    "provider.upstream_failed",
  );
}

export async function testUpstreamProviderConfig(): Promise<ProviderConfigRecord> {
  return postJSON<ProviderConfigRecord>(
    "/api/v1/provider-config/upstream/test",
    {},
    "provider.upstream_test_failed",
  );
}

export async function saveUpstreamProviderConfig(
  input: UpstreamProviderConfigInput,
): Promise<ProviderConfigRecord> {
  return putJSON<ProviderConfigRecord>(
    "/api/v1/provider-config/upstream",
    input,
    "provider.upstream_save_failed",
  );
}

export async function listInstanceAdmins(): Promise<InstanceAdminRecord[]> {
  return fetchJSON<InstanceAdminRecord[]>("/api/v1/instance/admins", "instance.admins_failed");
}

export async function listInstanceInvites(): Promise<PendingInviteRecord[]> {
  return fetchJSON<PendingInviteRecord[]>("/api/v1/instance/invites", "instance.invites_failed");
}

export async function inviteInstanceAdmin(email: string): Promise<PendingInviteRecord> {
  const response = await fetch("/api/v1/instance/invite", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, redirect_url: `${window.location.origin}/setup` }),
  });
  if (!response.ok) {
    const payload = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(payload.error ?? "instance.invite_failed");
  }
  return (await response.json()) as PendingInviteRecord;
}

export async function revokeInstanceInvite(id: string): Promise<void> {
  const response = await fetch(`/api/v1/instance/invites/${id}`, {
    method: "DELETE",
  });
  if (!response.ok) {
    const payload = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(payload.error ?? "instance.invite_revoke_failed");
  }
}

export async function getDashboardSummary(): Promise<DashboardSummaryRecord> {
  return fetchJSON<DashboardSummaryRecord>("/api/v1/instance/summary", "instance.summary_failed");
}

export async function demoteInstanceAdmin(id: string): Promise<void> {
  const response = await fetch(`/api/v1/instance/admins/${id}`, {
    method: "DELETE",
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
  );
}

async function fetchJSON<T>(url: string, fallbackError: string): Promise<T> {
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error(response.status === 403 ? "auth.forbidden" : fallbackError);
  }
  return (await response.json()) as T;
}

async function putJSON<T>(url: string, body: unknown, fallbackError: string): Promise<T> {
  const response = await fetch(url, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!response.ok) {
    const payload = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(payload.error ?? fallbackError);
  }
  return (await response.json()) as T;
}

async function postJSON<T>(url: string, body: unknown, fallbackError: string): Promise<T> {
  const response = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!response.ok) {
    const payload = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(payload.error ?? fallbackError);
  }
  return (await response.json()) as T;
}
