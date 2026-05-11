import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";

import {
  auditExportURL,
  beginSetupPasskey,
  completeSetup,
  deleteProject,
  deleteUserDSR,
  demoteInstanceAdmin,
  disableUserMFA,
  enrollUserFactor,
  forceRotateSigningKey,
  getEmailProviderConfig,
  getDashboardSummary,
  getInstanceDiagnostics,
  getReady,
  getUserDetail,
  getUpstreamProviderConfig,
  inviteInstanceAdmin,
  listInstanceAdmins,
  listInstanceInvites,
  listAuthMethods,
  listAuthProviders,
  getAuthProvider,
  saveAuthProvider,
  getDefaultAuthMethod,
  setDefaultAuthMethod,
  getRegistrationSettings,
  saveRegistrationSettings,
  listAuditEntries,
  listPasskeys,
  listPersonalAccessTokens,
  listSessions,
  getMFAFactors,
  getMe,
  removePasskey,
  revokeSession,
  listProjects,
  listSigningKeys,
  listTenantInvites,
  listTenantMembers,
  listTenants,
  listUsers,
  removeTenantMember,
  resendInvite,
  rotateProjectSecret,
  revokeOtherSessions,
  revokeInstanceInvite,
  revokePAT,
  revokeTenantInvite,
  reinviteUser,
  resetUserPassword,
  resumeTenant,
  cancelTenantDeletion,
  deleteTenantLogo,
  saveEmailProviderConfig,
  saveTenantBranding,
  uploadTenantLogo,
  saveUpstreamProviderConfig,
  listSocialConnections,
  saveSocialConnection,
  deleteSocialConnection,
  listOIDCConnections,
  createOIDCConnection,
  updateOIDCConnection,
  deleteOIDCConnection,
  scheduleTenantDeletion,
  signOut,
  suspendTenant,
  testEmailProviderConfig,
  testUpstreamProviderConfig,
  updateProject,
  updateTenantMemberRole,
  userExportURL,
  verifySetupToken,
  type InstanceAdminRecord,
  type AuditEntryRecord,
  type AuthMethodRecord,
  type AuthProviderMethod,
  type DashboardSummaryRecord,
  type RegistrationSettingsRecord,
  type SignupMode,
  type MFAFactorsRecord,
  type MeRecord,
  type PasskeyRecord,
  type PendingInviteRecord,
  type InstanceDiagnosticsRecord,
  type PersonalAccessTokenRecord,
  type ProjectRecord,
  type ProviderConfigRecord,
  type SessionRecord,
  type SigningKeyRecord,
  type SocialConnectionInput,
  type SocialProviderKind,
  type OIDCConnectionRecord,
  type OIDCConnectionInput,
  type TenantMemberRecord,
  type TenantRecord,
  type UserDetailRecord,
  type UserRecord,
} from "@/api";
import {
  AuditEntry,
  Avatar,
  BackupCodeGrid,
  Button,
  Card,
  CodeBlock,
  ConfirmationDialog,
  EmptyState,
  ErrorState,
  IconButton,
  IdentifierPill,
  ImageUpload,
  InlineAlert,
  KeyRotationTimeline,
  ListRow,
  LoadingState,
  MaskedSecret,
  Modal,
  PageHeader,
  PermissionMatrix,
  SaveBar,
  SettingsRow,
  StatusPip,
  Switch,
  Tag,
  TextInput,
  ThemeToggle,
  Toast,
  Tooltip,
  type StatusVariant,
} from "@/components";
import {
  AddPasskeyModal,
  CreatePATModal,
  CreateProjectModal,
  CreateTenantModal,
  InviteUserModal,
  RegenerateBackupCodesModal,
} from "@/modals";
import { Icons, PasskeyGlyph } from "@/icons";
import { cn, formatErrorCode } from "@/lib/utils";

const CURRENT_USER_ID = "00000000-0000-0000-0000-00000000bo0t";
const PASSKEY_RP_ID = typeof window !== "undefined" ? window.location.hostname : "cypra.localhost";

const defaultAccent = ["#", "767676"].join("");

function galleryState() {
  return new URLSearchParams(window.location.search).get("state");
}

function demoStateEnabled() {
  return galleryState() === "demo";
}

const demoTenants: TenantRecord[] = [
  {
    id: "00000000-0000-0000-0000-00000000acme",
    slug: "acme",
    name: "Acme Operations",
    branding: { display_name: "Acme Login", accent: defaultAccent, powered_by: true },
    member_count: 3,
    project_count: 2,
    user_count: 128,
    email_provider_required: true,
  },
  {
    id: "00000000-0000-0000-0000-00000000brav",
    slug: "bravo",
    name: "Bravo Labs",
    branding: { display_name: "Bravo Identity", accent: defaultAccent, powered_by: true },
    member_count: 1,
    project_count: 0,
    user_count: 8,
    email_provider_required: false,
  },
];

const demoProjects: ProjectRecord[] = [
  {
    id: "00000000-0000-0000-0000-00000000app1",
    slug: "console",
    name: "Console App",
    issuer_url: "https://acme.cypra.localhost",
    client_id: "client_cypra_acme_console",
    client_secret: "client_secret_demo_only",
    redirect_uris: ["https://app.example.com/api/auth/callback/cypra"],
    allowed_scopes: ["openid", "email", "profile"],
    token_endpoint_auth_method: "client_secret_basic",
    rotation_in_progress: true,
  },
];

const demoTokens: PersonalAccessTokenRecord[] = [
  {
    id: "00000000-0000-0000-0000-00000000pat1",
    name: "Deploy automation",
    scopes: ["projects.read", "users.write"],
    created_at: "2026-05-01",
    expires_at: "2026-08-01",
    last_used_at: "2026-05-05",
  },
];

const demoUsers: UserRecord[] = [
  {
    id: "00000000-0000-0000-0000-00000000ada1",
    email: "ada@example.com",
    sub: "usr_acme_ada",
    state: "active",
    enrolled_methods: ["passkey", "totp", "magic-link"],
  },
  {
    id: "00000000-0000-0000-0000-00000000alan",
    email: "alan@example.com",
    sub: "usr_acme_alan",
    state: "pending",
    enrolled_methods: ["password"],
  },
];

const demoAuthMethods: AuthMethodRecord[] = [
  { method: "password", enabled: true, enrolled_count: 47 },
  { method: "magic_link", enabled: true, enrolled_count: 18 },
  { method: "passkey", enabled: true, enrolled_count: 9 },
  { method: "totp", enabled: true, enrolled_count: 12 },
  { method: "google", enabled: true, enrolled_count: 5 },
  { method: "oidc_upstream", enabled: false, enrolled_count: 0 },
];

const demoSigningKeys: SigningKeyRecord[] = [
  {
    kid: "kid_active_2026_05",
    algorithm: "RS256",
    state: "active",
    activated_at: "2026-05-01T00:00:00Z",
    retires_at: "2026-08-01T00:00:00Z",
  },
];

const demoEmailProvider: ProviderConfigRecord = {
  kind: "terminal",
  configured: false,
  healthy: true,
  message: "Terminal email is available for local development.",
};

const demoUpstreamProvider: ProviderConfigRecord = {
  kind: "google",
  configured: false,
  healthy: false,
  message: "Google OAuth credentials are not configured yet.",
};

const demoInstanceAdmins: InstanceAdminRecord[] = [
  {
    id: "00000000-0000-0000-0000-00000000root",
    email: "root@example.com",
    role: "owner",
    created_at: "2026-05-01",
    last_seen_at: "2026-05-05",
  },
];

const demoTenantMembers: TenantMemberRecord[] = [
  {
    id: "00000000-0000-0000-0000-00000000mem1",
    user_id: "00000000-0000-0000-0000-00000000ada1",
    email: "ada@example.com",
    role: "owner",
    created_at: "2026-05-01T00:00:00Z",
    last_seen_at: "2026-05-06T00:00:00Z",
  },
  {
    id: "00000000-0000-0000-0000-00000000mem2",
    user_id: "00000000-0000-0000-0000-00000000alan",
    email: "alan@example.com",
    role: "admin",
    created_at: "2026-05-02T00:00:00Z",
    last_seen_at: "2026-05-05T00:00:00Z",
  },
];

const demoPendingInvites: PendingInviteRecord[] = [
  {
    id: "00000000-0000-0000-0000-00000000inv1",
    email: "pending@example.com",
    role: "member",
    expires_at: "2026-05-12T00:00:00Z",
    created_at: "2026-05-06T00:00:00Z",
  },
];

const demoDiagnostics: InstanceDiagnosticsRecord = {
  health: { db: true, storage: true, email: true },
  version: { version: "dev", commit: "test", build_date: "local" },
  migrations: { current: 4, pending: [] },
  master_key_rotation: { phase: "done", rows_done: 0, rows_total: 0 },
  storage: {
    kind: "local-disk",
    endpoint: "file://****/cypra-storage",
    credentials_present: true,
  },
};

function formatDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" });
}

// redirectToPostSetup is the "you can't be here anymore" landing for setup
// pages. If we have a session it sends the user to the dashboard; otherwise
// to the instance-admin login.
async function redirectToPostSetup(): Promise<void> {
  try {
    const response = await fetch("/api/v1/users/me");
    if (response.ok) {
      window.location.assign("/dashboard");
      return;
    }
  } catch {
    // fall through to /login
  }
  window.location.assign("/login");
}

function PasskeyEnrollingState() {
  return (
    <div
      className="flex items-center gap-3 rounded-[var(--radius-md)] border border-border-subtle bg-bg-canvas px-4 py-3 text-[14px] text-text-primary"
      role="status"
      aria-live="polite"
    >
      <span
        aria-hidden="true"
        className="inline-block h-4 w-4 animate-spin rounded-full border-2 border-border-default border-t-accent-primary"
      />
      <span>
        Follow your browser&apos;s passkey prompt, then keep this page open while we save your
        instance admin.
      </span>
    </div>
  );
}

type SetupWizardStep = "admin" | "passkey" | "backup";

export function SetupWizard({ token, step }: { token: string; step: SetupWizardStep }) {
  const [email, setEmail] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [codes, setCodes] = useState<string[]>([]);
  const [codesSaved, setCodesSaved] = useState(false);
  const [status, setStatus] = useState<"checking" | "valid" | "invalid">("checking");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const navigateStep = useCallback(
    (next: SetupWizardStep) => {
      const path = next === "admin" ? `/setup/${token}` : `/setup/${token}/${next}`;
      history.pushState(null, "", path);
      window.dispatchEvent(new PopStateEvent("popstate"));
    },
    [token],
  );

  // Token validation runs on every visit to /setup/<token>/admin or /passkey.
  // Once we've completed the ceremony the token is consumed; we should redirect
  // out of those steps rather than show "invalid".
  useEffect(() => {
    if (step !== "admin" && step !== "passkey") return;
    let active = true;
    setStatus("checking");
    setError("");
    void verifySetupToken(token)
      .then(() => {
        if (active) setStatus("valid");
      })
      .catch((err: unknown) => {
        if (!active) return;
        setStatus("invalid");
        setError(err instanceof Error ? err.message : "setup.token_invalid");
      });
    return () => {
      active = false;
    };
  }, [token, step]);

  // If a user lands on /backup without having just completed the ceremony
  // (codes only live in component state), bounce them out.
  useEffect(() => {
    if (step !== "backup") return;
    if (codes.length > 0) return;
    void redirectToPostSetup();
  }, [step, codes.length]);

  // Token invalid on /admin or /passkey means setup is already done elsewhere.
  // Send the user where they belong.
  useEffect(() => {
    if (status !== "invalid") return;
    if (step !== "admin" && step !== "passkey") return;
    void redirectToPostSetup();
  }, [status, step]);

  const onCompleteSetup = async () => {
    setBusy(true);
    setError("");
    try {
      const begin = await beginSetupPasskey({ token, email, displayName });
      const credential = await navigator.credentials.create({
        publicKey: publicKeyCredentialOptions(begin.options),
      });
      if (!credential) {
        throw new Error("setup.passkey_cancelled");
      }
      const result = await completeSetup({
        token,
        email,
        displayName,
        ceremonyID: begin.ceremony_id,
        response: credentialToJSON(credential as PublicKeyCredential),
      });
      setCodes(result.backup_codes);
      setCodesSaved(false);
      navigateStep("backup");
    } catch (err) {
      setError(err instanceof Error ? err.message : "setup.complete_failed");
    } finally {
      setBusy(false);
    }
  };
  const openDashboard = () => {
    window.location.assign("/dashboard");
  };
  const stepIndex: Record<SetupWizardStep, number> = {
    admin: 0,
    passkey: 1,
    backup: 2,
  };
  const currentStep = stepIndex[step];
  return (
    <main className="min-h-screen bg-bg-canvas px-4 py-16">
      <h1 className="sr-only">Set up Cypra</h1>
      <div className="mx-auto flex w-full max-w-[560px] flex-col items-center gap-8">
        <SetupWizardSteps current={currentStep} />
        {step === "admin" ? (
          <>
            <header className="grid gap-3 text-center">
              <h2 className="text-[32px] font-semibold leading-[1.25] text-text-primary">
                Initialize this Cypra instance
              </h2>
              <p className="text-[14px] leading-[1.571] text-text-secondary">
                Name your first instance admin. We&apos;ll enroll a passkey and save backup codes in
                the next steps.
              </p>
            </header>
            <div className="grid w-full gap-4 rounded-[var(--radius-lg)] border border-border-subtle bg-bg-surface p-6">
              {status === "checking" ? <LoadingState /> : null}
              {status === "valid" && error ? <Toast variant="error" message={error} /> : null}
              <TextInput
                label="Admin email"
                type="email"
                placeholder="admin@example.com"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
              />
              <TextInput
                label="Display name"
                placeholder="Admin"
                value={displayName}
                onChange={(event) => setDisplayName(event.target.value)}
              />
              <div className="flex justify-end">
                <Button
                  variant="primary"
                  disabled={status !== "valid" || email.trim() === ""}
                  onClick={() => navigateStep("passkey")}
                >
                  Continue
                </Button>
              </div>
            </div>
          </>
        ) : null}
        {step === "passkey" ? (
          <>
            <header className="grid gap-3 text-center">
              <h2 className="text-[32px] font-semibold leading-[1.25] text-text-primary">
                Enroll your passkey
              </h2>
              <p className="text-[14px] leading-[1.571] text-text-secondary">
                Touch your security key or use your fingerprint to enroll a passkey for the instance
                admin you just named.
              </p>
            </header>
            <div className="grid w-full gap-4 rounded-[var(--radius-lg)] border border-border-subtle bg-bg-surface p-6">
              {busy ? (
                <PasskeyEnrollingState />
              ) : (
                <>
                  <div className="flex items-center gap-3 text-[14px] text-text-secondary">
                    <PasskeyGlyph className="h-8 w-8 flex-shrink-0 text-accent-primary" />
                    Use a security key or platform authenticator.
                  </div>
                  {error ? <Toast variant="error" message={error} /> : null}
                </>
              )}
              <div className="flex justify-end gap-2">
                <Button variant="ghost" disabled={busy} onClick={() => navigateStep("admin")}>
                  Back
                </Button>
                <Button variant="primary" disabled={busy} onClick={() => void onCompleteSetup()}>
                  {busy ? "Working…" : "Enroll passkey"}
                </Button>
              </div>
            </div>
          </>
        ) : null}
        {step === "backup" ? (
          <>
            <header className="grid gap-3 text-center">
              <h2 className="text-[32px] font-semibold leading-[1.25] text-text-primary">
                Save your backup codes
              </h2>
              <p className="text-[14px] leading-[1.571] text-text-secondary">
                Each code logs you in once if you lose your passkey. Store them somewhere safe.
              </p>
            </header>
            <div className="grid w-full gap-4">
              <BackupCodeGrid codes={codes} onConfirmedChange={setCodesSaved} />
              <div className="flex justify-end">
                <Button variant="primary" disabled={!codesSaved} onClick={openDashboard}>
                  Go to dashboard
                </Button>
              </div>
            </div>
          </>
        ) : null}
      </div>
    </main>
  );
}

function SetupWizardSteps({ current }: { current: number }) {
  const steps = ["Admin", "Passkey", "Backup codes"];
  return (
    <ol aria-label="Setup steps" className="flex items-center gap-2 text-[13px] font-medium">
      {steps.map((label, index) => {
        const isActive = index === current;
        const isComplete = index < current;
        return (
          <li key={label} className="flex items-center gap-2">
            <span
              aria-current={isActive ? "step" : undefined}
              className={cn(
                "flex h-6 w-6 items-center justify-center rounded-full border text-[12px] font-semibold",
                isActive
                  ? "border-transparent bg-accent-primary text-bg-canvas"
                  : isComplete
                    ? "border-transparent bg-accent-primary-mu text-accent-primary"
                    : "border-border-default bg-bg-code text-text-tertiary",
              )}
            >
              {index + 1}
            </span>
            <span
              className={cn(
                isActive
                  ? "text-accent-primary"
                  : isComplete
                    ? "text-text-primary"
                    : "text-text-tertiary",
              )}
            >
              {label}
            </span>
            {index < steps.length - 1 ? (
              <span aria-hidden="true" className="ml-1 h-px w-8 bg-border-default" />
            ) : null}
          </li>
        );
      })}
    </ol>
  );
}

export function DashboardOverview() {
  const forcedState = galleryState();
  const isForcedDemo = forcedState === "demo";
  const isForcedLoading = forcedState === "loading";
  const isForcedError = forcedState === "error";
  const isForcedEmpty = forcedState === "empty";
  const isForcedNoPerm = forcedState === "no-permissions";
  const queriesEnabled = !isForcedDemo && !isForcedLoading && !isForcedError && !isForcedNoPerm;

  const [createTenantOpen, setCreateTenantOpen] = useState(false);
  const summaryQuery = useQuery({
    queryKey: ["instance-summary"],
    queryFn: getDashboardSummary,
    retry: false,
    enabled: queriesEnabled,
  });

  const demoSummary: DashboardSummaryRecord = {
    tenants: 1,
    projects: 2,
    users: 128,
    active_signing_key: true,
    recent_audit: [
      { action: "bootstrapped", resource_id: "instance_admin_root" },
      { action: "created", resource_id: "tenant_acme" },
    ],
  };
  const summary = isForcedDemo ? demoSummary : summaryQuery.data;
  const summaryLoading = isForcedLoading || (summaryQuery.isLoading && queriesEnabled);
  const summaryError =
    isForcedError ||
    (summaryQuery.isError && !summaryQuery.error.message.includes("auth.forbidden"));
  const summaryForbidden = isForcedNoPerm || summaryQuery.error?.message === "auth.forbidden";

  const tenantsCount = summary?.tenants ?? 0;
  const projectsCount = summary?.projects ?? 0;
  const usersCount = summary?.users ?? 0;
  const isEmpty =
    isForcedEmpty ||
    (queriesEnabled && !summaryLoading && !summaryError && !summaryForbidden && tenantsCount === 0);

  const recentAudit = summary?.recent_audit ?? [];

  return (
    <div className="grid gap-6">
      <PageHeader
        title="Overview"
        subtitle="Cypra install at a glance."
        action={
          summaryForbidden ? null : (
            <Button
              variant="primary"
              data-primary-create="true"
              onClick={() => setCreateTenantOpen(true)}
            >
              Create tenant
            </Button>
          )
        }
      />
      {summaryForbidden ? (
        <NoPermissionsCard />
      ) : isEmpty ? (
        <PostBootstrapEmptyCard
          title="No tenants yet"
          body="A tenant scopes users, projects, and signing keys. Create your first tenant to begin."
          ctaLabel="Create your first tenant"
          onCreate={() => setCreateTenantOpen(true)}
        />
      ) : (
        <>
          <InstanceTilesGrid
            loading={summaryLoading}
            error={summaryError}
            tenants={tenantsCount}
            projects={projectsCount}
            users={usersCount}
            activeSigningKey={summary?.active_signing_key ?? false}
            onRetry={() => void summaryQuery.refetch()}
          />
          <InstanceRecentActivityCard
            loading={summaryLoading}
            error={summaryError}
            entries={recentAudit}
            onRetry={() => void summaryQuery.refetch()}
          />
        </>
      )}
      <CreateTenantModal
        open={createTenantOpen}
        onClose={() => setCreateTenantOpen(false)}
        onSuccess={(tenant) => window.location.assign(`/dashboard/tenants/${tenant.slug}`)}
      />
    </div>
  );
}

function InstanceTilesGrid({
  loading,
  error,
  tenants,
  projects,
  users,
  activeSigningKey,
  onRetry,
}: {
  loading: boolean;
  error: boolean;
  tenants: number;
  projects: number;
  users: number;
  activeSigningKey: boolean;
  onRetry: () => void;
}) {
  if (error) {
    return (
      <div className="rounded-[var(--radius-md)] border border-border-subtle bg-bg-surface px-5 py-6">
        <div className="flex items-center justify-between gap-3 text-[13px]">
          <span className="flex items-center gap-2 text-text-primary">
            <Icons.XCircle className="h-4 w-4 text-status-error" aria-hidden="true" />
            Couldn&apos;t load install summary.
          </span>
          <button
            type="button"
            onClick={onRetry}
            className="text-[12px] text-accent-primary hover:underline"
          >
            Try again
          </button>
        </div>
      </div>
    );
  }
  return (
    <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
      <StatTile
        icon={<Icons.SquareDot className="h-3.5 w-3.5" aria-hidden="true" />}
        label="TENANTS"
        loading={loading}
        value={String(tenants)}
        caption="Instance scope"
      />
      <StatTile
        icon={<Icons.Folder className="h-3.5 w-3.5" aria-hidden="true" />}
        label="PROJECTS"
        loading={loading}
        value={String(projects)}
        caption="Across all tenants"
      />
      <StatTile
        icon={<Icons.Users className="h-3.5 w-3.5" aria-hidden="true" />}
        label="USERS"
        loading={loading}
        value={String(users)}
        caption="Tenant-scoped identities"
      />
      <StatTile
        icon={<Icons.KeyRound className="h-3.5 w-3.5" aria-hidden="true" />}
        label="SIGNING KEYS"
        loading={loading}
        valueNode={
          <span className="flex items-center gap-1.5 text-[24px] font-semibold leading-none">
            <span
              aria-hidden="true"
              className={cn(
                "h-2 w-2 rounded-full",
                activeSigningKey ? "bg-status-success" : "bg-text-tertiary",
              )}
            />
            {activeSigningKey ? "Healthy" : "None"}
          </span>
        }
        caption={
          activeSigningKey
            ? "At least one tenant has an active key"
            : "Mint a tenant to issue signing keys"
        }
      />
    </div>
  );
}

function InstanceRecentActivityCard({
  loading,
  error,
  entries,
  onRetry,
}: {
  loading: boolean;
  error: boolean;
  entries: { action: string; resource_id: string }[];
  onRetry: () => void;
}) {
  return (
    <section className="rounded-[var(--radius-md)] border border-border-subtle bg-bg-surface">
      <header className="flex items-center justify-between border-b border-border-subtle px-5 py-3">
        <h2 className="text-[14px] font-medium">Recent activity</h2>
        <a
          href="/dashboard/instance/audit"
          className="text-[13px] text-accent-primary hover:underline"
        >
          View all →
        </a>
      </header>
      {loading ? (
        <ul className="grid gap-2 p-4" data-skeleton="true">
          {Array.from({ length: 4 }, (_unused, index) => (
            <li
              key={index}
              aria-hidden="true"
              className="h-3 rounded-[var(--radius-sm)] bg-bg-code"
            />
          ))}
        </ul>
      ) : error ? (
        <div className="flex items-center justify-between gap-3 px-5 py-6 text-[13px]">
          <span className="flex items-center gap-2 text-text-primary">
            <Icons.XCircle className="h-4 w-4 text-status-error" aria-hidden="true" />
            Couldn&apos;t load recent activity.
          </span>
          <button
            type="button"
            onClick={onRetry}
            className="text-[12px] text-accent-primary hover:underline"
          >
            Try again
          </button>
        </div>
      ) : entries.length === 0 ? (
        <p className="px-5 py-6 text-[13px] text-text-tertiary">
          Audit activity appears here after setup or admin changes.
        </p>
      ) : (
        <ul>
          {entries.map((entry, index) => (
            <li
              key={`${entry.action}-${entry.resource_id}-${String(index)}`}
              className={cn(
                "flex items-center gap-4 px-5 py-3 text-[13px]",
                index !== entries.length - 1 && "border-b border-border-subtle",
              )}
            >
              <span className="rounded-[var(--radius-sm)] bg-bg-code px-2 py-0.5 font-mono text-[11px] text-text-secondary">
                {entry.action}
              </span>
              <IdentifierPill value={entry.resource_id} label="Resource ID" />
              <span className="flex-1" />
              <Icons.ChevronRight className="h-3.5 w-3.5 text-text-tertiary" aria-hidden="true" />
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function publicKeyCredentialOptions(envelope: unknown): PublicKeyCredentialCreationOptions {
  const data = envelope as {
    publicKey?: PublicKeyCredentialCreationOptions;
    options?: { publicKey?: PublicKeyCredentialCreationOptions };
  };
  const options =
    data.options?.publicKey ?? data.publicKey ?? (envelope as PublicKeyCredentialCreationOptions);
  options.challenge = base64UrlToBuffer(options.challenge as unknown as string);
  options.user.id = base64UrlToBuffer(options.user.id as unknown as string);
  for (const descriptor of options.excludeCredentials ?? []) {
    descriptor.id = base64UrlToBuffer(descriptor.id as unknown as string);
  }
  return options;
}

function credentialToJSON(credential: PublicKeyCredential) {
  const response = credential.response as AuthenticatorAttestationResponse;
  return {
    id: credential.id,
    rawId: bufferToBase64Url(credential.rawId),
    type: credential.type,
    authenticatorAttachment: credential.authenticatorAttachment,
    clientExtensionResults: credential.getClientExtensionResults(),
    response: {
      clientDataJSON: bufferToBase64Url(response.clientDataJSON),
      attestationObject: bufferToBase64Url(response.attestationObject),
      transports: response.getTransports(),
    },
  };
}

function base64UrlToBuffer(value: string): ArrayBuffer {
  const padded =
    value.replaceAll("-", "+").replaceAll("_", "/") + "===".slice((value.length + 3) % 4);
  const binary = atob(padded);
  const bytes = new Uint8Array(binary.length);
  for (let index = 0; index < binary.length; index += 1) bytes[index] = binary.charCodeAt(index);
  return bytes.buffer;
}

function bufferToBase64Url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = "";
  for (const byte of bytes) binary += String.fromCharCode(byte);
  return btoa(binary).replaceAll("+", "-").replaceAll("/", "_").replace(/=+$/, "");
}

export function TenantList() {
  const forcedState = galleryState();
  const [query, setQuery] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const tenantsQuery = useQuery({
    queryKey: ["tenants"],
    queryFn: listTenants,
    retry: false,
    enabled: forcedState !== "demo" && forcedState !== "loading" && forcedState !== "error",
  });
  const tenants = forcedState === "demo" ? demoTenants : (tenantsQuery.data ?? []);
  const visibleTenants = useMemo(() => {
    const normalized = query.trim().toLowerCase();
    return tenants
      .filter((tenant) =>
        normalized === ""
          ? true
          : `${tenant.slug} ${tenant.name}`.toLowerCase().includes(normalized),
      )
      .sort((a, b) => a.slug.localeCompare(b.slug));
  }, [query, tenants]);

  if (forcedState === "permission-denied" || tenantsQuery.error?.message === "auth.forbidden") {
    return <ErrorPage code="403" />;
  }
  if (forcedState === "loading" || tenantsQuery.isLoading) return <LoadingState />;
  if (forcedState === "error" || tenantsQuery.isError) {
    return (
      <ErrorState
        title="Couldn't load tenants."
        body="The tenant list request failed. Retry after checking your instance-admin session."
        retry={() => void tenantsQuery.refetch()}
      />
    );
  }

  const isEmpty = forcedState === "empty" || tenants.length === 0;
  const isSearchEmpty = !isEmpty && visibleTenants.length === 0;

  const headerAction = isEmpty ? null : (
    <div className="flex flex-wrap items-center gap-3">
      <TenantSearchInput value={query} onChange={setQuery} />
      <Button data-primary-create="true" variant="primary" onClick={() => setCreateOpen(true)}>
        Create tenant
      </Button>
    </div>
  );

  return (
    <div className="grid gap-6">
      <PageHeader
        title="Tenants"
        subtitle="All tenants on this Cypra install."
        action={headerAction}
      />
      {isEmpty ? (
        <div className="grid place-items-center gap-3 rounded-[var(--radius-md)] border border-border-subtle bg-bg-surface p-12 text-center">
          <Icons.SquareDot className="h-6 w-6 text-accent-primary" aria-hidden="true" />
          <h2 className="text-[20px] font-semibold leading-[1.4]">No tenants yet</h2>
          <p className="max-w-[420px] text-[14px] leading-[1.571] text-text-secondary">
            A tenant scopes users, projects, and signing keys. Create your first tenant.
          </p>
          <Button variant="primary" onClick={() => setCreateOpen(true)}>
            Create tenant
          </Button>
        </div>
      ) : (
        <>
          <div className="overflow-hidden rounded-[var(--radius-md)] border border-border-subtle bg-bg-surface">
            <table className="w-full border-collapse text-left text-[13px]" data-responsive="stack">
              <thead className="bg-bg-code text-text-tertiary">
                <tr>
                  <th className="w-[280px] px-4 py-2.5 font-medium">Tenant</th>
                  <th className="w-[200px] px-4 py-2.5 font-medium">Slug</th>
                  <th className="w-[100px] px-4 py-2.5 font-medium">Members</th>
                  <th className="px-4 py-2.5 font-medium">Created</th>
                  <th className="w-12 px-4 py-2.5 font-medium" aria-hidden="true" />
                </tr>
              </thead>
              <tbody>
                {isSearchEmpty ? (
                  <tr>
                    <td colSpan={5} className="border-t border-border-subtle px-4 py-12">
                      <SearchEmpty query={query} onClear={() => setQuery("")} />
                    </td>
                  </tr>
                ) : (
                  visibleTenants.map((tenant) => (
                    <tr
                      key={tenant.id}
                      className="border-t border-border-subtle hover:bg-bg-elevated"
                    >
                      <td className="px-4 py-2">
                        <a
                          className="flex items-center gap-2.5 font-medium text-text-primary hover:text-accent-primary"
                          href={`/dashboard/tenants/${tenant.slug}`}
                        >
                          <Avatar name={tenant.name} sub={tenant.id} size="sm" />
                          <span>{tenant.name}</span>
                        </a>
                      </td>
                      <td className="px-4 py-2">
                        <IdentifierPill value={tenant.slug} label="Tenant slug" />
                      </td>
                      <td className="px-4 py-2 text-text-secondary">{tenant.member_count ?? 0}</td>
                      <td className="px-4 py-2 font-mono text-text-tertiary">
                        {formatTenantCreatedAt(tenant.created_at)}
                      </td>
                      <td className="px-4 py-2 text-right">
                        <IconButton
                          label={`Open ${tenant.slug}`}
                          icon={<Icons.ChevronRight className="h-4 w-4" />}
                          variant="ghost"
                          size="sm"
                          onClick={() =>
                            window.location.assign(`/dashboard/tenants/${tenant.slug}`)
                          }
                        />
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
          <div className="flex items-center justify-between px-1 text-[12px] text-text-tertiary">
            <span>
              Showing {String(visibleTenants.length)} of {String(tenants.length)} tenant
              {tenants.length === 1 ? "" : "s"}
            </span>
            <div className="flex items-center gap-1">
              <IconButton
                label="Previous page"
                icon={<Icons.ChevronLeft className="h-4 w-4" />}
                variant="ghost"
                size="sm"
                disabled
              />
              <IconButton
                label="Next page"
                icon={<Icons.ChevronRight className="h-4 w-4" />}
                variant="ghost"
                size="sm"
                disabled
              />
            </div>
          </div>
        </>
      )}
      <CreateTenantModal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        onSuccess={(tenant) => window.location.assign(`/dashboard/tenants/${tenant.slug}`)}
      />
    </div>
  );
}

function TenantSearchInput({
  value,
  onChange,
}: {
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <div
      className={cn(
        "flex h-9 items-center gap-2 rounded-[var(--radius-md)] border border-border-default bg-bg-surface px-3 text-[13px]",
        "focus-within:border-border-focus focus-within:outline-2 focus-within:outline-offset-[-1px] focus-within:outline-border-focus",
      )}
    >
      <Icons.Search
        className={cn("h-3.5 w-3.5", value ? "text-text-secondary" : "text-text-tertiary")}
        aria-hidden="true"
      />
      <input
        type="search"
        aria-label="Search tenants"
        placeholder="Search tenants"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="w-56 bg-transparent text-text-primary placeholder:text-text-tertiary focus:outline-none"
      />
      {value ? (
        <button
          type="button"
          aria-label="Clear search"
          onClick={() => onChange("")}
          className="rounded-[var(--radius-sm)] p-0.5 text-text-tertiary hover:text-text-primary"
        >
          <Icons.X className="h-3.5 w-3.5" aria-hidden="true" />
        </button>
      ) : null}
    </div>
  );
}

function SearchEmpty({ query, onClear }: { query: string; onClear: () => void }) {
  return (
    <div className="grid place-items-center gap-3 text-center">
      <Icons.SearchX className="h-6 w-6 text-text-tertiary" aria-hidden="true" />
      <p className="flex flex-wrap items-center justify-center gap-2 text-[14px] font-medium text-text-primary">
        No tenants match <IdentifierPill value={query} label="Search query" />.
      </p>
      <p className="text-[13px] text-text-secondary">Try a shorter query or check your spelling.</p>
      <Button variant="primary" onClick={onClear}>
        Clear search
      </Button>
    </div>
  );
}

export function TenantDetail({
  slug,
  tab,
  settingsTab,
  authProviderTab,
  detailSlug,
}: {
  slug: string;
  tab: string;
  settingsTab?: string;
  authProviderTab?: string;
  detailSlug?: string;
}) {
  const tenantsQuery = useQuery({ queryKey: ["tenants"], queryFn: listTenants, retry: false });
  const tenant =
    tenantsQuery.data?.find((item) => item.slug === slug) ??
    (demoStateEnabled() ? demoTenants.find((item) => item.slug === slug) : undefined);
  const currentTab = tab === "overview" ? "overview" : tab;
  if (tenantsQuery.error?.message === "auth.forbidden") return <ErrorPage code="403" />;
  if (tenantsQuery.isLoading && !demoStateEnabled()) return <LoadingState />;
  if (tenantsQuery.isError && !demoStateEnabled()) {
    return (
      <ErrorState
        title="Couldn't load tenant."
        body="The tenant request failed. Retry after checking your session and permissions."
        retry={() => void tenantsQuery.refetch()}
      />
    );
  }
  if (!tenant) {
    return (
      <ErrorState title="Tenant not found." body="This tenant does not exist or is not visible." />
    );
  }
  return (
    <div className="grid gap-6">
      {currentTab === "overview" ? null : <TenantHeader tenant={tenant} />}
      {currentTab === "overview" ? <TenantOverview tenant={tenant} /> : null}
      {currentTab === "projects" ? (
        detailSlug ? (
          <ProjectDetail tenant={tenant} projectSlug={detailSlug} />
        ) : (
          <ProjectList tenant={tenant} />
        )
      ) : null}
      {currentTab === "users" ? (
        detailSlug ? (
          <UserDetail tenant={tenant} userId={detailSlug} />
        ) : (
          <UserList tenant={tenant} />
        )
      ) : null}
      {currentTab === "auth-providers" ? (
        <AuthProvidersScreen tenant={tenant} active={authProviderTab ?? "overview"} />
      ) : null}
      {currentTab === "signing-keys" ? <SigningKeysScreen /> : null}
      {currentTab === "audit" ? <AuditLogScreen scope="tenant" /> : null}
      {currentTab === "settings" ? (
        <TenantSettings tenant={tenant} active={settingsTab ?? "branding"} />
      ) : null}
      {currentTab !== "overview" &&
      currentTab !== "projects" &&
      currentTab !== "users" &&
      currentTab !== "auth-providers" &&
      currentTab !== "signing-keys" &&
      currentTab !== "audit" &&
      currentTab !== "settings" ? (
        <PlaceholderPage title={tenantTabTitle(currentTab)} />
      ) : null}
    </div>
  );
}

function TenantHeader({ tenant }: { tenant: TenantRecord }) {
  return (
    <PageHeader
      title={tenant.branding?.display_name ?? tenant.name}
      subtitle="Configure tenant-scoped projects, users, auth methods, signing keys, audit, and settings."
      action={
        <div className="flex flex-wrap items-center gap-2">
          <IdentifierPill value={tenant.slug} label="Tenant slug" />
          <Tag variant="context-tenant">{tenant.member_count ?? 0} members</Tag>
        </div>
      }
    />
  );
}

function TenantOverview({ tenant }: { tenant: TenantRecord }) {
  const forcedState = galleryState();
  const isForcedDemo = forcedState === "demo";
  const isForcedLoading = forcedState === "loading";
  const isForcedError = forcedState === "error";
  const isForcedEmpty = forcedState === "empty";
  const isForcedNoPerm = forcedState === "no-permissions";
  const isForcedTilesError = forcedState === "tiles-error";
  const isForcedTilesPermDenied = forcedState === "tiles-permission-denied";
  const queriesEnabled =
    !isForcedDemo && !isForcedLoading && !isForcedError && !isForcedNoPerm && !isForcedTilesError;

  const [createProjectOpen, setCreateProjectOpen] = useState(false);

  const signingKeysQuery = useQuery({
    queryKey: ["overview-signing-keys", tenant.slug],
    queryFn: listSigningKeys,
    retry: false,
    enabled: queriesEnabled,
  });
  const authMethodsQuery = useQuery({
    queryKey: ["overview-auth-methods", tenant.slug],
    queryFn: listAuthMethods,
    retry: false,
    enabled: queriesEnabled,
  });
  const auditQuery = useQuery({
    queryKey: ["overview-audit", tenant.slug],
    queryFn: () => listAuditEntries("tenant"),
    retry: false,
    enabled: queriesEnabled,
  });

  const signingKeys = isForcedDemo ? demoSigningKeys : signingKeysQuery.data;
  const authMethods = isForcedDemo ? demoAuthMethods : authMethodsQuery.data;
  const audit = isForcedDemo ? demoAuditEntries("tenant") : auditQuery.data;

  const skLoading = isForcedLoading || (signingKeysQuery.isLoading && queriesEnabled);
  const skError =
    isForcedError ||
    isForcedTilesError ||
    (signingKeysQuery.isError && !signingKeysQuery.error.message.includes("auth.forbidden"));
  const skForbidden =
    isForcedTilesPermDenied || signingKeysQuery.error?.message === "auth.forbidden";
  const amLoading = isForcedLoading || (authMethodsQuery.isLoading && queriesEnabled);
  const amError =
    isForcedError ||
    isForcedTilesError ||
    (authMethodsQuery.isError && authMethodsQuery.error.message !== "auth.forbidden");
  const amForbidden =
    isForcedTilesPermDenied || authMethodsQuery.error?.message === "auth.forbidden";
  const auLoading = isForcedLoading || (auditQuery.isLoading && queriesEnabled);
  const auError =
    isForcedError ||
    isForcedTilesError ||
    (auditQuery.isError && auditQuery.error.message !== "auth.forbidden");
  const auForbidden = isForcedTilesPermDenied || auditQuery.error?.message === "auth.forbidden";

  const noPermissions =
    isForcedNoPerm ||
    (signingKeysQuery.error?.message === "auth.forbidden" &&
      authMethodsQuery.error?.message === "auth.forbidden" &&
      auditQuery.error?.message === "auth.forbidden");

  const projectCount = tenant.project_count ?? 0;
  const userCount = tenant.user_count ?? 0;
  const activeKey = signingKeys?.find((key) => key.state === "active");
  const enabledMethods = authMethods?.filter((method) => method.enabled) ?? [];
  const recentAudit = (audit ?? []).slice(0, 8);

  const isEmpty =
    isForcedEmpty ||
    (queriesEnabled &&
      !skLoading &&
      !amLoading &&
      !auLoading &&
      !skError &&
      !amError &&
      !auError &&
      projectCount === 0 &&
      userCount === 0 &&
      recentAudit.length === 0);

  const subtitle = (() => {
    const parts: string[] = [];
    parts.push(`${String(projectCount)} project${projectCount === 1 ? "" : "s"}`);
    parts.push(`${String(userCount)} user${userCount === 1 ? "" : "s"}`);
    if (skLoading) {
      parts.push("checking signing keys…");
    } else if (skForbidden) {
      parts.push("signing keys hidden");
    } else if (skError) {
      parts.push("signing keys unavailable");
    } else {
      parts.push(activeKey ? "1 active signing key" : "no active signing key");
    }
    return parts.join(" · ");
  })();

  const headerActions = (
    <div className="flex items-center gap-2">
      <a
        href={`/dashboard/tenants/${tenant.slug}/audit`}
        className="inline-flex h-9 items-center gap-2 rounded-[var(--radius-md)] border border-border-default bg-bg-surface px-4 text-[14px] font-medium text-text-primary transition duration-[var(--dur-instant)] hover:bg-bg-elevated focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-border-focus"
      >
        View audit log
      </a>
      <Button
        variant="primary"
        data-primary-create="true"
        onClick={() => setCreateProjectOpen(true)}
      >
        Create project
      </Button>
    </div>
  );

  return (
    <div className="grid gap-6">
      <PageHeader
        title={tenant.branding?.display_name ?? tenant.name}
        subtitle={subtitle}
        action={headerActions}
      />
      {tenant.email_provider_required ? (
        <Card
          highlighted
          title="Email provider required"
          subtitle="This tenant cannot issue magic links, password resets, verifications, or admin invites until email is configured."
        >
          <a
            className="text-[14px] font-medium text-accent-primary"
            href={`/dashboard/tenants/${tenant.slug}/settings/email`}
          >
            Configure email provider
          </a>
        </Card>
      ) : null}
      {noPermissions ? (
        <NoPermissionsCard />
      ) : isEmpty ? (
        <PostBootstrapEmptyCard onCreate={() => setCreateProjectOpen(true)} />
      ) : (
        <>
          <OverviewTilesGrid
            projects={{
              loading: skLoading || amLoading,
              count: projectCount,
            }}
            users={{
              loading: auLoading || amLoading,
              count: userCount,
            }}
            signingKey={{
              loading: skLoading,
              error: skError,
              forbidden: skForbidden,
              active: !!activeKey,
              nextRotation: activeKey?.retires_at,
              retry: () => void signingKeysQuery.refetch(),
            }}
            authMethods={{
              loading: amLoading,
              error: amError,
              forbidden: amForbidden,
              count: enabledMethods.length,
              names: enabledMethods.map((method) => authMethodLabel(method.method)),
              retry: () => void authMethodsQuery.refetch(),
            }}
          />
          {(skForbidden || amForbidden || auForbidden) &&
          !(skForbidden && amForbidden && auForbidden) ? (
            <PermissionDeniedHint />
          ) : null}
          <RecentActivityCard
            tenantSlug={tenant.slug}
            loading={auLoading}
            error={auError}
            forbidden={auForbidden}
            entries={recentAudit}
            retry={() => void auditQuery.refetch()}
          />
        </>
      )}
      <CreateProjectModal
        open={createProjectOpen}
        onClose={() => setCreateProjectOpen(false)}
        tenantSlug={tenant.slug}
        onSuccess={(project) =>
          window.location.assign(`/dashboard/tenants/${tenant.slug}/projects/${project.slug}`)
        }
      />
    </div>
  );
}

function authMethodLabel(method: AuthMethodRecord["method"]): string {
  switch (method) {
    case "password":
      return "Password";
    case "magic_link":
      return "Magic";
    case "passkey":
      return "Passkey";
    case "totp":
      return "TOTP";
    case "google":
      return "Google";
    case "oidc_upstream":
      return "OIDC";
  }
}

interface ProjectsTileProps {
  loading: boolean;
  count: number;
}
interface UsersTileProps {
  loading: boolean;
  count: number;
}
interface SigningKeyTileProps {
  loading: boolean;
  error: boolean;
  forbidden: boolean;
  active: boolean;
  nextRotation?: string;
  retry: () => void;
}
interface AuthMethodsTileProps {
  loading: boolean;
  error: boolean;
  forbidden: boolean;
  count: number;
  names: string[];
  retry: () => void;
}

function OverviewTilesGrid({
  projects,
  users,
  signingKey,
  authMethods,
}: {
  projects: ProjectsTileProps;
  users: UsersTileProps;
  signingKey: SigningKeyTileProps;
  authMethods: AuthMethodsTileProps;
}) {
  return (
    <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
      <StatTile
        icon={<Icons.Folder className="h-3.5 w-3.5" aria-hidden="true" />}
        label="PROJECTS"
        loading={projects.loading}
        value={String(projects.count)}
        caption="OIDC clients"
      />
      <StatTile
        icon={<Icons.Users className="h-3.5 w-3.5" aria-hidden="true" />}
        label="USERS"
        loading={users.loading}
        value={String(users.count)}
        caption="Tenant identities"
      />
      {signingKey.forbidden ? (
        <PermissionDeniedTile label="SIGNING KEY" needsScope="signing_keys.read" />
      ) : signingKey.error ? (
        <ErrorTile label="signing keys" onRetry={signingKey.retry} />
      ) : (
        <StatTile
          icon={<Icons.KeyRound className="h-3.5 w-3.5" aria-hidden="true" />}
          label="SIGNING KEY"
          loading={signingKey.loading}
          valueNode={
            <span className="flex items-center gap-1.5 text-[18px] font-semibold leading-none">
              <span
                aria-hidden="true"
                className={cn(
                  "h-2 w-2 rounded-full",
                  signingKey.active ? "bg-status-success" : "bg-text-tertiary",
                )}
              />
              {signingKey.active ? "Active" : "None"}
            </span>
          }
          caption={
            signingKey.active
              ? signingKey.nextRotation
                ? `Next rotation ${formatDaysFrom(signingKey.nextRotation)}`
                : "Rotation not scheduled"
              : "Mint one to start issuing tokens"
          }
        />
      )}
      {authMethods.forbidden ? (
        <PermissionDeniedTile label="AUTH METHODS" needsScope="auth_methods.read" />
      ) : authMethods.error ? (
        <ErrorTile label="auth methods" onRetry={authMethods.retry} />
      ) : (
        <StatTile
          icon={<Icons.ShieldCheck className="h-3.5 w-3.5" aria-hidden="true" />}
          label="AUTH METHODS"
          loading={authMethods.loading}
          value={String(authMethods.count)}
          caption={authMethods.names.length > 0 ? authMethods.names.join(" · ") : "None enabled"}
        />
      )}
    </div>
  );
}

function StatTile({
  icon,
  label,
  loading,
  value,
  valueNode,
  caption,
}: {
  icon: ReactNode;
  label: string;
  loading: boolean;
  value?: string;
  valueNode?: ReactNode;
  caption?: string;
}) {
  return (
    <article className="flex flex-col gap-2 rounded-[var(--radius-md)] border border-border-subtle bg-bg-surface p-5">
      <div className="flex items-center gap-2 text-text-tertiary">
        {icon}
        <span className="text-[11px] font-semibold tracking-[0.06em]">{label}</span>
      </div>
      {loading ? (
        <span
          aria-hidden="true"
          className="h-8 w-16 rounded-[var(--radius-sm)] bg-bg-code"
          data-skeleton="true"
        />
      ) : valueNode !== undefined ? (
        valueNode
      ) : (
        <span className="text-[32px] font-semibold leading-none">{value}</span>
      )}
      {caption ? <p className="text-[12px] text-text-tertiary">{caption}</p> : null}
    </article>
  );
}

function ErrorTile({ label, onRetry }: { label: string; onRetry: () => void }) {
  return (
    <article
      role="alert"
      className="flex flex-col items-center justify-center gap-1.5 rounded-[var(--radius-md)] border border-border-subtle bg-bg-surface p-5 text-center"
    >
      <Icons.XCircle className="h-4 w-4 text-status-error" aria-hidden="true" />
      <p className="text-[13px] font-medium text-text-primary">Couldn&apos;t load {label}</p>
      <button
        type="button"
        onClick={onRetry}
        className="text-[12px] text-accent-primary hover:underline"
      >
        Try again
      </button>
    </article>
  );
}

function PermissionDeniedTile({ label, needsScope }: { label: string; needsScope: string }) {
  return (
    <article className="flex flex-col gap-1.5 rounded-[var(--radius-md)] border border-border-subtle bg-bg-surface p-5 opacity-50">
      <div className="flex items-center gap-2 text-text-tertiary">
        <Icons.Lock className="h-3.5 w-3.5" aria-hidden="true" />
        <span className="text-[11px] font-semibold tracking-[0.06em]">{label}</span>
      </div>
      <span aria-hidden="true" className="text-[24px] text-text-disabled">
        ——
      </span>
      <p className="text-[11px] text-text-tertiary">Needs {needsScope}</p>
    </article>
  );
}

function PermissionDeniedHint() {
  return (
    <div className="flex items-center gap-2 rounded-[var(--radius-md)] border border-border-emphasis bg-bg-elevated px-3 py-2.5 text-[12px] text-text-secondary">
      <Icons.Lock className="h-3.5 w-3.5 text-text-tertiary" aria-hidden="true" />
      <span>Some tiles are hidden. Ask the tenant owner for the missing scopes.</span>
    </div>
  );
}

function NoPermissionsCard() {
  return (
    <div className="grid place-items-center gap-3 rounded-[var(--radius-md)] border border-border-subtle bg-bg-surface px-8 py-16 text-center">
      <Icons.Shield className="h-6 w-6 text-text-tertiary" aria-hidden="true" />
      <h2 className="text-[20px] font-semibold leading-[1.4]">No permissions on this tenant</h2>
      <p className="max-w-[560px] text-[14px] leading-[1.571] text-text-secondary">
        You&apos;re a member of this tenant, but you don&apos;t have permissions to see anything
        yet. Ask the tenant owner.
      </p>
    </div>
  );
}

function PostBootstrapEmptyCard({
  onCreate,
  title = "No projects yet",
  body = "A project maps to one OIDC client. Create one so apps can authenticate against this tenant.",
  ctaLabel = "Create your first project",
}: {
  onCreate: () => void;
  title?: string;
  body?: string;
  ctaLabel?: string;
}) {
  return (
    <div className="grid place-items-center gap-3 rounded-[var(--radius-md)] border border-border-subtle bg-bg-surface p-12 text-center">
      <Icons.SquareDot className="h-6 w-6 text-text-tertiary" aria-hidden="true" />
      <h2 className="text-[20px] font-semibold leading-[1.4]">{title}</h2>
      <p className="max-w-[420px] text-[14px] leading-[1.571] text-text-secondary">{body}</p>
      <Button variant="primary" onClick={onCreate}>
        {ctaLabel}
      </Button>
    </div>
  );
}

function RecentActivityCard({
  tenantSlug,
  loading,
  error,
  forbidden,
  entries,
  retry,
}: {
  tenantSlug: string;
  loading: boolean;
  error: boolean;
  forbidden: boolean;
  entries: AuditEntryRecord[];
  retry: () => void;
}) {
  return (
    <section className="rounded-[var(--radius-md)] border border-border-subtle bg-bg-surface">
      <header className="flex items-center justify-between border-b border-border-subtle px-5 py-3">
        <h2 className="text-[14px] font-medium">Recent activity</h2>
        <div className="flex items-center gap-3 text-[12px] text-text-tertiary">
          <span>Last {String(entries.length || 8)} entries</span>
          <a
            href={`/dashboard/tenants/${tenantSlug}/audit`}
            className="text-[13px] text-accent-primary hover:underline"
          >
            View all →
          </a>
        </div>
      </header>
      {loading ? (
        <ul className="grid gap-2 p-4" data-skeleton="true">
          {Array.from({ length: 4 }, (_unused, index) => (
            <li
              key={index}
              aria-hidden="true"
              className="h-3 rounded-[var(--radius-sm)] bg-bg-code"
            />
          ))}
        </ul>
      ) : forbidden ? (
        <div className="flex items-center gap-2 px-5 py-6 text-[13px] text-text-tertiary">
          <Icons.Lock className="h-3.5 w-3.5" aria-hidden="true" />
          Audit entries are hidden. Needs audit.read.
        </div>
      ) : error ? (
        <div className="flex items-center justify-between gap-3 px-5 py-6 text-[13px]">
          <span className="flex items-center gap-2 text-text-primary">
            <Icons.XCircle className="h-4 w-4 text-status-error" aria-hidden="true" />
            Couldn&apos;t load recent activity.
          </span>
          <button
            type="button"
            onClick={retry}
            className="text-[12px] text-accent-primary hover:underline"
          >
            Try again
          </button>
        </div>
      ) : entries.length === 0 ? (
        <p className="px-5 py-6 text-[13px] text-text-tertiary">No activity in the last window.</p>
      ) : (
        <ul>
          {entries.map((entry, index) => (
            <RecentActivityRow
              key={entry.id}
              entry={entry}
              first={index === 0}
              last={index === entries.length - 1}
            />
          ))}
        </ul>
      )}
    </section>
  );
}

function RecentActivityRow({
  entry,
  last,
}: {
  entry: AuditEntryRecord;
  first: boolean;
  last: boolean;
}) {
  const actorName = audiActorName(entry);
  const actorId = entry.actor_id ?? "00000000-0000-0000-0000-000000000000";
  return (
    <li
      className={cn(
        "flex items-center gap-4 px-5 py-3 text-[13px]",
        !last && "border-b border-border-subtle",
      )}
    >
      <time className="w-20 shrink-0 font-mono text-[12px] text-text-tertiary">
        {formatRelativeTime(entry.occurred_at)}
      </time>
      <Avatar name={actorName} sub={actorId} size="sm" />
      <span className="font-medium text-text-primary">{actorName}</span>
      <span className="rounded-[var(--radius-sm)] bg-bg-code px-2 py-0.5 font-mono text-[11px] text-text-secondary">
        {entry.action}
      </span>
      {entry.resource_id ? (
        <IdentifierPill value={entry.resource_id} label="Resource ID" />
      ) : (
        <span className="text-[12px] text-text-tertiary">{entry.resource_kind}</span>
      )}
      <span className="flex-1" />
      <Icons.ChevronRight className="h-3.5 w-3.5 text-text-tertiary" aria-hidden="true" />
    </li>
  );
}

function audiActorName(entry: AuditEntryRecord): string {
  const explicit = entry.metadata.actor_name;
  if (typeof explicit === "string" && explicit.length > 0) return explicit;
  if (entry.actor_kind === "instance_admin") return "Instance admin";
  if (entry.actor_kind === "tenant_admin") return "Tenant admin";
  if (entry.actor_kind === "tenant_member") return "Tenant member";
  if (entry.actor_kind === "user") return "User";
  return "System";
}

function formatRelativeTime(iso: string): string {
  const ts = Date.parse(iso);
  if (Number.isNaN(ts)) return iso;
  const seconds = Math.max(1, Math.round((Date.now() - ts) / 1000));
  if (seconds < 60) return `${String(seconds)}s ago`;
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) return `${String(minutes)}m ago`;
  const hours = Math.round(minutes / 60);
  if (hours < 24) return `${String(hours)}h ago`;
  const days = Math.round(hours / 24);
  if (days === 1) return "yesterday";
  if (days < 30) return `${String(days)}d ago`;
  const months = Math.round(days / 30);
  if (months < 12) return `${String(months)}mo ago`;
  return `${String(Math.round(months / 12))}y ago`;
}

function formatTenantCreatedAt(iso?: string): string {
  if (!iso) return "—";
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  const yyyy = String(date.getUTCFullYear());
  const mm = String(date.getUTCMonth() + 1).padStart(2, "0");
  const dd = String(date.getUTCDate()).padStart(2, "0");
  return `${yyyy}-${mm}-${dd}`;
}

function formatDaysFrom(iso: string): string {
  const ts = Date.parse(iso);
  if (Number.isNaN(ts)) return "soon";
  const days = Math.round((ts - Date.now()) / (1000 * 60 * 60 * 24));
  if (days <= 0) return "now";
  if (days === 1) return "in 1 day";
  return `in ${String(days)} days`;
}

function ProjectList({ tenant }: { tenant: TenantRecord }) {
  const projectsQuery = useQuery({
    queryKey: ["projects", tenant.slug],
    queryFn: listProjects,
    retry: false,
  });
  const [createOpen, setCreateOpen] = useState(false);
  const projects = demoStateEnabled() ? demoProjects : (projectsQuery.data ?? []);
  if (projectsQuery.error?.message === "auth.forbidden") return <ErrorPage code="403" />;
  if (projectsQuery.isLoading && !demoStateEnabled()) return <LoadingState />;
  if (projectsQuery.isError && !demoStateEnabled()) {
    return (
      <ErrorState
        title="Couldn't load projects."
        body="The project list request failed. Retry after checking tenant access."
        retry={() => void projectsQuery.refetch()}
      />
    );
  }
  return (
    <Card
      title="Projects"
      subtitle="One project maps to one OIDC client in v1."
      actions={
        <Button variant="primary" data-primary-create="true" onClick={() => setCreateOpen(true)}>
          Create project
        </Button>
      }
    >
      {projects.length === 0 ? (
        <EmptyState
          title="No projects yet."
          body="Create a project to let an app authenticate against this tenant."
          action={
            <Button variant="primary" onClick={() => setCreateOpen(true)}>
              Create project
            </Button>
          }
        />
      ) : (
        <div className="overflow-hidden rounded-[var(--radius-md)] border border-border-subtle">
          <table className="w-full text-left text-[13px]" data-responsive="stack">
            <thead className="bg-bg-code text-text-secondary">
              <tr>
                <th className="px-4 py-3 font-medium">Project</th>
                <th className="px-4 py-3 font-medium">Slug</th>
                <th className="px-4 py-3 font-medium">Client</th>
                <th className="px-4 py-3 font-medium">Status</th>
              </tr>
            </thead>
            <tbody>
              {projects.map((project) => (
                <tr key={project.id} className="border-t border-border-subtle">
                  <td className="px-4 py-3">
                    <a
                      className="font-medium hover:text-accent-primary"
                      href={`/dashboard/tenants/${tenant.slug}/projects/${project.slug}`}
                    >
                      {project.name}
                    </a>
                  </td>
                  <td className="px-4 py-3">
                    <IdentifierPill value={project.slug} label="Project slug" />
                  </td>
                  <td className="px-4 py-3">
                    {project.client_id ? (
                      <IdentifierPill value={project.client_id} label="Client ID" />
                    ) : (
                      <Tag variant="warn">Not configured</Tag>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    {project.rotation_in_progress ? (
                      <Tag variant="warn">Rotation in progress</Tag>
                    ) : (
                      <Tag variant="success">Ready</Tag>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <CreateProjectModal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        tenantSlug={tenant.slug}
        onSuccess={(project) =>
          window.location.assign(`/dashboard/tenants/${tenant.slug}/projects/${project.slug}`)
        }
      />
    </Card>
  );
}

function ProjectDetail({ tenant, projectSlug }: { tenant: TenantRecord; projectSlug: string }) {
  const queryClient = useQueryClient();
  const projectsQuery = useQuery({
    queryKey: ["projects", tenant.slug],
    queryFn: listProjects,
    retry: false,
  });
  const project =
    projectsQuery.data?.find((item) => item.slug === projectSlug) ??
    (demoStateEnabled() ? demoProjects.find((item) => item.slug === projectSlug) : undefined);
  const [redirectUris, setRedirectUris] = useState((project?.redirect_uris ?? []).join("\n"));
  const [scopes, setScopes] = useState((project?.allowed_scopes ?? ["openid"]).join(" "));
  const [authMethod, setAuthMethod] = useState(
    project?.token_endpoint_auth_method ?? "client_secret_basic",
  );
  const [rotateOpen, setRotateOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleteValue, setDeleteValue] = useState("");
  const [clientSecret, setClientSecret] = useState<string | undefined>();
  const [saveError, setSaveError] = useState<string | undefined>();
  const [saveDone, setSaveDone] = useState(false);
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    if (!project) return;
    setRedirectUris((project.redirect_uris ?? []).join("\n"));
    setScopes((project.allowed_scopes ?? ["openid"]).join(" "));
    setAuthMethod(project.token_endpoint_auth_method ?? "client_secret_basic");
    setClientSecret(project.client_secret);
  }, [project]);
  if (projectsQuery.error?.message === "auth.forbidden") return <ErrorPage code="403" />;
  if (projectsQuery.isLoading && !demoStateEnabled()) return <LoadingState />;
  if (projectsQuery.isError && !demoStateEnabled()) {
    return (
      <ErrorState
        title="Couldn't load project."
        body="The project request failed. Retry after checking tenant access."
        retry={() => void projectsQuery.refetch()}
      />
    );
  }
  if (!project) {
    return (
      <ErrorState
        title="Project not found."
        body="This project does not exist or is not visible."
      />
    );
  }
  const redirectError = redirectUris
    .split("\n")
    .filter(Boolean)
    .some((uri) => {
      try {
        return new URL(uri).hash !== "";
      } catch {
        return true;
      }
    })
    ? "Redirect URIs must be valid absolute URLs and cannot contain fragments."
    : undefined;
  const issuerURL = project.issuer_url;
  const clientID = project.client_id;
  const saveProject = async () => {
    if (redirectError) return;
    setBusy(true);
    setSaveError(undefined);
    setSaveDone(false);
    try {
      await updateProject({
        id: project.id,
        name: project.name,
        redirect_uris: redirectUris
          .split("\n")
          .map((uri) => uri.trim())
          .filter(Boolean),
        allowed_scopes: scopes
          .split(/\s+/)
          .map((scope) => scope.trim())
          .filter(Boolean),
        token_endpoint_auth_method: authMethod,
      });
      await queryClient.invalidateQueries({ queryKey: ["projects", tenant.slug] });
      setSaveDone(true);
    } catch (error) {
      setSaveError(error instanceof Error ? error.message : "project.update_failed");
    } finally {
      setBusy(false);
    }
  };
  const rotateSecret = async () => {
    setBusy(true);
    setSaveError(undefined);
    try {
      const result = await rotateProjectSecret(project.id);
      setClientSecret(result.client_secret);
      await queryClient.invalidateQueries({ queryKey: ["projects", tenant.slug] });
      setRotateOpen(false);
      setSaveDone(true);
    } catch (error) {
      setSaveError(error instanceof Error ? error.message : "project.secret_rotate_failed");
    } finally {
      setBusy(false);
    }
  };
  const confirmDelete = async () => {
    if (deleteValue !== project.slug) return;
    setBusy(true);
    setSaveError(undefined);
    try {
      await deleteProject(project.id);
      await queryClient.invalidateQueries({ queryKey: ["projects", tenant.slug] });
      window.location.assign(`/dashboard/tenants/${tenant.slug}/projects`);
    } catch (error) {
      setSaveError(error instanceof Error ? error.message : "project.delete_failed");
      setBusy(false);
    }
  };
  return (
    <div className="grid gap-4">
      {saveDone ? <Toast variant="success" message="Project settings saved." /> : null}
      {saveError ? <Toast variant="error" message={saveError} /> : null}
      {project.rotation_in_progress ? (
        <Toast
          variant="warn"
          message="Rotation in progress - downstream apps using the previous secret will fail at the token endpoint after 5 minutes. Coordinate the cutover with your app owner."
        />
      ) : null}
      <Card title={project.name} subtitle="OIDC client configuration for this project.">
        <div className="grid gap-4 lg:grid-cols-2">
          <SettingsRow
            flush
            label="Issuer URL"
            helper="Use this as the provider issuer once the project is configured."
            control={
              project.issuer_url ? (
                <IdentifierPill value={project.issuer_url} label="Issuer URL" />
              ) : (
                <Tag variant="warn">Not configured</Tag>
              )
            }
          />
          <SettingsRow
            flush
            label="Client ID"
            helper="Public identifier for downstream apps."
            control={
              project.client_id ? (
                <IdentifierPill value={project.client_id} label="Client ID" />
              ) : (
                <Tag variant="warn">Not configured</Tag>
              )
            }
          />
          <SettingsRow
            flush
            label="Client secret"
            helper="Reveal before copying. It auto-hides after 30 seconds."
            control={
              clientSecret ? (
                <MaskedSecret name="client_secret" value={clientSecret} />
              ) : (
                <Tag variant="warn">Not available</Tag>
              )
            }
          />
          <SettingsRow
            flush
            label="Token endpoint auth"
            helper="Choose how confidential clients authenticate."
            control={
              <select
                aria-label="Token endpoint auth method"
                className="h-10 rounded-[var(--radius-md)] border border-border-default bg-bg-surface px-3 text-[14px]"
                value={authMethod}
                onChange={(event) => setAuthMethod(event.target.value as typeof authMethod)}
              >
                <option value="client_secret_basic">client_secret_basic</option>
                <option value="client_secret_post">client_secret_post</option>
                <option value="none">none</option>
              </select>
            }
          />
        </div>
        <div className="mt-4 grid gap-4 lg:grid-cols-2">
          <TextInput
            label="Redirect URIs"
            value={redirectUris}
            error={redirectError}
            onChange={(event) => setRedirectUris(event.target.value)}
          />
          <TextInput
            label="Allowed scopes"
            value={scopes}
            onChange={(event) => setScopes(event.target.value)}
          />
        </div>
        <div className="mt-4 flex flex-wrap gap-3">
          <Button
            variant="primary"
            loading={busy}
            disabled={Boolean(redirectError)}
            onClick={() => void saveProject()}
          >
            Save project
          </Button>
          <Button variant="secondary" onClick={() => setRotateOpen(true)}>
            Rotate client secret
          </Button>
          <Button variant="destructive" onClick={() => setDeleteOpen(true)}>
            Delete project
          </Button>
        </div>
      </Card>
      {issuerURL && clientID ? (
        <div className="grid gap-4 lg:grid-cols-2">
          <CodeBlock code={`AUTH_CYPRA_ISSUER=${issuerURL} AUTH_CYPRA_ID=${clientID}`} />
          <CodeBlock code={`go run ./cmd/app -issuer ${issuerURL}`} />
        </div>
      ) : (
        <InlineAlert
          variant="info"
          title="Configuration snippets unavailable"
          message="Issuer URL and client ID are not configured yet. Create or save the project before copying app setup commands."
        />
      )}
      <Modal
        title="Rotate client secret"
        open={rotateOpen}
        onClose={() => setRotateOpen(false)}
        footer={
          <Button variant="primary" loading={busy} onClick={() => void rotateSecret()}>
            Start rotation
          </Button>
        }
      >
        <p className="text-[14px] text-text-secondary">
          Downstream apps using the current secret will fail after the grace window. Coordinate the
          cutover with the app owner before rotating.
        </p>
      </Modal>
      <Modal
        title="Delete project"
        open={deleteOpen}
        onClose={() => setDeleteOpen(false)}
        footer={
          <Button
            variant="destructive"
            disabled={deleteValue !== project.slug}
            loading={busy}
            onClick={() => void confirmDelete()}
          >
            Delete project
          </Button>
        }
      >
        <TextInput
          label="Type project slug to confirm"
          value={deleteValue}
          onChange={(event) => setDeleteValue(event.target.value)}
        />
      </Modal>
    </div>
  );
}

function UserList({ tenant }: { tenant: TenantRecord }) {
  const usersQuery = useQuery({
    queryKey: ["users", tenant.slug],
    queryFn: listUsers,
    retry: false,
  });
  const [query, setQuery] = useState("");
  const [importOpen, setImportOpen] = useState(false);
  const [inviteOpen, setInviteOpen] = useState(false);
  const users = demoStateEnabled() ? demoUsers : (usersQuery.data ?? []);
  const visibleUsers = users.filter((user) =>
    query.trim() === "" ? true : user.email.toLowerCase().includes(query.trim().toLowerCase()),
  );
  const emailBlocked = tenant.email_provider_required ?? false;
  if (usersQuery.error?.message === "auth.forbidden") return <ErrorPage code="403" />;
  if (usersQuery.isLoading && !demoStateEnabled()) return <LoadingState />;
  if (usersQuery.isError && !demoStateEnabled()) {
    return (
      <ErrorState
        title="Couldn't load users."
        body="The user list request failed. Retry after checking tenant access."
        retry={() => void usersQuery.refetch()}
      />
    );
  }
  return (
    <Card
      title="Users"
      subtitle="Find users, invite new identities, or import them through the CLI."
      actions={
        <div className="flex gap-2">
          {emailBlocked ? (
            <Tooltip label="Configure an email provider before issuing invites.">
              <Button variant="primary" disabled>
                Invite user
              </Button>
            </Tooltip>
          ) : (
            <Button variant="primary" onClick={() => setInviteOpen(true)}>
              Invite user
            </Button>
          )}
          <Button variant="secondary" onClick={() => setImportOpen(true)}>
            Import via CLI
          </Button>
        </div>
      }
    >
      <TextInput
        label="Search users"
        type="search"
        placeholder="user@example.com"
        value={query}
        onChange={(event) => setQuery(event.target.value)}
      />
      <div className="mt-4">
        {users.length === 0 ? (
          <EmptyState
            title="No users yet."
            body="Invite a user or import identities with cypra import."
          />
        ) : visibleUsers.length === 0 ? (
          <EmptyState
            title={`No users match ${query}.`}
            body="Clear the search to see all users."
          />
        ) : (
          <div className="overflow-hidden rounded-[var(--radius-md)] border border-border-subtle">
            <table className="w-full text-left text-[13px]" data-responsive="stack">
              <thead className="bg-bg-code text-text-secondary">
                <tr>
                  <th className="px-4 py-3 font-medium">User</th>
                  <th className="px-4 py-3 font-medium">Subject</th>
                  <th className="px-4 py-3 font-medium">Methods</th>
                  <th className="px-4 py-3 font-medium">State</th>
                </tr>
              </thead>
              <tbody>
                {visibleUsers.map((user) => (
                  <tr key={user.id} className="border-t border-border-subtle">
                    <td className="px-4 py-3">
                      <a
                        className="font-medium hover:text-accent-primary"
                        href={`/dashboard/tenants/${tenant.slug}/users/${user.id}`}
                      >
                        {user.email}
                      </a>
                    </td>
                    <td className="px-4 py-3">
                      <IdentifierPill value={user.sub ?? user.id} label="Subject" />
                    </td>
                    <td className="px-4 py-3 text-text-secondary">
                      {(user.enrolled_methods ?? []).join(", ")}
                    </td>
                    <td className="px-4 py-3">
                      <Tag
                        variant={
                          user.state === "pending"
                            ? "pending"
                            : user.state === "deleting"
                              ? "warn"
                              : "success"
                        }
                      >
                        {user.state ?? "active"}
                      </Tag>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
      <InviteUserModal
        open={inviteOpen}
        onClose={() => setInviteOpen(false)}
        title="Invite user"
        roleLock="member"
      />
      <Modal title="Import users via CLI" open={importOpen} onClose={() => setImportOpen(false)}>
        <CodeBlock code={`cypra import --tenant ${tenant.slug} --file users.json`} />
        <div className="mt-4 rounded-[var(--radius-md)] border border-border-subtle bg-bg-code p-4 text-[13px] text-text-secondary">
          Use the CLI import path for bulk onboarding. The command validates users before writing
          tenant records.
        </div>
      </Modal>
    </Card>
  );
}

function UserDetail({ tenant, userId }: { tenant: TenantRecord; userId: string }) {
  const queryClient = useQueryClient();
  const usersQuery = useQuery({
    queryKey: ["users", tenant.slug],
    queryFn: listUsers,
    retry: false,
  });
  const detailQuery = useQuery({
    queryKey: ["user-detail", tenant.slug, userId],
    queryFn: () => getUserDetail(userId),
    retry: false,
  });
  const [activeTab, setActiveTab] = useState("Auth methods");
  const user =
    detailQuery.data?.user ??
    usersQuery.data?.find((item) => item.id === userId) ??
    (demoStateEnabled() ? demoUsers.find((item) => item.id === userId) : undefined);
  const [confirm, setConfirm] = useState<string | null>(null);
  const [pendingAction, setPendingAction] = useState<string | null>(null);
  const [actionMessage, setActionMessage] = useState<{
    variant: StatusVariant;
    message: string;
  } | null>(null);
  if (
    usersQuery.error?.message === "auth.forbidden" ||
    detailQuery.error?.message === "auth.forbidden"
  )
    return <ErrorPage code="403" />;
  if ((usersQuery.isLoading || detailQuery.isLoading) && !demoStateEnabled())
    return <LoadingState />;
  if ((usersQuery.isError || detailQuery.isError) && !demoStateEnabled()) {
    return (
      <ErrorState
        title="Couldn't load user."
        body="The user request failed. Retry after checking tenant access."
        retry={() => {
          void usersQuery.refetch();
          void detailQuery.refetch();
        }}
      />
    );
  }
  if (!user) {
    return (
      <ErrorState title="User not found." body="This user does not exist or is not visible." />
    );
  }
  const runAction = async () => {
    if (!confirm) return;
    setPendingAction(confirm);
    setActionMessage(null);
    try {
      if (confirm === "Reset password") await resetUserPassword(user.id);
      if (confirm === "Disable MFA") await disableUserMFA(user.id);
      if (confirm === "Enroll factor") await enrollUserFactor(user.id);
      if (confirm === "Delete user (DSR)") await deleteUserDSR(user.id);
      if (confirm === "Re-invite") await reinviteUser(user.id);
      if (confirm === "Export user data") {
        window.location.assign(userExportURL(user.id));
      } else {
        await queryClient.invalidateQueries({ queryKey: ["users", tenant.slug] });
        await queryClient.invalidateQueries({ queryKey: ["user-detail", tenant.slug, userId] });
      }
      setActionMessage({ variant: "success", message: `${confirm} completed for ${user.email}.` });
      setConfirm(null);
    } catch (error) {
      setActionMessage({
        variant: "error",
        message: error instanceof Error ? error.message : "user.action_failed",
      });
    } finally {
      setPendingAction(null);
    }
  };
  return (
    <div className="grid gap-4">
      {actionMessage ? (
        <Toast variant={actionMessage.variant} message={actionMessage.message} />
      ) : null}
      {user.state === "deleting" ? (
        <Toast
          variant="warn"
          message="gdpr.user_deletion_in_progress - this user's data export and deletion workflow is still running."
        />
      ) : null}
      <Card
        title={user.email}
        subtitle="Identity, enrolled methods, sessions, consents, audit, and metadata."
        actions={<IdentifierPill value={user.sub ?? user.id} label="Subject" />}
      >
        <div className="flex flex-wrap gap-2">
          {(detailQuery.data?.auth_methods ?? user.enrolled_methods ?? []).map((method) => (
            <Tag key={method} variant="info">
              {method}
            </Tag>
          ))}
        </div>
        <div className="mt-4 flex flex-wrap gap-2">
          {[
            "Reset password",
            "Disable MFA",
            "Enroll factor",
            "Delete user (DSR)",
            "Export user data",
            "Re-invite",
          ].map((action) => (
            <Button
              key={action}
              variant={action.includes("Delete") ? "destructive" : "secondary"}
              loading={pendingAction === action}
              onClick={() => setConfirm(action)}
            >
              {action}
            </Button>
          ))}
        </div>
      </Card>
      <nav
        className="flex gap-2 overflow-x-auto border-b border-border-subtle"
        aria-label="User tabs"
      >
        {["Auth methods", "Sessions", "Consents", "Audit", "Metadata"].map((tab) => (
          <button
            key={tab}
            className={`border-b-2 px-3 py-2 text-[13px] ${
              activeTab === tab
                ? "border-accent-primary text-text-primary"
                : "border-transparent text-text-secondary"
            }`}
            type="button"
            onClick={() => setActiveTab(tab)}
          >
            {tab}
          </button>
        ))}
      </nav>
      <UserDetailTab activeTab={activeTab} user={user} detail={detailQuery.data} />
      <Modal
        title="Confirm user action"
        open={confirm !== null}
        onClose={() => setConfirm(null)}
        footer={
          <Button
            variant={confirm?.includes("Delete") ? "destructive" : "primary"}
            loading={pendingAction !== null}
            onClick={() => void runAction()}
          >
            Confirm
          </Button>
        }
      >
        <p className="text-[14px] text-text-secondary">
          {confirm} for {user.email} writes an audit entry and applies tenant permissions.
        </p>
      </Modal>
    </div>
  );
}

function UserDetailTab({
  activeTab,
  user,
  detail,
}: {
  activeTab: string;
  user: UserRecord;
  detail?: UserDetailRecord;
}) {
  if (activeTab === "Auth methods") {
    const methods = detail?.auth_methods ?? user.enrolled_methods ?? [];
    if (methods.length === 0) {
      return (
        <EmptyState
          title="No auth methods enrolled."
          body="This user has not completed an auth method enrollment yet."
        />
      );
    }
    return (
      <Card title="Auth methods" subtitle="Tenant-scoped credential and factor enrollment state.">
        <div className="flex flex-wrap gap-2">
          {methods.map((method) => (
            <Tag
              key={method}
              variant={method.includes("2fa") || method === "totp" ? "warn" : "info"}
            >
              {method}
            </Tag>
          ))}
        </div>
      </Card>
    );
  }
  if (activeTab === "Sessions") {
    const sessions = detail?.sessions ?? [];
    if (sessions.length === 0)
      return (
        <EmptyState
          title="No sessions."
          body="No active or historical hosted sessions were found for this user."
        />
      );
    return (
      <Card title="Sessions" subtitle="Hosted dashboard/login sessions scoped to this tenant.">
        <div className="grid gap-2">
          {sessions.map((session) => (
            <ListRow
              key={session.id}
              title={session.revoked ? "Revoked session" : "Active session"}
              meta={`Last seen ${session.last_seen_at} · Expires ${session.expires_at}`}
              right={<IdentifierPill value={session.id} label="Session ID" />}
            />
          ))}
        </div>
      </Card>
    );
  }
  if (activeTab === "Consents") {
    const consents = detail?.consents ?? [];
    if (consents.length === 0)
      return (
        <EmptyState
          title="No OIDC consents."
          body="This user has not granted an OIDC client consent yet."
        />
      );
    return (
      <Card title="Consents" subtitle="OIDC client grants and scopes.">
        <div className="grid gap-2">
          {consents.map((consent) => (
            <ListRow
              key={consent.id}
              title={consent.client_id}
              meta={`Granted ${consent.granted_at}`}
              right={
                <span className="flex flex-wrap justify-end gap-1">
                  {consent.scopes.map((scope) => (
                    <Tag key={scope}>{scope}</Tag>
                  ))}
                </span>
              }
            />
          ))}
        </div>
      </Card>
    );
  }
  if (activeTab === "Audit") {
    const auditRows = detail?.audit ?? [];
    if (auditRows.length === 0)
      return (
        <EmptyState title="No audit entries." body="No user-scoped audit records were found." />
      );
    return (
      <Card title="Audit" subtitle="Recent audit rows for this user resource.">
        <div className="grid gap-2">
          {auditRows.map((entry) => (
            <AuditEntry key={entry.id} action={entry.action} resource={entry.resource_kind} />
          ))}
        </div>
      </Card>
    );
  }
  return (
    <Card title="Metadata" subtitle="Raw user metadata stored in the tenant database.">
      <CodeBlock code={JSON.stringify(detail?.metadata ?? user.metadata ?? {}, null, 2)} />
    </Card>
  );
}

const AUTH_PROVIDER_LABELS: Record<AuthProviderMethod, string> = {
  password: "Password",
  magic_link: "Magic link",
  passkey: "Passkey",
  totp: "TOTP",
  google: "Google",
  oidc_upstream: "OIDC upstream",
};

const AUTH_PROVIDER_DESCRIPTIONS: Record<AuthProviderMethod, string> = {
  password: "Email + password sign-in with reset flow.",
  magic_link: "Passwordless email sign-in via one-time link.",
  passkey: "WebAuthn passkeys scoped to this tenant's RP ID.",
  totp: "Authenticator app second factor.",
  google: "Google OAuth identity provider.",
  oidc_upstream: "Generic upstream OIDC providers.",
};

const TOP_AUTH_TABS = [
  { key: "overview", label: "Overview" },
  { key: "identifiers", label: "Identifiers" },
  { key: "mfa", label: "Multi-factor" },
  { key: "social", label: "Social SSO" },
  { key: "enterprise-sso", label: "Enterprise SSO" },
] as const;

const IDENTIFIER_TABS = [
  { key: "password", label: "Password" },
  { key: "magic-link", label: "Magic link" },
  { key: "passkey", label: "Passkey" },
] as const;

const MFA_TABS = [{ key: "totp", label: "TOTP" }] as const;

const SOCIAL_KINDS_ORDERED: SocialProviderKind[] = [
  "google",
  "microsoft",
  "apple",
  "github",
  "discord",
];

const SOCIAL_KIND_LABELS: Record<SocialProviderKind, string> = {
  google: "Google",
  microsoft: "Microsoft",
  apple: "Apple",
  github: "GitHub",
  discord: "Discord",
};

function AuthProvidersScreen({ tenant, active }: { tenant: TenantRecord; active: string }) {
  const slug = tenant.slug;
  const baseHref = `/dashboard/tenants/${slug}/auth-providers`;
  const segments = active.split("/").filter(Boolean);
  // Back-compat: legacy flat URLs (?/auth-providers/password etc.) still
  // resolve. Map them into the new top buckets for highlight purposes.
  const topKey = (() => {
    const head = segments[0] ?? "overview";
    if (head === "overview") return "overview";
    if (
      head === "identifiers" ||
      head === "password" ||
      head === "magic-link" ||
      head === "passkey"
    )
      return "identifiers";
    if (head === "mfa" || head === "totp") return "mfa";
    if (head === "social" || head === "google") return "social";
    if (head === "enterprise-sso" || head === "oidc-upstream") return "enterprise-sso";
    return "overview";
  })();
  const subKey = (() => {
    if (segments[0] === topKey) return segments[1];
    // Legacy flat key (e.g. ["password"]) reused as sub key.
    if (topKey === "identifiers" && segments[0] !== "identifiers") return segments[0];
    if (topKey === "mfa" && segments[0] !== "mfa") return segments[0];
    if (topKey === "social" && segments[0] === "google") return "google";
    if (topKey === "enterprise-sso" && segments[0] === "oidc-upstream") return "oidc-upstream";
    return undefined;
  })();
  return (
    <div className="grid gap-4">
      <Card
        title="Authentication providers"
        subtitle="Enable providers, tune their behavior, and pick which method leads on the hosted-login page."
      >
        <nav className="flex flex-wrap gap-2" aria-label="Auth provider sections">
          {TOP_AUTH_TABS.map((entry) => {
            const isActive = entry.key === topKey;
            const href = entry.key === "overview" ? baseHref : `${baseHref}/${entry.key}`;
            return (
              <a
                key={entry.key}
                href={href}
                className={cn(
                  "rounded-[var(--radius-md)] border px-3 py-1.5 text-[13px] font-medium",
                  isActive
                    ? "border-accent-primary bg-accent-primary-mu text-text-primary"
                    : "border-border-subtle text-text-secondary hover:border-border-emphasis",
                )}
              >
                {entry.label}
              </a>
            );
          })}
        </nav>
      </Card>
      {topKey === "overview" ? <AuthProvidersOverview tenant={tenant} /> : null}
      {topKey === "identifiers" ? (
        <IdentifiersTabSection tenant={tenant} active={subKey ?? "password"} />
      ) : null}
      {topKey === "mfa" ? <MFATabSection tenant={tenant} active={subKey ?? "totp"} /> : null}
      {topKey === "social" ? (
        <SocialSSOTabSection
          tenant={tenant}
          active={subKey ? (subKey as SocialProviderKind) : "google"}
        />
      ) : null}
      {topKey === "enterprise-sso" ? (
        <EnterpriseSSOTabSection tenant={tenant} active={subKey} />
      ) : null}
    </div>
  );
}

function NestedTabStrip({
  baseHref,
  tabs,
  active,
}: {
  baseHref: string;
  tabs: readonly { key: string; label: string }[];
  active: string;
}) {
  return (
    <nav className="flex flex-wrap gap-2" aria-label="Subsection">
      {tabs.map((tab) => {
        const isActive = tab.key === active;
        return (
          <a
            key={tab.key}
            href={`${baseHref}/${tab.key}`}
            className={cn(
              "rounded-[var(--radius-md)] border px-3 py-1 text-[12px] font-medium",
              isActive
                ? "border-accent-primary bg-accent-primary-mu text-text-primary"
                : "border-border-subtle text-text-secondary hover:border-border-emphasis",
            )}
          >
            {tab.label}
          </a>
        );
      })}
    </nav>
  );
}

function IdentifiersTabSection({ tenant, active }: { tenant: TenantRecord; active: string }) {
  const baseHref = `/dashboard/tenants/${tenant.slug}/auth-providers/identifiers`;
  return (
    <div className="grid gap-4">
      <Card>
        <NestedTabStrip baseHref={baseHref} tabs={IDENTIFIER_TABS} active={active} />
      </Card>
      {active === "password" ? <PasswordProviderScreen tenant={tenant} /> : null}
      {active === "magic-link" ? <MagicLinkProviderScreen tenant={tenant} /> : null}
      {active === "passkey" ? <PasskeyProviderScreen tenant={tenant} /> : null}
    </div>
  );
}

function MFATabSection({ tenant, active }: { tenant: TenantRecord; active: string }) {
  const baseHref = `/dashboard/tenants/${tenant.slug}/auth-providers/mfa`;
  return (
    <div className="grid gap-4">
      <Card>
        <NestedTabStrip baseHref={baseHref} tabs={MFA_TABS} active={active} />
      </Card>
      {active === "totp" ? <TOTPProviderScreen tenant={tenant} /> : null}
    </div>
  );
}

function SocialSSOTabSection({
  tenant,
  active,
}: {
  tenant: TenantRecord;
  active: SocialProviderKind;
}) {
  const baseHref = `/dashboard/tenants/${tenant.slug}/auth-providers/social`;
  const tabs = SOCIAL_KINDS_ORDERED.map((kind) => ({ key: kind, label: SOCIAL_KIND_LABELS[kind] }));
  return (
    <div className="grid gap-4">
      <Card>
        <NestedTabStrip baseHref={baseHref} tabs={tabs} active={active} />
      </Card>
      <SocialProviderPanel tenant={tenant} kind={active} />
    </div>
  );
}

function SocialProviderPanel({ tenant, kind }: { tenant: TenantRecord; kind: SocialProviderKind }) {
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: ["social-connections", tenant.slug],
    queryFn: listSocialConnections,
    retry: false,
  });
  const record = (query.data ?? []).find((row) => row.kind === kind);
  const [enabled, setEnabled] = useState(false);
  const [clientID, setClientID] = useState("");
  const [clientSecret, setClientSecret] = useState("");
  const [allowedDomains, setAllowedDomains] = useState("");
  const [allowSignup, setAllowSignup] = useState(true);
  const [error, setError] = useState("");
  const hydrated = useRef<string | null>(null);
  useEffect(() => {
    if (!query.data) return;
    if (hydrated.current === kind) return;
    hydrated.current = kind;
    setEnabled(record?.enabled ?? false);
    setClientID("");
    setClientSecret("");
    setAllowedDomains((record?.allowed_domains ?? []).join(", "));
    setAllowSignup(record?.allow_signup ?? true);
  }, [query.data, kind, record]);
  if (query.isLoading) return <LoadingState />;
  const reset = () => {
    hydrated.current = null;
    setEnabled(record?.enabled ?? false);
    setClientID("");
    setClientSecret("");
    setAllowedDomains((record?.allowed_domains ?? []).join(", "));
    setAllowSignup(record?.allow_signup ?? true);
  };
  const onSave = async () => {
    setError("");
    const trimmedID = clientID.trim();
    const trimmedSecret = clientSecret.trim();
    if (!record?.client_id_set && (!trimmedID || !trimmedSecret)) {
      setError("auth_providers.social_credentials_required");
      return;
    }
    const input: SocialConnectionInput = {
      enabled,
      client_id: trimmedID,
      client_secret: trimmedSecret,
      allowed_domains: allowedDomains
        .split(",")
        .map((d) => d.trim())
        .filter(Boolean),
      allow_signup: allowSignup,
    };
    try {
      await saveSocialConnection(kind, input);
      await queryClient.invalidateQueries({ queryKey: ["social-connections", tenant.slug] });
      hydrated.current = null;
      setClientID("");
      setClientSecret("");
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "auth_providers.social_save_failed");
    }
  };
  const onDelete = async () => {
    if (!record?.configured) return;
    setError("");
    try {
      await deleteSocialConnection(kind);
      await queryClient.invalidateQueries({ queryKey: ["social-connections", tenant.slug] });
      hydrated.current = null;
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "auth_providers.social_delete_failed");
    }
  };
  const dirty =
    enabled !== (record?.enabled ?? false) ||
    clientID.trim().length > 0 ||
    clientSecret.trim().length > 0 ||
    allowedDomains !== (record?.allowed_domains ?? []).join(", ") ||
    allowSignup !== (record?.allow_signup ?? true);
  return (
    <Card
      title={SOCIAL_KIND_LABELS[kind]}
      subtitle={`OAuth identity provider — ${SOCIAL_KIND_LABELS[kind]}`}
    >
      <SettingsRow
        flush
        label="Enabled"
        helper="Hide or show this provider on the hosted-login page."
        control={<Switch label="Enabled" checked={enabled} onChange={setEnabled} />}
      />
      <SettingsRow
        flush
        label="Client ID"
        helper={
          record?.client_id_set
            ? "Stored encrypted. Replace by typing a new value."
            : "OAuth client identifier from the provider console."
        }
        control={
          <TextInput
            label="Client ID"
            value={clientID}
            placeholder={record?.client_id_set ? "•••••••• (set)" : "client-id"}
            onChange={(event) => setClientID(event.target.value)}
          />
        }
      />
      <SettingsRow
        flush
        label="Client secret"
        helper={
          record?.client_id_set
            ? "Stored encrypted. Replace by typing a new value."
            : "OAuth client secret from the provider console."
        }
        control={
          <TextInput
            label="Client secret"
            type="password"
            value={clientSecret}
            placeholder={record?.client_id_set ? "•••••••• (set)" : "client-secret"}
            onChange={(event) => setClientSecret(event.target.value)}
          />
        }
      />
      <SettingsRow
        flush
        label="Allowed email domains"
        helper="Comma-separated list. Leave empty to accept any email."
        control={
          <TextInput
            label="Allowed domains"
            value={allowedDomains}
            placeholder="example.com, acme.org"
            onChange={(event) => setAllowedDomains(event.target.value)}
          />
        }
      />
      <SettingsRow
        flush
        label="Allow signup"
        helper="Permit new accounts via this provider."
        control={<Switch label="Allow signup" checked={allowSignup} onChange={setAllowSignup} />}
      />
      {error ? <InlineAlert variant="error" message={formatErrorCode(error)} /> : null}
      {dirty ? <SaveBar dirtyCount={1} onSave={() => void onSave()} onDiscard={reset} /> : null}
      {record?.configured ? (
        <button
          type="button"
          className="mt-3 text-[13px] font-medium text-status-error"
          onClick={() => void onDelete()}
        >
          Remove this connection
        </button>
      ) : null}
    </Card>
  );
}

function EnterpriseSSOTabSection({
  tenant,
  active,
}: {
  tenant: TenantRecord;
  active: string | undefined;
}) {
  const queryClient = useQueryClient();
  const baseHref = `/dashboard/tenants/${tenant.slug}/auth-providers/enterprise-sso`;
  const query = useQuery({
    queryKey: ["oidc-connections", tenant.slug],
    queryFn: listOIDCConnections,
    retry: false,
  });
  if (query.isLoading) return <LoadingState />;
  const connections = query.data ?? [];
  const tabs = [
    ...connections.map((c) => ({ key: c.slug, label: c.display_name })),
    { key: "__new", label: "+ Add connection" },
  ];
  const activeKey = active ?? (connections.length > 0 ? connections[0].slug : "__new");
  return (
    <div className="grid gap-4">
      <Card>
        <NestedTabStrip baseHref={baseHref} tabs={tabs} active={activeKey} />
      </Card>
      {activeKey === "__new" ? (
        <OIDCConnectionForm
          tenant={tenant}
          onSaved={async () => {
            await queryClient.invalidateQueries({ queryKey: ["oidc-connections", tenant.slug] });
          }}
        />
      ) : (
        <OIDCConnectionForm
          tenant={tenant}
          existing={connections.find((c) => c.slug === activeKey)}
          onSaved={async () => {
            await queryClient.invalidateQueries({ queryKey: ["oidc-connections", tenant.slug] });
          }}
        />
      )}
    </div>
  );
}

function OIDCConnectionForm({
  tenant,
  existing,
  onSaved,
}: {
  tenant: TenantRecord;
  existing?: OIDCConnectionRecord;
  onSaved: () => Promise<void>;
}) {
  const isNew = !existing;
  const [slug, setSlug] = useState(existing?.slug ?? "");
  const [displayName, setDisplayName] = useState(existing?.display_name ?? "");
  const [issuerURL, setIssuerURL] = useState(existing?.issuer_url ?? "");
  const [clientID, setClientID] = useState("");
  const [clientSecret, setClientSecret] = useState("");
  const [scopes, setScopes] = useState(
    (existing?.scopes ?? ["openid", "email", "profile"]).join(", "),
  );
  const [enabled, setEnabled] = useState(existing?.enabled ?? false);
  const [allowedDomains, setAllowedDomains] = useState(
    (existing?.allowed_domains ?? []).join(", "),
  );
  const [allowSignup, setAllowSignup] = useState(existing?.allow_signup ?? true);
  const [error, setError] = useState("");
  const onSave = async () => {
    setError("");
    const input: OIDCConnectionInput = {
      slug: isNew ? slug.trim() : undefined,
      display_name: displayName.trim(),
      issuer_url: issuerURL.trim(),
      client_id: clientID.trim() || undefined,
      client_secret: clientSecret.trim() || undefined,
      scopes: scopes
        .split(",")
        .map((s) => s.trim())
        .filter(Boolean),
      enabled,
      allowed_domains: allowedDomains
        .split(",")
        .map((d) => d.trim())
        .filter(Boolean),
      allow_signup: allowSignup,
    };
    try {
      if (existing) {
        await updateOIDCConnection(existing.slug, input);
      } else {
        await createOIDCConnection(input);
      }
      await onSaved();
      if (isNew) {
        window.location.href = `/dashboard/tenants/${tenant.slug}/auth-providers/enterprise-sso/${slug.trim()}`;
      }
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "auth_providers.oidc_save_failed");
    }
  };
  const onDelete = async () => {
    if (!existing) return;
    setError("");
    try {
      await deleteOIDCConnection(existing.slug);
      await onSaved();
      window.location.href = `/dashboard/tenants/${tenant.slug}/auth-providers/enterprise-sso`;
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "auth_providers.oidc_delete_failed");
    }
  };
  return (
    <Card
      title={existing ? existing.display_name : "Add OIDC connection"}
      subtitle={
        isNew
          ? "Configure a new enterprise OIDC identity provider."
          : "Edit this enterprise OIDC connection."
      }
    >
      {isNew ? (
        <SettingsRow
          flush
          label="Slug"
          helper="URL-safe identifier (lowercase letters, digits, hyphens)."
          control={
            <TextInput
              label="Slug"
              value={slug}
              placeholder="okta-prod"
              onChange={(event) => setSlug(event.target.value)}
            />
          }
        />
      ) : null}
      <SettingsRow
        flush
        label="Display name"
        helper="Shown on the hosted-login button."
        control={
          <TextInput
            label="Display name"
            value={displayName}
            onChange={(event) => setDisplayName(event.target.value)}
          />
        }
      />
      <SettingsRow
        flush
        label="Issuer URL"
        helper="Base URL where /.well-known/openid-configuration is hosted."
        control={
          <TextInput
            label="Issuer URL"
            value={issuerURL}
            placeholder="https://auth.example.com"
            onChange={(event) => setIssuerURL(event.target.value)}
          />
        }
      />
      <SettingsRow
        flush
        label="Client ID"
        helper={
          existing?.client_id_set
            ? "Stored encrypted. Replace by typing a new value."
            : "OAuth client identifier."
        }
        control={
          <TextInput
            label="Client ID"
            value={clientID}
            placeholder={existing?.client_id_set ? "•••••••• (set)" : "client-id"}
            onChange={(event) => setClientID(event.target.value)}
          />
        }
      />
      <SettingsRow
        flush
        label="Client secret"
        helper={
          existing?.client_id_set
            ? "Stored encrypted. Replace by typing a new value."
            : "OAuth client secret."
        }
        control={
          <TextInput
            label="Client secret"
            type="password"
            value={clientSecret}
            placeholder={existing?.client_id_set ? "•••••••• (set)" : "client-secret"}
            onChange={(event) => setClientSecret(event.target.value)}
          />
        }
      />
      <SettingsRow
        flush
        label="Scopes"
        helper="Comma-separated OIDC scopes."
        control={
          <TextInput
            label="Scopes"
            value={scopes}
            onChange={(event) => setScopes(event.target.value)}
          />
        }
      />
      <SettingsRow
        flush
        label="Allowed email domains"
        helper="Comma-separated list. Leave empty to accept any email."
        control={
          <TextInput
            label="Allowed domains"
            value={allowedDomains}
            onChange={(event) => setAllowedDomains(event.target.value)}
          />
        }
      />
      <SettingsRow
        flush
        label="Allow signup"
        helper="Permit new accounts via this connection."
        control={<Switch label="Allow signup" checked={allowSignup} onChange={setAllowSignup} />}
      />
      <SettingsRow
        flush
        label="Enabled"
        helper="Hide or show this provider on the hosted-login page."
        control={<Switch label="Enabled" checked={enabled} onChange={setEnabled} />}
      />
      {error ? <InlineAlert variant="error" message={formatErrorCode(error)} /> : null}
      <SaveBar
        dirtyCount={1}
        onSave={() => void onSave()}
        onDiscard={() => {
          if (isNew) {
            setSlug("");
            setDisplayName("");
            setIssuerURL("");
            setEnabled(false);
            setClientID("");
            setClientSecret("");
            setScopes("openid, email, profile");
            setAllowedDomains("");
            setAllowSignup(true);
          } else {
            setDisplayName(existing.display_name);
            setIssuerURL(existing.issuer_url);
            setEnabled(existing.enabled);
            setClientID("");
            setClientSecret("");
            setScopes(existing.scopes.join(", "));
            setAllowedDomains(existing.allowed_domains.join(", "));
            setAllowSignup(existing.allow_signup ?? true);
          }
        }}
      />
      {existing ? (
        <button
          type="button"
          className="mt-3 text-[13px] font-medium text-status-error"
          onClick={() => void onDelete()}
        >
          Remove this connection
        </button>
      ) : null}
    </Card>
  );
}

function AuthProvidersOverview({ tenant }: { tenant: TenantRecord }) {
  const queryClient = useQueryClient();
  const providersQuery = useQuery({
    queryKey: ["auth-providers", tenant.slug],
    queryFn: listAuthProviders,
    retry: false,
  });
  const defaultQuery = useQuery({
    queryKey: ["auth-providers-default", tenant.slug],
    queryFn: getDefaultAuthMethod,
    retry: false,
  });
  const [error, setError] = useState("");
  if (providersQuery.error?.message === "auth.forbidden") return <ErrorPage code="403" />;
  if (providersQuery.isLoading) return <LoadingState />;
  if (providersQuery.isError) {
    return (
      <ErrorState
        title="Couldn't load auth providers."
        body="The request failed. Retry after checking tenant permissions."
        retry={() => void providersQuery.refetch()}
      />
    );
  }
  const providers = providersQuery.data ?? [];
  const defaultMethod = defaultQuery.data?.method ?? "";

  const onChangeDefault = async (next: string) => {
    setError("");
    try {
      await setDefaultAuthMethod(next === "" ? null : (next as AuthProviderMethod));
      await queryClient.invalidateQueries({ queryKey: ["auth-providers-default", tenant.slug] });
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "auth_providers.default_save_failed");
    }
  };

  return (
    <div className="grid gap-4">
      <Card
        title="Overview"
        subtitle="Default method drives the primary block on hosted login. Per-method config lives in the tabs above."
      >
        <SettingsRow
          flush
          label="Default sign-in method"
          helper="Tenant-chosen primary on the hosted-login page. Falls back to the first enabled method when unset."
          control={
            <select
              aria-label="Default sign-in method"
              className="h-10 rounded-[var(--radius-md)] border border-border-default bg-bg-surface px-3 text-[14px]"
              value={defaultMethod}
              onChange={(event) => void onChangeDefault(event.target.value)}
            >
              <option value="">Auto (first enabled)</option>
              {providers
                .filter((provider) => provider.enabled)
                .map((provider) => (
                  <option key={provider.method} value={provider.method}>
                    {AUTH_PROVIDER_LABELS[provider.method]}
                  </option>
                ))}
            </select>
          }
        />
        {error ? <InlineAlert variant="error" message={formatErrorCode(error)} /> : null}
        <div className="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-3">
          {providers.map((provider) => (
            <Card key={provider.method}>
              <div className="mb-2 flex items-start justify-between gap-3">
                <div className="flex items-center gap-2">
                  <span
                    aria-hidden="true"
                    className={cn(
                      "inline-block h-2 w-2 flex-shrink-0 rounded-full",
                      provider.enabled ? "bg-status-success" : "bg-text-tertiary",
                    )}
                  />
                  <h2 className="text-[20px] font-semibold leading-7">
                    {AUTH_PROVIDER_LABELS[provider.method]}
                  </h2>
                  <span className="sr-only">{provider.enabled ? "Enabled" : "Disabled"}</span>
                </div>
                <Tag variant="info">{provider.enrolled_count} enrolled</Tag>
              </div>
              <p className="mb-3 text-[14px] leading-[1.571] text-text-secondary">
                {AUTH_PROVIDER_DESCRIPTIONS[provider.method]}
              </p>
              <a
                href={`/dashboard/tenants/${tenant.slug}/auth-providers/${methodSlug(provider.method)}`}
                className="text-[13px] font-medium text-accent-primary"
              >
                Configure →
              </a>
            </Card>
          ))}
        </div>
      </Card>
      <RegistrationSettingsCard tenant={tenant} />
    </div>
  );
}

const SIGNUP_MODE_HELP: Record<SignupMode, string> = {
  open: "Anyone with a valid email can register through any enabled method.",
  restricted:
    "Only addresses matching the allowlist can self-serve sign up. Invites bypass this list.",
  closed: "Self-serve registration is off entirely. Invites still work when enabled below.",
};

const REGISTRATION_DEFAULTS: RegistrationSettingsRecord = {
  mode: "open",
  allowlist: [],
  invites_enabled: true,
};

function RegistrationSettingsCard({ tenant }: { tenant: TenantRecord }) {
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: ["auth-providers-registration", tenant.slug],
    queryFn: getRegistrationSettings,
    retry: false,
  });
  const [mode, setMode] = useState<SignupMode>("open");
  const [allowlistDraft, setAllowlistDraft] = useState("");
  const [invitesEnabled, setInvitesEnabled] = useState(true);
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const hydrated = useRef(false);
  useEffect(() => {
    if (hydrated.current || !query.data) return;
    hydrated.current = true;
    setMode(query.data.mode);
    setAllowlistDraft(query.data.allowlist.join("\n"));
    setInvitesEnabled(query.data.invites_enabled);
  }, [query.data]);
  if (query.isLoading) return <LoadingState />;

  const onSave = async () => {
    setSaving(true);
    setError("");
    try {
      const allowlist = allowlistDraft
        .split(/\r?\n/)
        .map((line) => line.trim())
        .filter(Boolean);
      const next = await saveRegistrationSettings({
        mode,
        allowlist,
        invites_enabled: invitesEnabled,
      });
      setMode(next.mode);
      setAllowlistDraft(next.allowlist.join("\n"));
      setInvitesEnabled(next.invites_enabled);
      setDirty(false);
      await queryClient.invalidateQueries({
        queryKey: ["auth-providers-registration", tenant.slug],
      });
    } catch (caught) {
      setError(
        caught instanceof Error ? caught.message : "auth_providers.registration_save_failed",
      );
    } finally {
      setSaving(false);
    }
  };

  const onDiscard = () => {
    if (query.data) {
      setMode(query.data.mode);
      setAllowlistDraft(query.data.allowlist.join("\n"));
      setInvitesEnabled(query.data.invites_enabled);
    } else {
      setMode(REGISTRATION_DEFAULTS.mode);
      setAllowlistDraft("");
      setInvitesEnabled(REGISTRATION_DEFAULTS.invites_enabled);
    }
    setDirty(false);
    setError("");
  };

  return (
    <Card
      title="Registration"
      subtitle="Who can create accounts in this tenant. Per-method allow_signup flags layer on top."
    >
      <SettingsRow
        flush
        label="Sign-up mode"
        helper={SIGNUP_MODE_HELP[mode]}
        control={
          <select
            aria-label="Sign-up mode"
            className="h-10 rounded-[var(--radius-md)] border border-border-default bg-bg-surface px-3 text-[14px]"
            value={mode}
            onChange={(event) => {
              setMode(event.target.value as SignupMode);
              setDirty(true);
            }}
          >
            <option value="open">Open</option>
            <option value="restricted">Restricted</option>
            <option value="closed">Closed</option>
          </select>
        }
      />
      {mode === "restricted" ? (
        <SettingsRow
          flush
          label="Allowlist"
          helper="One pattern per line. Use *@example.com to allow a domain or alice@example.com for an exact match."
          control={
            <textarea
              aria-label="Sign-up allowlist"
              className="min-h-[120px] w-full rounded-[var(--radius-md)] border border-border-default bg-bg-surface p-3 text-[14px]"
              value={allowlistDraft}
              onChange={(event) => {
                setAllowlistDraft(event.target.value);
                setDirty(true);
              }}
            />
          }
        />
      ) : null}
      <SettingsRow
        flush
        label="Invite redemption"
        helper="Allow invited users to redeem their invite into a new account, even when self-serve sign-up is closed."
        control={
          <Switch
            label="Invites enabled"
            checked={invitesEnabled}
            onChange={(next) => {
              setInvitesEnabled(next);
              setDirty(true);
            }}
          />
        }
      />
      {error ? <InlineAlert variant="error" message={formatErrorCode(error)} /> : null}
      {dirty ? (
        <SaveBar
          dirtyCount={1}
          saving={saving}
          onSave={() => void onSave()}
          onDiscard={onDiscard}
        />
      ) : null}
    </Card>
  );
}

function methodSlug(method: AuthProviderMethod): string {
  return method.replace("_", "-");
}

function useProviderForm<T extends Record<string, unknown>>(
  tenant: TenantRecord,
  method: AuthProviderMethod,
  initial: T,
) {
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: ["auth-provider", tenant.slug, method],
    queryFn: () => getAuthProvider(method),
    retry: false,
  });
  const [enabled, setEnabled] = useState(false);
  const [config, setConfig] = useState<T>(initial);
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const hydrated = useRef(false);
  useEffect(() => {
    if (hydrated.current || !query.data) return;
    hydrated.current = true;
    setEnabled(query.data.enabled);
    setConfig({ ...initial, ...(query.data.config as T) });
  }, [query.data, initial]);

  const save = async () => {
    setSaving(true);
    setError("");
    try {
      await saveAuthProvider(method, { enabled, config: stripDefaults(config, initial) });
      await queryClient.invalidateQueries({ queryKey: ["auth-provider", tenant.slug, method] });
      await queryClient.invalidateQueries({ queryKey: ["auth-providers", tenant.slug] });
      setDirty(false);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "auth_providers.update_failed");
    } finally {
      setSaving(false);
    }
  };

  const discard = () => {
    if (query.data) {
      setEnabled(query.data.enabled);
      setConfig({ ...initial, ...(query.data.config as T) });
    } else {
      setEnabled(false);
      setConfig(initial);
    }
    setDirty(false);
    setError("");
  };

  return {
    query,
    enabled,
    setEnabled: (next: boolean) => {
      setEnabled(next);
      setDirty(true);
    },
    config,
    updateConfig: (patch: Partial<T>) => {
      setConfig((current) => ({ ...current, ...patch }));
      setDirty(true);
    },
    dirty,
    saving,
    error,
    setError,
    save,
    discard,
  };
}

// Strip keys whose value matches the typed default and whose JSON representation
// would otherwise clutter the saved blob. (Keeps the wire form compact.)
function stripDefaults<T extends Record<string, unknown>>(value: T, defaults: T): T {
  const out: Record<string, unknown> = {};
  for (const key of Object.keys(value)) {
    const v = value[key];
    if (v === undefined || v === "" || v === null) continue;
    if (Array.isArray(v) && v.length === 0) continue;
    if (defaults[key] === v) continue;
    out[key] = v;
  }
  return out as T;
}

interface PasswordPolicyForm extends Record<string, unknown> {
  min_length: number;
  max_length: number;
  require_upper: boolean;
  require_digit: boolean;
  require_special: boolean;
  allow_common_passwords: boolean;
  reset_enabled: boolean;
  allow_signup: boolean;
}

const PASSWORD_DEFAULTS: PasswordPolicyForm = {
  min_length: 12,
  max_length: 256,
  require_upper: false,
  require_digit: false,
  require_special: false,
  allow_common_passwords: false,
  reset_enabled: true,
  allow_signup: true,
};

function PasswordProviderScreen({ tenant }: { tenant: TenantRecord }) {
  const form = useProviderForm<PasswordPolicyForm>(tenant, "password", PASSWORD_DEFAULTS);
  if (form.query.isLoading) return <LoadingState />;
  return (
    <Card title="Password" subtitle={AUTH_PROVIDER_DESCRIPTIONS.password}>
      <ProviderEnabledRow form={form} />
      <SettingsRow
        flush
        label="Minimum length"
        helper="Reject passwords shorter than this."
        control={
          <TextInput
            label="Minimum length"
            type="number"
            min={1}
            value={String(form.config.min_length)}
            onChange={(event) =>
              form.updateConfig({ min_length: Number.parseInt(event.target.value, 10) || 0 })
            }
          />
        }
      />
      <SettingsRow
        flush
        label="Maximum length"
        helper="Cap the longest password we accept."
        control={
          <TextInput
            label="Maximum length"
            type="number"
            min={1}
            value={String(form.config.max_length)}
            onChange={(event) =>
              form.updateConfig({ max_length: Number.parseInt(event.target.value, 10) || 0 })
            }
          />
        }
      />
      <SettingsRow
        flush
        label="Require uppercase"
        helper="Password must contain at least one A–Z."
        control={
          <Switch
            label="Require uppercase"
            checked={form.config.require_upper}
            onChange={(next) => form.updateConfig({ require_upper: next })}
          />
        }
      />
      <SettingsRow
        flush
        label="Require digit"
        helper="Password must contain at least one 0–9."
        control={
          <Switch
            label="Require digit"
            checked={form.config.require_digit}
            onChange={(next) => form.updateConfig({ require_digit: next })}
          />
        }
      />
      <SettingsRow
        flush
        label="Require special character"
        helper="Password must contain at least one symbol (e.g. ! @ # $)."
        control={
          <Switch
            label="Require special"
            checked={form.config.require_special}
            onChange={(next) => form.updateConfig({ require_special: next })}
          />
        }
      />
      <SettingsRow
        flush
        label="Allow common passwords"
        helper="Disable the built-in common-password denylist (not recommended)."
        control={
          <Switch
            label="Allow common passwords"
            checked={form.config.allow_common_passwords}
            onChange={(next) => form.updateConfig({ allow_common_passwords: next })}
          />
        }
      />
      <SettingsRow
        flush
        label="Password reset enabled"
        helper="Whether users may request a reset email."
        control={
          <Switch
            label="Reset enabled"
            checked={form.config.reset_enabled}
            onChange={(next) => form.updateConfig({ reset_enabled: next })}
          />
        }
      />
      <SettingsRow
        flush
        label="Allow signup"
        helper="Permit new users to register with email + password."
        control={
          <Switch
            label="Allow signup"
            checked={form.config.allow_signup}
            onChange={(next) => form.updateConfig({ allow_signup: next })}
          />
        }
      />
      <ProviderSaveBar form={form} />
    </Card>
  );
}

interface MagicLinkPolicyForm extends Record<string, unknown> {
  ttl_minutes: number;
  max_active_per_user: number;
  allow_signup: boolean;
}

const MAGIC_LINK_DEFAULTS: MagicLinkPolicyForm = {
  ttl_minutes: 15,
  max_active_per_user: 3,
  allow_signup: true,
};

function MagicLinkProviderScreen({ tenant }: { tenant: TenantRecord }) {
  const form = useProviderForm<MagicLinkPolicyForm>(tenant, "magic_link", MAGIC_LINK_DEFAULTS);
  if (form.query.isLoading) return <LoadingState />;
  return (
    <Card title="Magic link" subtitle={AUTH_PROVIDER_DESCRIPTIONS.magic_link}>
      <ProviderEnabledRow form={form} />
      <SettingsRow
        flush
        label="Token TTL (minutes)"
        helper="How long an issued link stays valid."
        control={
          <TextInput
            label="TTL minutes"
            type="number"
            min={1}
            value={String(form.config.ttl_minutes)}
            onChange={(event) =>
              form.updateConfig({ ttl_minutes: Number.parseInt(event.target.value, 10) || 0 })
            }
          />
        }
      />
      <SettingsRow
        flush
        label="Max active links per user"
        helper="Cap how many unconsumed links a user may have at once."
        control={
          <TextInput
            label="Max active per user"
            type="number"
            min={1}
            value={String(form.config.max_active_per_user)}
            onChange={(event) =>
              form.updateConfig({
                max_active_per_user: Number.parseInt(event.target.value, 10) || 0,
              })
            }
          />
        }
      />
      <SettingsRow
        flush
        label="Allow signup"
        helper="Permit new users to register via a magic link."
        control={
          <Switch
            label="Allow signup"
            checked={form.config.allow_signup}
            onChange={(next) => form.updateConfig({ allow_signup: next })}
          />
        }
      />
      <ProviderSaveBar form={form} />
    </Card>
  );
}

interface PasskeyPolicyForm extends Record<string, unknown> {
  allow_signup: boolean;
  require_user_verification: boolean;
}

const PASSKEY_DEFAULTS: PasskeyPolicyForm = {
  allow_signup: true,
  require_user_verification: true,
};

function PasskeyProviderScreen({ tenant }: { tenant: TenantRecord }) {
  const form = useProviderForm<PasskeyPolicyForm>(tenant, "passkey", PASSKEY_DEFAULTS);
  if (form.query.isLoading) return <LoadingState />;
  return (
    <Card title="Passkey" subtitle={AUTH_PROVIDER_DESCRIPTIONS.passkey}>
      <ProviderEnabledRow form={form} />
      <SettingsRow
        flush
        label="Allow signup"
        helper="Permit new users to enroll a passkey from sign-up."
        control={
          <Switch
            label="Allow signup"
            checked={form.config.allow_signup}
            onChange={(next) => form.updateConfig({ allow_signup: next })}
          />
        }
      />
      <SettingsRow
        flush
        label="Require user verification"
        helper="Enforce the WebAuthn UV flag (PIN, biometric, etc.) on every assertion."
        control={
          <Switch
            label="Require UV"
            checked={form.config.require_user_verification}
            onChange={(next) => form.updateConfig({ require_user_verification: next })}
          />
        }
      />
      <ProviderSaveBar form={form} />
    </Card>
  );
}

interface TOTPPolicyForm extends Record<string, unknown> {
  issuer: string;
}

const TOTP_DEFAULTS: TOTPPolicyForm = { issuer: "" };

function TOTPProviderScreen({ tenant }: { tenant: TenantRecord }) {
  const form = useProviderForm<TOTPPolicyForm>(tenant, "totp", TOTP_DEFAULTS);
  if (form.query.isLoading) return <LoadingState />;
  return (
    <Card title="TOTP" subtitle={AUTH_PROVIDER_DESCRIPTIONS.totp}>
      <ProviderEnabledRow form={form} />
      <SettingsRow
        flush
        label="Issuer"
        helper="Override the issuer name shown in authenticator apps. Defaults to the tenant display name."
        control={
          <TextInput
            label="Issuer"
            placeholder="Acme Auth"
            value={form.config.issuer}
            onChange={(event) => form.updateConfig({ issuer: event.target.value })}
          />
        }
      />
      <ProviderSaveBar form={form} />
    </Card>
  );
}

function ProviderEnabledRow<T extends Record<string, unknown>>({
  form,
}: {
  form: ReturnType<typeof useProviderForm<T>>;
}) {
  return (
    <SettingsRow
      flush
      label="Enabled"
      helper="When disabled this provider is hidden from the hosted-login page."
      control={
        <Switch label="Enabled" checked={form.enabled} onChange={(next) => form.setEnabled(next)} />
      }
    />
  );
}

function ProviderSaveBar<T extends Record<string, unknown>>({
  form,
}: {
  form: ReturnType<typeof useProviderForm<T>>;
}) {
  return (
    <>
      {form.error ? <InlineAlert variant="error" message={formatErrorCode(form.error)} /> : null}
      {form.dirty ? (
        <SaveBar
          dirtyCount={1}
          saving={form.saving}
          onSave={() => void form.save()}
          onDiscard={form.discard}
        />
      ) : null}
    </>
  );
}

function ApiTokensTab() {
  const queryClient = useQueryClient();
  const tokensQuery = useQuery({
    queryKey: ["pats"],
    queryFn: listPersonalAccessTokens,
    retry: false,
  });
  const tokens = demoStateEnabled() ? demoTokens : (tokensQuery.data ?? []);
  const [createOpen, setCreateOpen] = useState(false);
  const [revokeToken, setRevokeToken] = useState<PersonalAccessTokenRecord | null>(null);
  const [revokeError, setRevokeError] = useState<string | undefined>();
  const [revokeBusy, setRevokeBusy] = useState(false);
  if (tokensQuery.error?.message === "auth.forbidden") return <ErrorPage code="403" />;
  if (tokensQuery.isLoading && !demoStateEnabled()) return <LoadingState />;
  if (tokensQuery.isError && !demoStateEnabled()) {
    return (
      <ErrorState
        title="Couldn't load API tokens."
        body="The token list request failed. Retry after checking permissions."
        retry={() => void tokensQuery.refetch()}
      />
    );
  }
  const onConfirmRevoke = async () => {
    if (!revokeToken) return;
    setRevokeBusy(true);
    setRevokeError(undefined);
    try {
      await revokePAT(revokeToken.id);
      await queryClient.invalidateQueries({ queryKey: ["pats"] });
      setRevokeToken(null);
    } catch (error) {
      setRevokeError(error instanceof Error ? error.message : "pat.revoke_failed");
    } finally {
      setRevokeBusy(false);
    }
  };
  return (
    <Card
      title="API tokens"
      subtitle="Personal access tokens are shown once at creation."
      actions={
        <Button variant="primary" data-primary-create="true" onClick={() => setCreateOpen(true)}>
          Create token
        </Button>
      }
    >
      <div className="grid gap-3">
        {tokens.length === 0 ? (
          <EmptyState
            title="No API tokens yet."
            body="Create a token to automate tenant API calls."
          />
        ) : null}
        {tokens.map((token) => (
          <div
            key={token.id}
            className="flex items-center gap-3 border-b border-border-subtle py-3"
          >
            <div className="flex-1">
              <span className="text-[14px]">{token.name}</span>
              <div className="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-[13px] text-text-secondary">
                <span>{token.scopes.join(", ")}</span>
                {token.last4 ? <span>ending {token.last4}</span> : null}
                <span>created {formatDate(token.created_at)}</span>
                <span>expires {token.expires_at ? formatDate(token.expires_at) : "never"}</span>
                <span>
                  last used {token.last_used_at ? formatDate(token.last_used_at) : "never"}
                </span>
                {token.revoked_at ? <span>revoked {formatDate(token.revoked_at)}</span> : null}
              </div>
            </div>
            <Button variant="destructive" size="sm" onClick={() => setRevokeToken(token)}>
              Revoke
            </Button>
          </div>
        ))}
      </div>
      <CreatePATModal open={createOpen} onClose={() => setCreateOpen(false)} />
      <ConfirmationDialog
        open={revokeToken !== null}
        variant="destructive"
        headline="Revoke API token?"
        body={
          revokeToken ? (
            <>
              <strong>{revokeToken.name}</strong> will stop authenticating API requests immediately.
              Existing in-flight requests with this token will fail at the next call.
            </>
          ) : null
        }
        confirmLabel="Revoke token"
        loading={revokeBusy}
        errorMessage={revokeError}
        onConfirm={onConfirmRevoke}
        onCancel={() => {
          if (!revokeBusy) {
            setRevokeToken(null);
            setRevokeError(undefined);
          }
        }}
      />
    </Card>
  );
}

function MembersTab() {
  const queryClient = useQueryClient();
  const membersQuery = useQuery({
    queryKey: ["tenant-members"],
    queryFn: listTenantMembers,
    retry: false,
  });
  const invitesQuery = useQuery({
    queryKey: ["tenant-invites"],
    queryFn: listTenantInvites,
    retry: false,
  });
  const [inviteOpen, setInviteOpen] = useState(false);
  const [roleDrafts, setRoleDrafts] = useState<Record<string, TenantMemberRecord["role"]>>({});
  const [saving, setSaving] = useState(false);
  const [revokeInvite, setRevokeInvite] = useState<string | null>(null);
  const [removeMember, setRemoveMember] = useState<TenantMemberRecord | null>(null);
  const [resendStatus, setResendStatus] = useState<{ email: string; ok: boolean } | null>(null);
  const [memberError, setMemberError] = useState("");
  const members = demoStateEnabled() ? demoTenantMembers : (membersQuery.data ?? []);
  const invites = demoStateEnabled() ? demoPendingInvites : (invitesQuery.data ?? []);
  const revokeInviteEmail =
    invites.find((invite) => invite.id === revokeInvite)?.email ?? "This invite";
  const editedRoles = Object.entries(roleDrafts).filter(([id, role]) => {
    const member = members.find((item) => item.id === id);
    return member && member.role !== role;
  });
  const onSaveRoles = async () => {
    setSaving(true);
    setMemberError("");
    try {
      await Promise.all(editedRoles.map(([id, role]) => updateTenantMemberRole(id, role)));
      setRoleDrafts({});
      await queryClient.invalidateQueries({ queryKey: ["tenant-members"] });
    } catch (error) {
      setMemberError(error instanceof Error ? error.message : "members.update_failed");
    } finally {
      setSaving(false);
    }
  };
  const handleRemoveMember = async () => {
    if (!removeMember) return;
    setMemberError("");
    try {
      await removeTenantMember(removeMember.id);
      setRemoveMember(null);
      await queryClient.invalidateQueries({ queryKey: ["tenant-members"] });
    } catch (error) {
      setMemberError(error instanceof Error ? error.message : "members.remove_failed");
    }
  };
  const handleResend = async (email: string) => {
    try {
      await resendInvite(email, "member");
      setResendStatus({ email, ok: true });
    } catch {
      setResendStatus({ email, ok: false });
    }
  };
  const handleRevokeInvite = async () => {
    if (!revokeInvite) return;
    try {
      await revokeTenantInvite(revokeInvite);
      await queryClient.invalidateQueries({ queryKey: ["tenant-invites"] });
      setRevokeInvite(null);
    } catch {
      setResendStatus({ email: "invite", ok: false });
    }
  };
  return (
    <div className="grid gap-4">
      <Card
        title="Members & roles"
        subtitle="Invite tenant operators and manage their dashboard roles."
        actions={
          <Button variant="primary" onClick={() => setInviteOpen(true)}>
            Invite member
          </Button>
        }
      >
        {membersQuery.isLoading && !demoStateEnabled() ? <LoadingState /> : null}
        {membersQuery.isError && !demoStateEnabled() ? (
          <ErrorState
            title="Couldn't load members."
            body="The members request failed. Retry after checking tenant permissions."
            retry={() => void membersQuery.refetch()}
          />
        ) : null}
        {!membersQuery.isLoading && !membersQuery.isError && members.length === 0 ? (
          <EmptyState title="No members." body="Redeemed invites will appear here." />
        ) : null}
        <div className="grid gap-3">
          {members.map((member) => {
            const currentRole = roleDrafts[member.id] ?? member.role;
            const ownerCount = members.filter(
              (item) => (roleDrafts[item.id] ?? item.role) === "owner",
            ).length;
            const lastOwner = member.role === "owner" && ownerCount <= 1;
            return (
              <div
                key={member.id}
                className="flex items-center gap-3 border-b border-border-subtle py-3"
              >
                <span className="flex-1 text-[14px]">{member.email}</span>
                <select
                  aria-label={`${member.email} role`}
                  className="h-9 rounded-[var(--radius-md)] border border-border-default bg-bg-surface px-3 text-[13px]"
                  value={currentRole}
                  onChange={(event) =>
                    setRoleDrafts((current) => ({
                      ...current,
                      [member.id]: event.target.value as TenantMemberRecord["role"],
                    }))
                  }
                >
                  <option value="owner">Owner</option>
                  <option value="admin">Admin</option>
                  <option value="member">Member</option>
                </select>
                <span className="text-[13px] text-text-secondary">
                  last seen {member.last_seen_at ? formatDate(member.last_seen_at) : "never"}
                </span>
                <Tooltip
                  label={
                    lastOwner
                      ? "You're the last owner. Promote someone else first."
                      : "Remove member"
                  }
                >
                  <Button
                    variant="destructive"
                    size="sm"
                    disabled={lastOwner}
                    onClick={() => setRemoveMember(member)}
                  >
                    Remove
                  </Button>
                </Tooltip>
              </div>
            );
          })}
        </div>
        {memberError ? <Toast variant="error" message={memberError} /> : null}
        {editedRoles.length > 0 ? (
          <SaveBar
            dirtyCount={editedRoles.length}
            onSave={onSaveRoles}
            onDiscard={() => {
              setRoleDrafts({});
              setMemberError("");
            }}
            saving={saving}
          />
        ) : null}
      </Card>
      <PermissionMatrix
        dirtyCount={editedRoles.length}
        onSave={onSaveRoles}
        onDiscard={() => setRoleDrafts({})}
        saving={saving}
      />
      <Card title="Pending invites" subtitle="Unredeemed pending_invitations rows.">
        {invitesQuery.isLoading && !demoStateEnabled() ? <LoadingState /> : null}
        {!invitesQuery.isLoading && invites.length === 0 ? (
          <EmptyState
            title="No pending invites."
            body="Invite a member to see pending invitations here."
          />
        ) : null}
        {invites.map((invite) => (
          <div
            key={invite.id}
            className="flex items-center gap-3 border-b border-border-subtle py-3 text-[13px]"
          >
            <span className="flex-1">{invite.email}</span>
            <Tag variant="pending">{invite.role}</Tag>
            <span className="text-text-secondary">expires {formatDate(invite.expires_at)}</span>
            <Button size="sm" onClick={() => void handleResend(invite.email)}>
              Resend
            </Button>
            <Button size="sm" variant="destructive" onClick={() => setRevokeInvite(invite.id)}>
              Revoke
            </Button>
          </div>
        ))}
        {resendStatus ? (
          <Toast
            variant={resendStatus.ok ? "info" : "error"}
            message={
              resendStatus.ok
                ? `Invite resent to ${resendStatus.email}.`
                : `Couldn't resend invite to ${resendStatus.email}.`
            }
          />
        ) : null}
      </Card>
      <InviteUserModal
        open={inviteOpen}
        onClose={() => setInviteOpen(false)}
        title="Invite member"
        roleLock="member"
      />
      <Modal
        title="Revoke pending invite"
        open={revokeInvite !== null}
        onClose={() => setRevokeInvite(null)}
        footer={
          <Button variant="destructive" onClick={() => void handleRevokeInvite()}>
            Revoke invite
          </Button>
        }
      >
        <p className="text-[14px] text-text-secondary">
          {revokeInviteEmail} will no longer be able to redeem this invite.
        </p>
      </Modal>
      <Modal
        title="Remove member"
        open={removeMember !== null}
        onClose={() => setRemoveMember(null)}
        footer={
          <Button variant="destructive" onClick={() => void handleRemoveMember()}>
            Remove member
          </Button>
        }
      >
        <p className="text-[14px] text-text-secondary">
          {removeMember?.email} will lose dashboard access for this tenant. Their tenant PATs are
          revoked immediately.
        </p>
      </Modal>
    </div>
  );
}

function SigningKeysScreen() {
  const queryClient = useQueryClient();
  const keysQuery = useQuery({
    queryKey: ["signing-keys"],
    queryFn: listSigningKeys,
    retry: false,
    refetchInterval: 30_000,
  });
  const [rotateOpen, setRotateOpen] = useState(false);
  const [kid, setKid] = useState("");
  const [rotateError, setRotateError] = useState<string | undefined>();
  const [rotateSuccess, setRotateSuccess] = useState(false);
  const [rotateBusy, setRotateBusy] = useState(false);
  const keys = demoStateEnabled() ? demoSigningKeys : (keysQuery.data ?? []);
  const activeKey = keys.find((key) => key.state === "active");
  const targetKid = activeKey?.kid ?? "";
  if (keysQuery.error?.message === "auth.forbidden") return <ErrorPage code="403" />;
  if (keysQuery.isLoading && !demoStateEnabled()) return <LoadingState />;
  if (keysQuery.isError && !demoStateEnabled()) {
    return (
      <ErrorState
        title="Couldn't load signing keys."
        body="The signing-key request failed. Retry after checking permissions."
        retry={() => void keysQuery.refetch()}
      />
    );
  }
  const rotate = async () => {
    if (!targetKid || kid !== targetKid) return;
    setRotateBusy(true);
    setRotateError(undefined);
    setRotateSuccess(false);
    try {
      await forceRotateSigningKey(targetKid);
      await queryClient.invalidateQueries({ queryKey: ["signing-keys"] });
      setKid("");
      setRotateOpen(false);
      setRotateSuccess(true);
    } catch (error) {
      setRotateError(error instanceof Error ? error.message : "signing_keys.rotate_failed");
    } finally {
      setRotateBusy(false);
    }
  };
  return (
    <div className="grid gap-4">
      {rotateSuccess ? <Toast variant="success" message="Signing key rotation started." /> : null}
      <Card
        title="Signing keys"
        subtitle="View key overlap, sunset, and rotation timing."
        actions={
          <Button variant="primary" onClick={() => setRotateOpen(true)} disabled={!activeKey}>
            Rotate now
          </Button>
        }
      >
        <KeyRotationTimeline keyID={targetKid || "no-active-key"} />
      </Card>
      <Card title="Key list">
        <div className="overflow-hidden rounded-[var(--radius-md)] border border-border-subtle">
          <table className="w-full text-left text-[13px]" data-responsive="stack">
            <thead className="bg-bg-code text-text-secondary">
              <tr>
                <th className="px-4 py-3 font-medium">KID</th>
                <th className="px-4 py-3 font-medium">State</th>
                <th className="px-4 py-3 font-medium">Activated</th>
                <th className="px-4 py-3 font-medium">Retires</th>
                <th className="px-4 py-3 font-medium">Sunset until</th>
              </tr>
            </thead>
            <tbody>
              {keys.map((key) => (
                <tr key={key.kid} className="border-t border-border-subtle">
                  <td className="px-4 py-3">
                    <IdentifierPill value={key.kid} label="Key ID" />
                  </td>
                  <td className="px-4 py-3">
                    <StatusPip variant={signingKeyStatus(key.state)} label={key.state} />
                  </td>
                  <td className="px-4 py-3 text-text-secondary">{formatDate(key.activated_at)}</td>
                  <td className="px-4 py-3 text-text-secondary">
                    {key.retires_at ? formatDate(key.retires_at) : "not set"}
                  </td>
                  <td className="px-4 py-3 text-text-secondary">
                    {key.sunset_until ? formatDate(key.sunset_until) : "not set"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {keys.length === 0 ? (
          <EmptyState
            title="No signing keys yet."
            body="Create a tenant to mint its first OIDC signing key."
          />
        ) : null}
      </Card>
      <Modal
        title="Rotate signing key"
        open={rotateOpen}
        onClose={() => {
          if (!rotateBusy) setRotateOpen(false);
        }}
        footer={
          <Button
            variant="primary"
            disabled={kid !== targetKid}
            loading={rotateBusy}
            onClick={() => void rotate()}
          >
            Rotate key
          </Button>
        }
      >
        <TextInput
          label="Type active kid to confirm"
          value={kid}
          onChange={(event) => setKid(event.target.value)}
        />
        <p className="mt-3 text-[13px] text-text-secondary">
          Active KID: <span className="font-mono text-text-primary">{targetKid || "none"}</span>
        </p>
        {rotateError ? <Toast variant="error" message={rotateError} /> : null}
      </Modal>
    </div>
  );
}

function signingKeyStatus(state: SigningKeyRecord["state"]): StatusVariant {
  if (state === "retired") return "revoked";
  return state;
}

export function AuditLogScreen({ scope }: { scope: "tenant" | "instance" }) {
  const initialFilters = initialAuditFilters();
  const [expanded, setExpanded] = useState<string | null>(null);
  const [range, setRange] = useState(initialFilters.range ?? "");
  const [action, setAction] = useState(initialFilters.action ?? "");
  const [actor, setActor] = useState(initialFilters.actor ?? "");
  const [resourceKind, setResourceKind] = useState(initialFilters.resourceKind ?? "");
  const filters = { range, action, actor, resourceKind };
  const entriesQuery = useQuery({
    queryKey: ["audit", scope, filters],
    queryFn: () => listAuditEntries(scope, filters),
    retry: false,
    refetchInterval: () => (document.visibilityState === "visible" ? 10_000 : false),
  });
  const entries = demoStateEnabled() ? demoAuditEntries(scope) : (entriesQuery.data ?? []);
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    setOrDelete(params, "range", range);
    setOrDelete(params, "action", action);
    setOrDelete(params, "actor", actor);
    setOrDelete(params, "resource_kind", resourceKind);
    const next = `${window.location.pathname}${params.toString() ? `?${params.toString()}` : ""}`;
    window.history.replaceState(window.history.state, "", next);
  }, [range, action, actor, resourceKind]);
  if (entriesQuery.error?.message === "auth.forbidden") return <ErrorPage code="403" />;
  const title = scope === "tenant" ? "Tenant audit" : "Instance audit";
  return (
    <div className="grid gap-6">
      <PageHeader title={title} subtitle="Filter, inspect, and export append-only audit entries." />
      <Card
        title="Audit entries"
        subtitle="Live filters and NDJSON export stay scoped to this view."
      >
        <div className="mb-4 grid gap-3 md:grid-cols-4">
          <TextInput
            label="Date range"
            placeholder="last 7 days"
            value={range}
            onChange={(event) => setRange(event.target.value)}
          />
          <TextInput
            label="Action"
            placeholder="tenant.update"
            value={action}
            onChange={(event) => setAction(event.target.value)}
          />
          <TextInput
            label="Actor"
            placeholder="ada@example.com"
            value={actor}
            onChange={(event) => setActor(event.target.value)}
          />
          <TextInput
            label="Resource kind"
            placeholder="user"
            value={resourceKind}
            onChange={(event) => setResourceKind(event.target.value)}
          />
        </div>
        <Toast
          variant={entriesQuery.isError && !demoStateEnabled() ? "error" : "info"}
          message={
            entriesQuery.isError && !demoStateEnabled()
              ? "Disconnected from audit stream. Showing the last successful result if one exists."
              : "Live updates active. Polling every 10 seconds while this page is visible."
          }
          action={
            <Button size="sm" onClick={() => void entriesQuery.refetch()}>
              Refresh
            </Button>
          }
        />
        <div className="mt-4">
          {entriesQuery.isLoading && !demoStateEnabled() ? <LoadingState /> : null}
          {!entriesQuery.isLoading && entries.length === 0 ? (
            <EmptyState
              title="No audit entries found."
              body="Adjust filters or wait for new audited activity."
            />
          ) : null}
          {entries.map((entry) => (
            <AuditEntryRow
              key={entry.id}
              entry={entry}
              expanded={expanded === entry.id}
              onToggle={() => setExpanded((current) => (current === entry.id ? null : entry.id))}
            />
          ))}
        </div>
        <div className="mt-4">
          <Button variant="primary" onClick={() => window.location.assign(auditExportURL(filters))}>
            Export NDJSON
          </Button>
        </div>
      </Card>
    </div>
  );
}

function AuditEntryRow({
  entry,
  expanded,
  onToggle,
}: {
  entry: AuditEntryRecord;
  expanded: boolean;
  onToggle: () => void;
}) {
  const resource = entry.resource_id ?? entry.resource_kind;
  return (
    <article className="border-b border-border-subtle py-3 text-[13px]">
      <div className="grid gap-2 sm:flex sm:items-center sm:gap-3">
        <span className="flex-1">
          <span className="font-medium">{entry.actor_kind}</span>{" "}
          <Tag variant="info">{entry.action}</Tag>
          {entry.redacted_at ? <Tag variant="warn">redacted</Tag> : null}
        </span>
        <IdentifierPill value={resource} label="Resource ID" />
        <time className="text-text-secondary">{formatDate(entry.occurred_at)}</time>
        <Button size="sm" onClick={onToggle}>
          {expanded ? "Collapse entry" : "Expand entry"}
        </Button>
      </div>
      {expanded ? (
        <pre className="mt-3 overflow-x-auto rounded-[var(--radius-md)] bg-bg-code p-3 font-mono text-[13px] text-text-identifier">
          {JSON.stringify(
            {
              state_before: entry.state_before,
              state_after: entry.state_after,
              metadata: entry.metadata,
            },
            null,
            2,
          )}
        </pre>
      ) : null}
    </article>
  );
}

function initialAuditFilters() {
  const params = new URLSearchParams(window.location.search);
  return {
    range: params.get("range") ?? undefined,
    action: params.get("action") ?? undefined,
    actor: params.get("actor") ?? undefined,
    resourceKind: params.get("resource_kind") ?? undefined,
  };
}

function setOrDelete(params: URLSearchParams, key: string, value: string) {
  if (value.trim() === "") {
    params.delete(key);
  } else {
    params.set(key, value.trim());
  }
}

function demoAuditEntries(scope: "tenant" | "instance"): AuditEntryRecord[] {
  return [
    {
      id: `demo-${scope}-audit`,
      occurred_at: "2026-05-06T00:00:00Z",
      actor_kind: scope === "tenant" ? "tenant_admin" : "instance_admin",
      action: "updated",
      resource_kind: scope === "tenant" ? "user" : "tenant",
      resource_id: scope === "tenant" ? "user_ada" : "tenant_acme",
      state_before: { email: "[redacted]" },
      state_after: { email: "[redacted]", role: "admin" },
      metadata: {},
    },
  ];
}

function PendingAdminInviteRow({ invite }: { invite: PendingInviteRecord }) {
  const queryClient = useQueryClient();
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [error, setError] = useState<string | undefined>();
  const [revoked, setRevoked] = useState(false);
  const onConfirm = async () => {
    try {
      await revokeInstanceInvite(invite.id);
      await queryClient.invalidateQueries({ queryKey: ["instance-invites"] });
      setRevoked(true);
      setConfirmOpen(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : "instance_admin.invite_revoke_failed");
    }
  };
  if (revoked) {
    return <p className="text-[13px] text-text-secondary">Invite to {invite.email} revoked.</p>;
  }
  return (
    <>
      <div className="flex items-center justify-between text-[13px]">
        <span>{invite.email}</span>
        <Tag variant="pending">{invite.role}</Tag>
        <span className="text-text-secondary">expires {invite.expires_at}</span>
        <Button
          size="sm"
          variant="destructive"
          onClick={() => {
            setError(undefined);
            setConfirmOpen(true);
          }}
        >
          Revoke
        </Button>
      </div>
      <ConfirmationDialog
        open={confirmOpen}
        variant="destructive"
        headline="Revoke pending admin invite?"
        body={
          <>
            {invite.email} will no longer be able to redeem this invite. You will need to re-issue
            an invite to onboard this admin.
          </>
        }
        resourceMatch={invite.email}
        confirmLabel="Revoke invite"
        errorMessage={error}
        onConfirm={onConfirm}
        onCancel={() => setConfirmOpen(false)}
      />
    </>
  );
}

export function InstanceAdminsScreen() {
  const queryClient = useQueryClient();
  const adminsQuery = useQuery({
    queryKey: ["instance-admins"],
    queryFn: listInstanceAdmins,
    retry: false,
  });
  const invitesQuery = useQuery({
    queryKey: ["instance-invites"],
    queryFn: listInstanceInvites,
    retry: false,
  });
  const admins = demoStateEnabled() ? demoInstanceAdmins : (adminsQuery.data ?? []);
  const invites = demoStateEnabled()
    ? [
        {
          id: "demo-invite",
          email: "ops@example.com",
          role: "instance_admin",
          expires_at: "2026-05-12T00:00:00Z",
          created_at: "2026-05-06T00:00:00Z",
        },
      ]
    : (invitesQuery.data ?? []);
  const [inviteOpen, setInviteOpen] = useState(false);
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteError, setInviteError] = useState("");
  const [inviteBusy, setInviteBusy] = useState(false);
  const [demote, setDemote] = useState<InstanceAdminRecord | null>(null);
  const [demoteError, setDemoteError] = useState("");
  if (adminsQuery.error?.message === "auth.forbidden") return <ErrorPage code="403" />;
  if (adminsQuery.isLoading && !demoStateEnabled()) return <LoadingState />;
  if ((adminsQuery.isError || invitesQuery.isError) && !demoStateEnabled()) {
    return (
      <ErrorState
        title="Couldn't load instance admins."
        body="The instance-admin request failed. Retry after checking install-level access."
        retry={() => void adminsQuery.refetch()}
      />
    );
  }
  const submitInvite = async () => {
    setInviteBusy(true);
    setInviteError("");
    try {
      await inviteInstanceAdmin(inviteEmail);
      await queryClient.invalidateQueries({ queryKey: ["instance-invites"] });
      setInviteEmail("");
      setInviteOpen(false);
    } catch (error) {
      setInviteError(error instanceof Error ? error.message : "instance.invite_failed");
    } finally {
      setInviteBusy(false);
    }
  };
  return (
    <div>
      <PageHeader
        title="Instance admins"
        subtitle="Install-level operators outside the tenant model."
      />
      <Card
        title="Admin access"
        subtitle="Invite admins, review last-seen timestamps, and protect the last-admin recovery path."
        actions={
          <Button variant="primary" onClick={() => setInviteOpen(true)}>
            Invite admin
          </Button>
        }
      >
        <div className="overflow-hidden rounded-[var(--radius-md)] border border-border-subtle">
          {admins.length === 0 ? (
            <EmptyState
              title="No instance admins visible."
              body="Invite an admin to recover access."
            />
          ) : null}
          <table className="w-full text-left text-[13px]" data-responsive="stack">
            <thead className="bg-bg-code text-text-secondary">
              <tr>
                <th className="px-4 py-3 font-medium">Email</th>
                <th className="px-4 py-3 font-medium">Role</th>
                <th className="px-4 py-3 font-medium">Last seen</th>
                <th className="px-4 py-3 font-medium">Created</th>
                <th className="px-4 py-3 font-medium">Action</th>
              </tr>
            </thead>
            <tbody>
              {admins.map((admin) => {
                const lastAdmin = admins.length === 1;
                return (
                  <tr key={admin.id} className="border-t border-border-subtle">
                    <td className="px-4 py-3">{admin.email}</td>
                    <td className="px-4 py-3">
                      <Tag variant="context-instance">{admin.role}</Tag>
                    </td>
                    <td className="px-4 py-3 text-text-secondary">
                      {admin.last_seen_at ?? "never"}
                    </td>
                    <td className="px-4 py-3 text-text-secondary">{admin.created_at}</td>
                    <td className="px-4 py-3">
                      <Tooltip
                        label={
                          lastAdmin
                            ? "This is the last instance admin. Invite another admin before demoting."
                            : "Demote instance admin"
                        }
                      >
                        <Button
                          size="sm"
                          variant="destructive"
                          disabled={lastAdmin}
                          onClick={() => setDemote(admin)}
                        >
                          Demote
                        </Button>
                      </Tooltip>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
        <Card
          className="mt-4"
          title="Pending admin invites"
          subtitle="Instance-admin pending invitations use the install-branded admin-invite template."
        >
          {invites.length === 0 ? (
            <EmptyState
              title="No pending admin invites."
              body="Invite an admin to share recovery access."
            />
          ) : (
            <div className="grid gap-3">
              {invites.map((invite) => (
                <PendingAdminInviteRow key={invite.id} invite={invite} />
              ))}
            </div>
          )}
        </Card>
      </Card>
      <Modal
        title="Invite instance admin"
        open={inviteOpen}
        onClose={() => setInviteOpen(false)}
        footer={
          <Button variant="primary" loading={inviteBusy} onClick={() => void submitInvite()}>
            {inviteBusy ? "Sending..." : "Send invite"}
          </Button>
        }
      >
        <TextInput
          label="Email"
          type="email"
          placeholder="admin@example.com"
          value={inviteEmail}
          onChange={(event) => setInviteEmail(event.target.value)}
        />
        {inviteError ? <Toast variant="error" message={inviteError} /> : null}
      </Modal>
      <Modal
        title="Demote instance admin"
        open={demote !== null}
        onClose={() => setDemote(null)}
        footer={
          <Button
            variant="destructive"
            onClick={() =>
              void (async () => {
                if (!demote) return;
                try {
                  await demoteInstanceAdmin(demote.id);
                  setDemote(null);
                } catch (error) {
                  setDemoteError(
                    error instanceof Error ? error.message : "instance_admin.demote_failed",
                  );
                }
              })()
            }
          >
            Demote admin
          </Button>
        }
      >
        <p className="text-[14px] text-text-secondary">
          {demote?.email} will lose instance-admin access.
        </p>
        {demoteError ? <Toast variant="error" message={demoteError} /> : null}
      </Modal>
    </div>
  );
}

export function InstanceDiagnosticsScreen() {
  const diagnosticsQuery = useQuery({
    queryKey: ["instance-diagnostics"],
    queryFn: getInstanceDiagnostics,
    refetchInterval: 5_000,
    retry: false,
  });
  const diagnostics = demoStateEnabled() ? demoDiagnostics : diagnosticsQuery.data;
  if (diagnosticsQuery.error?.message === "auth.forbidden") return <ErrorPage code="403" />;
  if (diagnosticsQuery.isLoading) return <LoadingState />;
  if (diagnosticsQuery.isError && !demoStateEnabled()) {
    return (
      <ErrorState
        title="Couldn't load diagnostics."
        body="The diagnostics request failed. Retry after checking install-level access."
        retry={() => void diagnosticsQuery.refetch()}
      />
    );
  }
  if (!diagnostics) return <LoadingState />;
  return (
    <div>
      <PageHeader
        title="Diagnostics"
        subtitle="Read-only instance health, version, migration, master-key, and storage status."
      />
      <div className="grid gap-4 lg:grid-cols-2">
        <Card title="Health" subtitle="Polled every 5 seconds.">
          {Object.entries(diagnostics.health).map(([name, ok]) => (
            <StatusPip
              key={name}
              variant={ok ? "success" : "error"}
              label={`${name}: ${ok ? "reachable" : "unhealthy"}`}
            />
          ))}
        </Card>
        <Card title="Version">
          <IdentifierPill
            value={`${diagnostics.version.version}-${diagnostics.version.commit}`}
            label="Version"
          />
          <p className="mt-2 text-[13px] text-text-secondary">
            Build date: {diagnostics.version.build_date ?? "unknown"}
          </p>
        </Card>
        <Card title="Migration state">
          <StatusPip
            variant={diagnostics.migrations.pending.length === 0 ? "success" : "warn"}
            label={`${String(diagnostics.migrations.pending.length)} pending`}
          />
          <p className="mt-2 text-[13px] text-text-secondary">
            Current schema version: {diagnostics.migrations.current}
          </p>
        </Card>
        <Card title="Master-key rotation">
          <StatusPip
            variant={diagnostics.master_key_rotation.phase === "done" ? "success" : "pending"}
            label={diagnostics.master_key_rotation.phase}
          />
          <p className="mt-2 text-[13px] text-text-secondary">
            {diagnostics.master_key_rotation.rows_done} /{" "}
            {diagnostics.master_key_rotation.rows_total} rows rewrapped
          </p>
        </Card>
        <Card
          title="Storage backend"
          subtitle="Bootstrap-only environment values. Cypra never writes storage config to the DB."
        >
          <div className="grid gap-2 text-[13px]">
            <span>Kind: {diagnostics.storage.kind}</span>
            <span>Bucket: {diagnostics.storage.bucket ?? "not applicable"}</span>
            <span>Endpoint: {diagnostics.storage.endpoint ?? "local"}</span>
            <span>Region: {diagnostics.storage.region ?? "not applicable"}</span>
            <StatusPip
              variant={diagnostics.storage.credentials_present ? "success" : "error"}
              label="credentials present"
            />
          </div>
        </Card>
      </div>
    </div>
  );
}

function TenantSettings({ tenant, active }: { tenant: TenantRecord; active: string }) {
  const tabs = ["branding", "email", "members", "api-tokens", "danger"];
  return (
    <div className="grid gap-5 lg:grid-cols-[220px_1fr]">
      <nav
        className="grid content-start gap-1 rounded-[var(--radius-md)] border border-border-subtle bg-bg-surface p-2"
        aria-label="Settings tabs"
      >
        {tabs.map((tab) => (
          <a
            key={tab}
            href={`/dashboard/tenants/${tenant.slug}/settings/${tab}`}
            aria-current={active === tab ? "page" : undefined}
            className={`rounded-[var(--radius-md)] px-3 py-2 text-[13px] capitalize ${active === tab ? "bg-accent-primary-mu text-text-primary" : "text-text-secondary hover:bg-bg-code hover:text-text-primary"}`}
          >
            {tab.replace("-", " ")}
          </a>
        ))}
      </nav>
      {active === "branding" ? <BrandingTab tenant={tenant} /> : null}
      {active === "email" ? <EmailProviderScreen /> : null}
      {active === "api-tokens" ? <ApiTokensTab /> : null}
      {active === "members" ? <MembersTab /> : null}
      {active === "danger" ? <TenantDangerScreen tenant={tenant} /> : null}
      {active !== "branding" &&
      active !== "email" &&
      active !== "api-tokens" &&
      active !== "members" &&
      active !== "danger" ? (
        <PlaceholderPage title={`${active.replace("-", " ")} settings`} />
      ) : null}
    </div>
  );
}

function BrandingTab({ tenant }: { tenant: TenantRecord }) {
  const queryClient = useQueryClient();
  const [displayName, setDisplayName] = useState(tenant.branding?.display_name ?? tenant.name);
  const [accent, setAccent] = useState(tenant.branding?.accent ?? defaultAccent);
  const [poweredBy, setPoweredBy] = useState(tenant.branding?.powered_by ?? true);
  const [logoError, setLogoError] = useState("");
  const [logoFile, setLogoFile] = useState<File | null>(null);
  const [logoCleared, setLogoCleared] = useState(false);
  const [logoPreviewURL, setLogoPreviewURL] = useState(tenant.branding?.logo_url ?? "");
  const [accentError, setAccentError] = useState("");
  const [previewOpen, setPreviewOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState("");
  const dirty =
    displayName !== (tenant.branding?.display_name ?? tenant.name) ||
    accent !== (tenant.branding?.accent ?? defaultAccent) ||
    poweredBy !== (tenant.branding?.powered_by ?? true) ||
    logoFile !== null ||
    logoCleared;

  const validateLogo = (file: File | null) => {
    if (!file) {
      setLogoFile(null);
      setLogoError("");
      if (logoPreviewURL) {
        setLogoCleared(true);
        setLogoPreviewURL("");
      }
      return;
    }
    setLogoCleared(false);
    setLogoFile(file);
    if (file.type !== "image/png" && file.type !== "image/jpeg" && file.type !== "image/webp") {
      setLogoError("Use a PNG, JPG, or WEBP image.");
      return;
    }
    const image = new Image();
    image.onload = () => {
      const ratio = image.width / image.height;
      const accepted =
        image.width >= 256 && [1, 16 / 9, 21 / 9].some((target) => Math.abs(ratio - target) < 0.08);
      setLogoError(accepted ? "" : "Image must be at least 256px wide and square, 16:9, or 21:9.");
    };
    image.src = URL.createObjectURL(file);
  };

  const save = async () => {
    if (!accentLooksValid(accent)) {
      setAccentError("Accent must be readable on white and support readable text on accent.");
      return;
    }
    if (logoFile && logoError) {
      return;
    }
    setSaving(true);
    setSaveError("");
    try {
      if (logoFile) {
        const result = await uploadTenantLogo(tenant.id, logoFile);
        setLogoFile(null);
        setLogoCleared(false);
        setLogoPreviewURL(`${result.logo_url}?t=${String(Date.now())}`);
      } else if (logoCleared) {
        await deleteTenantLogo(tenant.id);
        setLogoCleared(false);
        setLogoPreviewURL("");
      }
      await saveTenantBranding(tenant.id, {
        display_name: displayName,
        accent,
        powered_by: poweredBy,
      });
      await queryClient.invalidateQueries({ queryKey: ["tenants"] });
    } catch (error) {
      setSaveError(error instanceof Error ? error.message : "tenant.branding_failed");
    } finally {
      setSaving(false);
    }
  };
  const discard = () => {
    setDisplayName(tenant.branding?.display_name ?? tenant.name);
    setAccent(tenant.branding?.accent ?? defaultAccent);
    setPoweredBy(tenant.branding?.powered_by ?? true);
    setLogoFile(null);
    setLogoCleared(false);
    setLogoPreviewURL(tenant.branding?.logo_url ?? "");
    setLogoError("");
    setAccentError("");
    setSaveError("");
  };

  return (
    <Card title="Branding" subtitle="Tenant overrides affect hosted-login surfaces only.">
      <SettingsRow
        label="Show 'Powered by Cypra' on hosted login"
        helper="Open a hosted-login preview before saving changes."
        control={
          <div className="flex items-center gap-3">
            <Switch label="Powered by Cypra" checked={poweredBy} onChange={setPoweredBy} />
            <Button size="sm" onClick={() => setPreviewOpen(true)}>
              Preview
            </Button>
          </div>
        }
      />
      <SettingsRow
        label="Display name"
        helper="Shown in hosted-login page headings and email templates."
        control={
          <TextInput
            label="Display name"
            value={displayName}
            onChange={(event) => setDisplayName(event.target.value)}
          />
        }
      />
      <SettingsRow
        label="Logo"
        helper="PNG, JPG, or WEBP. At least 256px wide and square, 16:9, or 21:9."
        control={
          <div className="w-[360px]">
            <ImageUpload
              accept="image/png,image/jpeg,image/webp"
              maxSizeBytes={5 * 1024 * 1024}
              value={logoFile}
              initialPreviewUrl={logoPreviewURL || undefined}
              error={logoError || undefined}
              onChange={validateLogo}
            />
          </div>
        }
      />
      <SettingsRow
        label="Accent"
        helper="Accent must be readable on both white and your accent's text color."
        control={
          <TextInput
            label="Accent color"
            type="color"
            value={accent}
            error={accentError || undefined}
            onChange={(event) => {
              setAccent(event.target.value);
              setAccentError("");
            }}
          />
        }
      />
      {saveError ? <Toast variant="error" message={saveError} /> : null}
      {dirty ? <SaveBar dirtyCount={1} saving={saving} onSave={save} onDiscard={discard} /> : null}
      <Modal title="Hosted-login preview" open={previewOpen} onClose={() => setPreviewOpen(false)}>
        <div className="mx-auto max-w-sm rounded-[var(--radius-lg)] border border-border-subtle bg-bg-canvas p-5 text-center">
          <div className="mx-auto mb-4 grid h-12 w-12 place-items-center rounded-[var(--radius-pill)] bg-accent-primary-mu text-accent-primary">
            <PasskeyGlyph className="h-7 w-7" />
          </div>
          <h2 className="text-[22px] font-semibold">Sign in to {displayName}</h2>
          <p className="mt-2 text-[13px] text-text-secondary">
            Passkey, password, magic link, and second-factor flows inherit this brand.
          </p>
          {poweredBy ? (
            <p className="mt-5 text-[12px] text-text-tertiary">Powered by Cypra</p>
          ) : null}
        </div>
      </Modal>
    </Card>
  );
}

function EmailProviderScreen() {
  const providerQuery = useQuery({
    queryKey: ["provider-config", "email"],
    queryFn: getEmailProviderConfig,
    retry: false,
  });
  if (providerQuery.error?.message === "auth.forbidden") return <ErrorPage code="403" />;
  if (providerQuery.isLoading && !demoStateEnabled()) return <LoadingState />;
  if (providerQuery.isError && !demoStateEnabled()) {
    return (
      <ErrorState
        title="Couldn't load email provider."
        body="The provider request failed. Retry after checking tenant permissions."
        retry={() => void providerQuery.refetch()}
      />
    );
  }
  const provider = (demoStateEnabled() ? demoEmailProvider : providerQuery.data) ?? {
    kind: "",
    configured: false,
    healthy: false,
  };
  return <EmailProviderForm provider={provider} />;
}

function EmailProviderForm({ provider }: { provider: ProviderConfigRecord }) {
  const queryClient = useQueryClient();
  const [kind, setKind] = useState(provider.kind || "");
  const [fromAddress, setFromAddress] = useState("");
  const [fromName, setFromName] = useState("");
  const [smtpHost, setSmtpHost] = useState("");
  const [smtpPort, setSmtpPort] = useState("");
  const [smtpUsername, setSmtpUsername] = useState("");
  const [smtpPassword, setSmtpPassword] = useState("");
  const [resendKey, setResendKey] = useState("");
  const [testing, setTesting] = useState(false);
  const [dirty, setDirty] = useState(false);
  const [saveError, setSaveError] = useState("");
  const [saving, setSaving] = useState(false);
  const showSaveBar = dirty || !provider.configured;
  return (
    <ProviderSettingsShell
      title="Email provider"
      provider={provider}
      diagnostic="Send test email"
      testing={testing}
      onDiagnostic={async () => {
        setTesting(true);
        setSaveError("");
        try {
          await testEmailProviderConfig();
          await queryClient.invalidateQueries({ queryKey: ["provider-config", "email"] });
        } catch (error) {
          setSaveError(error instanceof Error ? error.message : "provider.email_test_failed");
        } finally {
          setTesting(false);
        }
      }}
    >
      {!provider.configured ? (
        <EmptyState
          icon={<Icons.Settings className="h-6 w-6" />}
          title="No email provider configured yet"
          body="Pick a provider below to start sending magic links, password resets, verification, and invite emails. Resend is the fastest production path; terminal is fine for local development."
        />
      ) : null}
      <SettingsRow
        label="Provider kind"
        helper="Terminal is local-only. SMTP and Resend are production-capable."
        control={
          <select
            aria-label="Email provider kind"
            className="h-10 rounded-[var(--radius-md)] border border-border-default bg-bg-surface px-3 text-[14px]"
            value={kind}
            onChange={(event) => {
              setKind(event.target.value);
              setDirty(true);
            }}
          >
            <option value="" disabled>
              Choose a provider…
            </option>
            <option value="terminal">terminal</option>
            <option value="smtp">smtp</option>
            <option value="resend">resend</option>
          </select>
        }
      />
      <SettingsRow
        label="From address"
        helper="Used by magic-link, password-reset, verification, and invite templates."
        control={
          <TextInput
            label="From address"
            placeholder="auth@example.com"
            value={fromAddress}
            onChange={(event) => {
              setFromAddress(event.target.value);
              setDirty(true);
            }}
          />
        }
      />
      <SettingsRow
        label="From name"
        helper="Tenant-branded sender name."
        control={
          <TextInput
            label="From name"
            placeholder="Cypra Auth"
            value={fromName}
            onChange={(event) => {
              setFromName(event.target.value);
              setDirty(true);
            }}
          />
        }
      />
      {kind === "smtp" ? (
        <>
          <SettingsRow
            label="SMTP host"
            helper="Hostname of the SMTP relay."
            control={
              <TextInput
                label="SMTP host"
                placeholder="smtp.example.com"
                value={smtpHost}
                onChange={(event) => {
                  setSmtpHost(event.target.value);
                  setDirty(true);
                }}
              />
            }
          />
          <SettingsRow
            label="SMTP port"
            helper="Typically 587 for STARTTLS or 465 for implicit TLS."
            control={
              <TextInput
                label="SMTP port"
                inputMode="numeric"
                placeholder="587"
                value={smtpPort}
                onChange={(event) => {
                  setSmtpPort(event.target.value);
                  setDirty(true);
                }}
              />
            }
          />
          <SettingsRow
            label="SMTP username"
            helper="Account used to authenticate to the relay."
            control={
              <TextInput
                label="SMTP username"
                placeholder="apikey"
                value={smtpUsername}
                onChange={(event) => {
                  setSmtpUsername(event.target.value);
                  setDirty(true);
                }}
              />
            }
          />
          <SettingsRow
            label="SMTP password"
            helper="Encrypted at rest before persistence."
            control={
              <TextInput
                label="SMTP password"
                type="password"
                value={smtpPassword}
                onChange={(event) => {
                  setSmtpPassword(event.target.value);
                  setDirty(true);
                }}
              />
            }
          />
        </>
      ) : null}
      {kind === "resend" ? (
        <SettingsRow
          label="Resend API key"
          helper="Encrypted at rest before persistence."
          control={
            <TextInput
              label="Resend API key"
              type="password"
              placeholder="re_..."
              value={resendKey}
              onChange={(event) => {
                setResendKey(event.target.value);
                setDirty(true);
              }}
            />
          }
        />
      ) : null}
      {saveError ? <InlineAlert variant="error" message={formatErrorCode(saveError)} /> : null}
      {showSaveBar ? (
        <SaveBar
          dirtyCount={1}
          saving={saving}
          onSave={async () => {
            if (!kind) {
              setSaveError("provider.email_invalid");
              return;
            }
            if (!fromAddress.trim() || !fromName.trim()) {
              setSaveError("provider.email_invalid");
              return;
            }
            if (kind === "smtp" && (!smtpHost.trim() || !smtpPort.trim())) {
              setSaveError("provider.email_invalid");
              return;
            }
            if (kind === "resend" && !resendKey.trim()) {
              setSaveError("provider.email_invalid");
              return;
            }
            setSaving(true);
            setSaveError("");
            try {
              let config = "";
              if (kind === "smtp") {
                config = JSON.stringify({
                  host: smtpHost.trim(),
                  port: Number.parseInt(smtpPort, 10) || 0,
                  username: smtpUsername,
                  password: smtpPassword,
                });
              } else if (kind === "resend") {
                config = resendKey.trim();
              }
              await saveEmailProviderConfig({
                kind,
                from_address: fromAddress,
                from_name: fromName,
                config,
              });
              setDirty(false);
              await queryClient.invalidateQueries({ queryKey: ["provider-config", "email"] });
            } catch (error) {
              setSaveError(error instanceof Error ? error.message : "provider.email_save_failed");
            } finally {
              setSaving(false);
            }
          }}
          onDiscard={() => {
            setKind(provider.kind || "");
            setFromAddress("");
            setFromName("");
            setSmtpHost("");
            setSmtpPort("");
            setSmtpUsername("");
            setSmtpPassword("");
            setResendKey("");
            setDirty(false);
            setSaveError("");
          }}
        />
      ) : null}
    </ProviderSettingsShell>
  );
}

export function UpstreamProviderScreen() {
  const queryClient = useQueryClient();
  const providerQuery = useQuery({
    queryKey: ["provider-config", "upstream"],
    queryFn: getUpstreamProviderConfig,
    retry: false,
  });
  const provider = (demoStateEnabled() ? demoUpstreamProvider : providerQuery.data) ?? {
    kind: "google",
    configured: false,
    healthy: false,
  };
  const [dirty, setDirty] = useState(false);
  const [testing, setTesting] = useState(false);
  const [clientID, setClientID] = useState("");
  const [clientSecret, setClientSecret] = useState("");
  const [enabled, setEnabled] = useState(provider.configured);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState("");
  if (providerQuery.error?.message === "auth.forbidden") return <ErrorPage code="403" />;
  if (providerQuery.isLoading && !demoStateEnabled()) return <LoadingState />;
  if (providerQuery.isError && !demoStateEnabled()) {
    return (
      <ErrorState
        title="Couldn't load Google upstream provider."
        body="The provider request failed. Retry after checking tenant permissions."
        retry={() => void providerQuery.refetch()}
      />
    );
  }
  return (
    <ProviderSettingsShell
      title="Google upstream"
      provider={provider}
      diagnostic="Try OAuth round-trip"
      testing={testing}
      onDiagnostic={async () => {
        setTesting(true);
        setSaveError("");
        try {
          await testUpstreamProviderConfig();
          await queryClient.invalidateQueries({ queryKey: ["provider-config", "upstream"] });
        } catch (error) {
          setSaveError(error instanceof Error ? error.message : "provider.upstream_test_failed");
        } finally {
          setTesting(false);
        }
      }}
    >
      <SettingsRow
        label="Google client ID"
        helper="Encrypted at rest before persistence."
        control={
          <TextInput
            label="Google client ID"
            placeholder="123.apps.googleusercontent.com"
            value={clientID}
            onChange={(event) => {
              setClientID(event.target.value);
              setDirty(true);
            }}
          />
        }
      />
      <SettingsRow
        label="Google client secret"
        helper="Stored through the encrypted provider-config path."
        control={
          <TextInput
            label="Google client secret"
            type="password"
            value={clientSecret}
            onChange={(event) => {
              setClientSecret(event.target.value);
              setDirty(true);
            }}
          />
        }
      />
      <SettingsRow
        label="Enabled"
        helper="Changes apply to the next sign-in attempt."
        control={
          <Switch
            label="Google enabled"
            checked={enabled}
            onChange={(next) => {
              setEnabled(next);
              setDirty(true);
            }}
          />
        }
      />
      {saveError ? <Toast variant="error" message={saveError} /> : null}
      {dirty ? (
        <SaveBar
          dirtyCount={1}
          saving={saving}
          onSave={async () => {
            setSaving(true);
            setSaveError("");
            try {
              await saveUpstreamProviderConfig({
                client_id: clientID,
                client_secret: clientSecret,
                enabled,
              });
              setDirty(false);
              await queryClient.invalidateQueries({ queryKey: ["provider-config", "upstream"] });
            } catch (error) {
              setSaveError(
                error instanceof Error ? error.message : "provider.upstream_save_failed",
              );
            } finally {
              setSaving(false);
            }
          }}
          onDiscard={() => {
            setClientID("");
            setClientSecret("");
            setEnabled(provider.configured);
            setDirty(false);
            setSaveError("");
          }}
        />
      ) : null}
    </ProviderSettingsShell>
  );
}

function ProviderSettingsShell({
  title,
  provider,
  diagnostic,
  testing,
  onDiagnostic,
  children,
}: {
  title: string;
  provider: ProviderConfigRecord;
  diagnostic: string;
  testing: boolean;
  onDiagnostic: () => void | Promise<void>;
  children: ReactNode;
}) {
  return (
    <div className="grid gap-4">
      <Card
        title={title}
        subtitle={provider.message ?? "Provider health and configuration state."}
        actions={
          <StatusPip
            variant={provider.healthy ? "success" : "warn"}
            label={provider.configured ? "Configured" : "Unconfigured"}
          />
        }
      >
        <Button variant="secondary" loading={testing} onClick={() => void onDiagnostic()}>
          {diagnostic}
        </Button>
      </Card>
      <Card title="Configuration">{children}</Card>
    </div>
  );
}

function TenantDangerScreen({ tenant }: { tenant: TenantRecord }) {
  const queryClient = useQueryClient();
  const [value, setValue] = useState("");
  const [busy, setBusy] = useState<"suspend" | "delete" | "cancel" | null>(null);
  const [error, setError] = useState("");
  const settingString = (value: unknown) =>
    typeof value === "string" || typeof value === "number" ? String(value) : undefined;
  const suspended = typeof tenant.settings?.suspended_at === "string";
  const deletionScheduled = typeof tenant.settings?.deletion_scheduled_at === "string";
  const cancellableUntil = settingString(tenant.settings?.deletion_cancellable_until);
  const hardDeleteAfter = settingString(tenant.settings?.hard_delete_after);
  const refreshTenants = async () => {
    await queryClient.invalidateQueries({ queryKey: ["tenants"] });
  };
  const run = async (action: "suspend" | "delete" | "cancel") => {
    setBusy(action);
    setError("");
    try {
      if (action === "suspend") {
        await (suspended ? resumeTenant(tenant.id) : suspendTenant(tenant.id));
      } else if (action === "delete") {
        await scheduleTenantDeletion(tenant.id, value);
        setValue("");
      } else {
        await cancelTenantDeletion(tenant.id);
      }
      await refreshTenants();
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "tenant.danger_failed");
    } finally {
      setBusy(null);
    }
  };
  return (
    <div className="grid gap-4">
      {error ? <Toast variant="error" message={error} /> : null}
      <Card
        title={suspended ? "Tenant suspended" : "Suspend tenant"}
        subtitle="Reversible. Blocks new sign-ins, ends active sessions, and leaves data intact."
      >
        <Button
          variant={suspended ? "primary" : "destructive"}
          loading={busy === "suspend"}
          disabled={deletionScheduled}
          onClick={() => void run("suspend")}
        >
          {suspended ? "Resume tenant" : "Suspend tenant"}
        </Button>
      </Card>
      <Card
        title={deletionScheduled ? "Deletion scheduled" : "Delete tenant"}
        subtitle="Irreversible after the 7-day cancellation window. Signing keys sunset for 30 days so JWKS consumers can recover cleanly."
      >
        {deletionScheduled ? (
          <div className="grid gap-3">
            <Toast
              variant="warn"
              message={`Tenant deletion scheduled. Cancellation closes ${cancellableUntil ? formatDate(cancellableUntil) : "the 7-day window"}; JWKS serves sunsetting keys until ${hardDeleteAfter ? formatDate(hardDeleteAfter) : "the 30-day sunset"}.`}
            />
            <Button
              variant="secondary"
              loading={busy === "cancel"}
              onClick={() => void run("cancel")}
            >
              Cancel deletion
            </Button>
          </div>
        ) : (
          <div className="grid gap-3">
            <TextInput
              label="Type tenant slug to confirm"
              value={value}
              onChange={(event) => setValue(event.target.value)}
            />
            <Button
              variant="destructive"
              disabled={value !== tenant.slug}
              loading={busy === "delete"}
              onClick={() => void run("delete")}
            >
              Schedule deletion
            </Button>
          </div>
        )}
      </Card>
    </div>
  );
}

function tenantTabTitle(tab: string) {
  const titles: Record<string, string> = {
    projects: "Tenant projects",
    users: "Tenant users",
    "auth-providers": "Auth providers",
    "signing-keys": "Signing keys",
    audit: "Tenant audit",
  };
  return titles[tab] ?? "Tenant overview";
}

function accentLooksValid(value: string) {
  return /^#[0-9a-f]{6}$/i.test(value) && value.toLowerCase() !== ["#", "ffff00"].join("");
}

export function AccountProfile() {
  const forcedState = galleryState();
  const lastPasskeyMode = forcedState === "last-passkey";

  const [addPasskeyOpen, setAddPasskeyOpen] = useState(false);
  const [regenOpen, setRegenOpen] = useState(false);
  const [createPATOpen, setCreatePATOpen] = useState(false);
  const [revokeOthersError, setRevokeOthersError] = useState<string | undefined>();
  const [revokeOthersDone, setRevokeOthersDone] = useState(false);
  const [removePasskeyError, setRemovePasskeyError] = useState<string | undefined>();
  const [revokeSessionError, setRevokeSessionError] = useState<string | undefined>();

  const queryClient = useQueryClient();

  const meQuery = useQuery({ queryKey: ["account-me"], queryFn: getMe, retry: false });
  const passkeysQuery = useQuery({
    queryKey: ["account-passkeys"],
    queryFn: listPasskeys,
    retry: false,
  });
  const sessionsQuery = useQuery({
    queryKey: ["account-sessions"],
    queryFn: listSessions,
    retry: false,
  });
  const mfaQuery = useQuery({
    queryKey: ["account-mfa"],
    queryFn: getMFAFactors,
    retry: false,
  });
  const meQueryData = meQuery.data;
  const isInstanceAdmin = meQueryData?.kind === "instance_admin";
  const tokensQuery = useQuery({
    queryKey: ["account-pats"],
    queryFn: listPersonalAccessTokens,
    retry: false,
    enabled: !isInstanceAdmin,
  });

  const passkeys = Array.isArray(passkeysQuery.data) ? passkeysQuery.data : [];
  const sessions = Array.isArray(sessionsQuery.data) ? sessionsQuery.data : [];
  const tokens = Array.isArray(tokensQuery.data) ? tokensQuery.data : [];
  const mfa =
    mfaQuery.data && typeof mfaQuery.data === "object" && "backup_codes" in mfaQuery.data
      ? mfaQuery.data
      : undefined;
  const me = meQuery.data;

  // Last-passkey guard: locked when this is the user's only sign-in factor
  // (no other passkeys AND no enrolled TOTP).
  const isLastPasskey =
    lastPasskeyMode || (passkeys.length <= 1 && !(mfa?.totp?.enrolled ?? false));

  const handleRemovePasskey = async (id: string) => {
    if (isLastPasskey) {
      setRemovePasskeyError(
        "This is your only sign-in method. Add another before removing this one.",
      );
      return;
    }
    setRemovePasskeyError(undefined);
    try {
      await removePasskey(id);
      await queryClient.invalidateQueries({ queryKey: ["account-passkeys"] });
    } catch (error) {
      const message = error instanceof Error ? error.message : "passkey.delete_failed";
      setRemovePasskeyError(
        message === "passkey.last_remaining"
          ? "This is your only sign-in method. Add another before removing this one."
          : message,
      );
    }
  };
  const handleRevokeOthers = async () => {
    setRevokeOthersError(undefined);
    try {
      await revokeOtherSessions();
      setRevokeOthersDone(true);
      await queryClient.invalidateQueries({ queryKey: ["account-sessions"] });
    } catch (error) {
      setRevokeOthersError(error instanceof Error ? error.message : "auth.session_revoke_failed");
    }
  };
  const handleRevokeSession = async (id: string) => {
    setRevokeSessionError(undefined);
    try {
      await revokeSession(id);
      await queryClient.invalidateQueries({ queryKey: ["account-sessions"] });
    } catch (error) {
      setRevokeSessionError(error instanceof Error ? error.message : "session.revoke_failed");
    }
  };
  const handleRevokePAT = async (id: string) => {
    try {
      await revokePAT(id);
      await queryClient.invalidateQueries({ queryKey: ["account-pats"] });
    } catch {
      // Surfaced via tokensQuery refetch; no inline toast yet.
    }
  };

  return (
    <div className="grid gap-6">
      <PageHeader
        title="Account"
        subtitle={
          lastPasskeyMode
            ? "Last-passkey self-removal guard."
            : "Manage your passkeys, second factors, sessions, and personal access tokens."
        }
      />
      {me?.email ? <AccountIdentityStrip me={me} /> : null}
      <PasskeysSection
        passkeys={passkeys}
        loading={passkeysQuery.isLoading}
        error={passkeysQuery.isError ? "Couldn’t load passkeys." : undefined}
        forbidden={passkeysQuery.error?.message === "auth.forbidden"}
        onRetry={() => void passkeysQuery.refetch()}
        isLast={isLastPasskey}
        lastPasskeyMode={lastPasskeyMode}
        onAdd={() => setAddPasskeyOpen(true)}
        onRemove={(id) => void handleRemovePasskey(id)}
        removeError={removePasskeyError}
      />
      <TwoFactorSection mfa={mfa} onRegenerate={() => setRegenOpen(true)} />
      <ActiveSessionsSection
        sessions={sessions}
        loading={sessionsQuery.isLoading}
        error={sessionsQuery.isError ? "Couldn’t load sessions." : undefined}
        forbidden={sessionsQuery.error?.message === "auth.forbidden"}
        onRetry={() => void sessionsQuery.refetch()}
        onRevoke={(id) => void handleRevokeSession(id)}
        onRevokeOthers={() => void handleRevokeOthers()}
        onSignOut={() => void signOut()}
        revokeOthersDone={revokeOthersDone}
        revokeOthersError={revokeOthersError}
        revokeSessionError={revokeSessionError}
      />
      {isInstanceAdmin ? null : (
        <PersonalAccessTokensSection
          tokens={tokens}
          loading={tokensQuery.isLoading}
          error={tokensQuery.isError ? "Couldn’t load tokens." : undefined}
          forbidden={tokensQuery.error?.message === "auth.forbidden"}
          onRetry={() => void tokensQuery.refetch()}
          onCreate={() => setCreatePATOpen(true)}
          onRevoke={(id) => void handleRevokePAT(id)}
        />
      )}
      <SectionCard
        icon={<Icons.Sun className="h-4 w-4" />}
        title="Theme"
        description="Persisted to your actor metadata."
      >
        <div className="px-5 py-4">
          <ThemeToggle />
        </div>
      </SectionCard>
      <AddPasskeyModal
        open={addPasskeyOpen}
        onClose={() => setAddPasskeyOpen(false)}
        userID={me?.id ?? CURRENT_USER_ID}
        rpID={PASSKEY_RP_ID}
      />
      <RegenerateBackupCodesModal
        open={regenOpen}
        onClose={() => setRegenOpen(false)}
        userID={me?.id ?? CURRENT_USER_ID}
      />
      <CreatePATModal open={createPATOpen} onClose={() => setCreatePATOpen(false)} />
    </div>
  );
}

function AccountIdentityStrip({ me }: { me: MeRecord }) {
  const initials = me.display_name
    .split(/[\s@.]/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0].toUpperCase())
    .join("");
  return (
    <header className="flex items-center gap-5 rounded-[var(--radius-md)] border border-border-subtle bg-bg-surface px-6 py-5">
      <span
        aria-hidden="true"
        className="grid h-16 w-16 place-items-center rounded-[var(--radius-pill)] bg-accent-primary-mu text-[22px] font-semibold text-accent-primary"
      >
        {initials || "—"}
      </span>
      <div className="grid min-w-0 gap-1.5">
        <h2 className="truncate text-[20px] font-semibold leading-tight">{me.display_name}</h2>
        <div className="flex flex-wrap items-center gap-3 text-[14px] text-text-secondary">
          <span className="truncate">{me.email}</span>
          <span aria-hidden="true" className="text-text-tertiary">
            ·
          </span>
          <IdentifierPill value={me.sub} label="User ID" />
        </div>
      </div>
    </header>
  );
}

function SectionCard({
  icon,
  title,
  description,
  action,
  children,
}: {
  icon: ReactNode;
  title: string;
  description?: string;
  action?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className="rounded-[var(--radius-md)] border border-border-subtle bg-bg-surface">
      <header className="flex items-center justify-between gap-4 border-b border-border-subtle px-5 py-4">
        <div className="grid gap-1">
          <h2 className="flex items-center gap-2 text-[16px] font-semibold leading-tight">
            <span aria-hidden="true" className="text-text-primary">
              {icon}
            </span>
            {title}
          </h2>
          {description ? <p className="text-[13px] text-text-secondary">{description}</p> : null}
        </div>
        {action ? <div className="flex shrink-0 items-center gap-2">{action}</div> : null}
      </header>
      {children}
    </section>
  );
}

function SectionRow({
  icon,
  primary,
  meta,
  trailing,
  last,
}: {
  icon?: ReactNode;
  primary: ReactNode;
  meta?: ReactNode;
  trailing?: ReactNode;
  last?: boolean;
}) {
  return (
    <div
      className={cn(
        "flex items-center gap-3 px-5 py-3.5",
        !last && "border-b border-border-subtle",
      )}
    >
      {icon ? (
        <span aria-hidden="true" className="shrink-0">
          {icon}
        </span>
      ) : null}
      <div className="grid min-w-0 flex-1 gap-0.5">
        <div className="flex items-center gap-2 text-[14px] font-medium text-text-primary">
          {primary}
        </div>
        {meta ? <div className="text-[12px] text-text-tertiary">{meta}</div> : null}
      </div>
      {trailing ? <div className="flex shrink-0 items-center gap-2">{trailing}</div> : null}
    </div>
  );
}

function PasskeysSection({
  passkeys,
  loading,
  error,
  forbidden,
  onRetry,
  isLast,
  lastPasskeyMode,
  onAdd,
  onRemove,
  removeError,
}: {
  passkeys: PasskeyRecord[];
  loading: boolean;
  error?: string;
  forbidden: boolean;
  onRetry: () => void;
  isLast: boolean;
  lastPasskeyMode: boolean;
  onAdd: () => void;
  onRemove: (id: string) => void;
  removeError?: string;
}) {
  return (
    <SectionCard
      icon={<Icons.KeyRound className="h-4 w-4" />}
      title="Passkeys"
      description="Sign in without a password using your device."
      action={
        <Button variant="primary" data-primary-create="true" onClick={onAdd}>
          {lastPasskeyMode || passkeys.length <= 1 ? "Add another passkey" : "Add passkey"}
        </Button>
      }
    >
      {lastPasskeyMode ? (
        <div className="flex items-start gap-3 border-b border-status-warn bg-bg-code px-5 py-3.5">
          <Icons.TriangleAlert
            className="mt-0.5 h-4 w-4 shrink-0 text-status-warn"
            aria-hidden="true"
          />
          <div className="grid gap-1 text-[13px]">
            <p className="font-medium text-text-primary">Last passkey on this account</p>
            <p className="text-text-secondary">
              Add another sign-in method before removing this one — otherwise you&apos;ll lock
              yourself out.
            </p>
          </div>
        </div>
      ) : null}
      {loading ? (
        <div className="px-5 py-6 text-[13px] text-text-tertiary">Loading passkeys…</div>
      ) : forbidden ? (
        <div className="flex items-center gap-2 px-5 py-6 text-[13px] text-text-tertiary">
          <Icons.Lock className="h-3.5 w-3.5" aria-hidden="true" />
          You don&apos;t have access to passkeys here.
        </div>
      ) : error ? (
        <div className="flex items-center justify-between gap-3 px-5 py-6 text-[13px]">
          <span className="flex items-center gap-2 text-text-primary">
            <Icons.XCircle className="h-4 w-4 text-status-error" aria-hidden="true" />
            {error}
          </span>
          <button
            type="button"
            onClick={onRetry}
            className="text-[12px] text-accent-primary hover:underline"
          >
            Try again
          </button>
        </div>
      ) : passkeys.length === 0 ? (
        <div className="px-5 py-6 text-[13px] text-text-tertiary">
          No passkeys yet. Add one above to enable sign-in without a password.
        </div>
      ) : (
        passkeys.map((passkey, index) => (
          <SectionRow
            key={passkey.id}
            icon={<Icons.KeyRound className="h-[18px] w-[18px] text-accent-primary" />}
            primary={
              <>
                <span>{passkey.label}</span>
                {passkey.this_device ? <Tag variant="success">this device</Tag> : null}
              </>
            }
            meta={passkeyMeta(passkey)}
            trailing={
              isLast ? (
                <Tooltip label="This is your only sign-in method. Add another before removing this one.">
                  <span className="opacity-40">
                    <IconButton
                      label="Remove passkey"
                      icon={<Icons.Trash2 className="h-4 w-4" />}
                      variant="destructive"
                      disabled
                    />
                  </span>
                </Tooltip>
              ) : (
                <IconButton
                  label="Remove passkey"
                  icon={<Icons.Trash2 className="h-4 w-4" />}
                  variant="destructive"
                  onClick={() => onRemove(passkey.id)}
                />
              )
            }
            last={index === passkeys.length - 1}
          />
        ))
      )}
      {removeError ? (
        <div className="border-t border-border-subtle px-5 py-3">
          <Toast variant="warn" message={removeError} />
        </div>
      ) : null}
    </SectionCard>
  );
}

function passkeyMeta(passkey: PasskeyRecord): string {
  const created = `Added ${formatRelativeTime(passkey.created_at)}`;
  if (passkey.last_used_at) {
    return `${created} · last used ${formatRelativeTime(passkey.last_used_at)}`;
  }
  return `${created} · never used`;
}

function TwoFactorSection({
  mfa,
  onRegenerate,
}: {
  mfa: MFAFactorsRecord | undefined;
  onRegenerate: () => void;
}) {
  const totpEnrolled = mfa?.totp?.enrolled ?? false;
  const remaining = mfa?.backup_codes.remaining ?? 0;
  const total = mfa?.backup_codes.total ?? 0;
  const backupMeta =
    total > 0
      ? `${String(remaining)} of ${String(total)} codes remain. Regenerate to invalidate the rest.`
      : "Regenerate to mint a fresh batch of backup codes.";
  return (
    <SectionCard
      icon={<Icons.ShieldCheck className="h-4 w-4" />}
      title="Two-factor authentication"
      description="A second factor for password sign-in."
      action={totpEnrolled ? <Tag variant="success">enrolled</Tag> : null}
    >
      {totpEnrolled ? (
        <SectionRow
          icon={<Icons.Smartphone className="h-[18px] w-[18px] text-text-secondary" />}
          primary="TOTP authenticator app"
          meta={
            mfa?.totp?.confirmed_at
              ? `Confirmed ${formatRelativeTime(mfa.totp.confirmed_at)}`
              : "Confirmed."
          }
        />
      ) : null}
      <SectionRow
        icon={<Icons.Hash className="h-[18px] w-[18px] text-text-secondary" />}
        primary="Backup codes"
        meta={backupMeta}
        trailing={
          <Button variant="ghost" size="sm" onClick={onRegenerate}>
            Regenerate
          </Button>
        }
        last
      />
    </SectionCard>
  );
}

function ActiveSessionsSection({
  sessions,
  loading,
  error,
  forbidden,
  onRetry,
  onRevoke,
  onRevokeOthers,
  onSignOut,
  revokeOthersDone,
  revokeOthersError,
  revokeSessionError,
}: {
  sessions: SessionRecord[];
  loading: boolean;
  error?: string;
  forbidden: boolean;
  onRetry: () => void;
  onRevoke: (id: string) => void;
  onRevokeOthers: () => void;
  onSignOut: () => void;
  revokeOthersDone: boolean;
  revokeOthersError?: string;
  revokeSessionError?: string;
}) {
  const otherCount = sessions.filter((session) => !session.current).length;
  return (
    <SectionCard
      icon={<Icons.Monitor className="h-4 w-4" />}
      title="Active sessions"
      description="Sign-in surfaces currently authenticated as you."
      action={
        <div className="flex items-center gap-2">
          <Button
            variant="secondary"
            size="sm"
            onClick={onRevokeOthers}
            disabled={otherCount === 0 && sessions.length > 0}
          >
            Sign out other sessions
          </Button>
          <Button variant="destructive" size="sm" onClick={onSignOut}>
            Sign out
          </Button>
        </div>
      }
    >
      {loading ? (
        <div className="px-5 py-6 text-[13px] text-text-tertiary">Loading sessions…</div>
      ) : forbidden ? (
        <div className="flex items-center gap-2 px-5 py-6 text-[13px] text-text-tertiary">
          <Icons.Lock className="h-3.5 w-3.5" aria-hidden="true" />
          You don&apos;t have access to sessions here.
        </div>
      ) : error ? (
        <div className="flex items-center justify-between gap-3 px-5 py-6 text-[13px]">
          <span className="flex items-center gap-2 text-text-primary">
            <Icons.XCircle className="h-4 w-4 text-status-error" aria-hidden="true" />
            {error}
          </span>
          <button
            type="button"
            onClick={onRetry}
            className="text-[12px] text-accent-primary hover:underline"
          >
            Try again
          </button>
        </div>
      ) : sessions.length === 0 ? (
        <div className="px-5 py-6 text-[13px] text-text-tertiary">No active sessions.</div>
      ) : (
        sessions.map((session, index) => {
          const Icon =
            session.auth_kind === "pat"
              ? Icons.Terminal
              : isMobileUserAgent(session.user_agent)
                ? Icons.Smartphone
                : session.current
                  ? Icons.Monitor
                  : Icons.Laptop;
          return (
            <SectionRow
              key={session.id}
              icon={
                <Icon
                  className={cn(
                    "h-[18px] w-[18px]",
                    session.current ? "text-accent-primary" : "text-text-secondary",
                  )}
                />
              }
              primary={
                <>
                  <span>{sessionLabel(session)}</span>
                  {session.current ? <Tag variant="success">this session</Tag> : null}
                  {session.auth_kind === "pat" ? <Tag variant="neutral">PAT auth</Tag> : null}
                </>
              }
              meta={sessionMeta(session)}
              trailing={
                session.current ? null : (
                  <IconButton
                    label="Revoke session"
                    icon={<Icons.Trash2 className="h-4 w-4" />}
                    variant="destructive"
                    onClick={() => onRevoke(session.id)}
                  />
                )
              }
              last={index === sessions.length - 1}
            />
          );
        })
      )}
      {revokeOthersDone ? (
        <div className="border-t border-border-subtle px-5 py-3">
          <Toast variant="info" message="Other sessions signed out." />
        </div>
      ) : null}
      {revokeOthersError ? (
        <div className="border-t border-border-subtle px-5 py-3">
          <Toast variant="error" message={revokeOthersError} />
        </div>
      ) : null}
      {revokeSessionError ? (
        <div className="border-t border-border-subtle px-5 py-3">
          <Toast variant="error" message={revokeSessionError} />
        </div>
      ) : null}
    </SectionCard>
  );
}

function sessionLabel(session: SessionRecord): string {
  if (session.auth_kind === "pat") {
    return session.user_agent ?? "Personal access token";
  }
  return session.user_agent ? truncate(session.user_agent, 60) : "Browser session";
}

function sessionMeta(session: SessionRecord): string {
  const lastSeen = `last activity ${formatRelativeTime(session.last_seen_at)}`;
  if (session.ip) {
    return `${session.ip} · ${lastSeen}`;
  }
  return lastSeen;
}

function isMobileUserAgent(ua?: string): boolean {
  if (!ua) return false;
  return /iPhone|Android|Mobile/.test(ua);
}

function truncate(value: string, max: number): string {
  if (value.length <= max) return value;
  return value.slice(0, max - 1) + "…";
}

function PersonalAccessTokensSection({
  tokens,
  loading,
  error,
  forbidden,
  onRetry,
  onCreate,
  onRevoke,
}: {
  tokens: PersonalAccessTokenRecord[];
  loading: boolean;
  error?: string;
  forbidden: boolean;
  onRetry: () => void;
  onCreate: () => void;
  onRevoke: (id: string) => void;
}) {
  return (
    <SectionCard
      icon={<Icons.Key className="h-4 w-4" />}
      title="Personal access tokens"
      description="Tokens act as you. Treat them as passwords."
      action={
        <Button variant="primary" data-primary-create="true" onClick={onCreate}>
          Create PAT
        </Button>
      }
    >
      {loading ? (
        <div className="px-5 py-6 text-[13px] text-text-tertiary">Loading tokens…</div>
      ) : forbidden ? (
        <div className="flex items-center gap-2 px-5 py-6 text-[13px] text-text-tertiary">
          <Icons.Lock className="h-3.5 w-3.5" aria-hidden="true" />
          You don&apos;t have access to personal access tokens here.
        </div>
      ) : error ? (
        <div className="flex items-center justify-between gap-3 px-5 py-6 text-[13px]">
          <span className="flex items-center gap-2 text-text-primary">
            <Icons.XCircle className="h-4 w-4 text-status-error" aria-hidden="true" />
            {error}
          </span>
          <button
            type="button"
            onClick={onRetry}
            className="text-[12px] text-accent-primary hover:underline"
          >
            Try again
          </button>
        </div>
      ) : tokens.length === 0 ? (
        <div className="px-5 py-6 text-[13px] text-text-tertiary">
          No personal access tokens yet. Create one above.
        </div>
      ) : (
        tokens.map((token, index) => {
          const revoked = !!token.revoked_at;
          const lastUsed = token.last_used_at
            ? `last used ${formatRelativeTime(token.last_used_at)}`
            : "never used";
          return (
            <div
              key={token.id}
              className={cn(
                "flex items-center gap-4 px-5 py-3.5",
                index !== tokens.length - 1 && "border-b border-border-subtle",
                revoked && "opacity-70",
              )}
            >
              <div className="grid min-w-0 flex-1 gap-1">
                <div className="flex items-center gap-2 text-[14px] font-medium">
                  <span className={cn(revoked ? "text-text-secondary" : "text-text-primary")}>
                    {token.name}
                  </span>
                  {revoked ? (
                    <Tag variant="neutral">revoked</Tag>
                  ) : (
                    <Tag variant="success">active</Tag>
                  )}
                </div>
                <div className="flex flex-wrap items-center gap-2 text-[12px] text-text-tertiary">
                  {token.last4 ? (
                    <IdentifierPill value={`cypra_pat_…${token.last4}`} label="Token suffix" />
                  ) : null}
                  <span>
                    Created {formatRelativeTime(token.created_at)} · {lastUsed}
                  </span>
                </div>
              </div>
              {revoked ? null : (
                <IconButton
                  label="Revoke token"
                  icon={<Icons.Trash2 className="h-4 w-4" />}
                  variant="destructive"
                  onClick={() => onRevoke(token.id)}
                />
              )}
            </div>
          );
        })
      )}
    </SectionCard>
  );
}

export interface PlaceholderPageProps {
  title: string;
  subtitle?: string;
  emptyTitle?: string;
  emptyBody?: string;
  primaryCta?: { label: string; onClick: () => void } | null;
}

export function PlaceholderPage({
  title,
  subtitle,
  emptyTitle,
  emptyBody,
  primaryCta,
}: PlaceholderPageProps) {
  return (
    <div>
      <PageHeader title={title} subtitle={subtitle} />
      <EmptyState
        title={emptyTitle ?? `No ${title.toLowerCase()} yet.`}
        body={emptyBody}
        action={
          primaryCta ? (
            <Button variant="primary" onClick={primaryCta.onClick}>
              {primaryCta.label}
            </Button>
          ) : undefined
        }
      />
    </div>
  );
}

export function ErrorPage({ code }: { code: "403" | "404" | "500" | "503" }) {
  const [ready, setReady] = useState(false);
  useEffect(() => {
    if (code !== "503") return undefined;
    const timer = window.setInterval(() => {
      void getReady().then(setReady);
    }, 30_000);
    return () => window.clearInterval(timer);
  }, [code]);
  const copy: Record<typeof code, { title: string; body: ReactNode }> = {
    "403": {
      title: "You don't have access to this.",
      body: "Ask the tenant owner for the missing permission.",
    },
    "404": {
      title: "We couldn't find that page.",
      body: "The link may be wrong, or this resource was deleted.",
    },
    "500": {
      title: "Something on Cypra failed.",
      body: (
        <span>
          The error has been logged with request ID{" "}
          <IdentifierPill value="req_01JY0000000000000000000000" label="Request ID" />. Tell your
          operator.
        </span>
      ),
    },
    "503": {
      title: ready ? "Cypra is ready." : "Cypra is briefly unavailable.",
      body: "Migrations or master-key rotation may be in progress. Try again in a minute.",
    },
  };
  return (
    <ErrorState
      title={copy[code].title}
      body={
        typeof copy[code].body === "string" ? copy[code].body : "The request ID is shown below."
      }
    />
  );
}

export function SetupStateGallery() {
  return (
    <div className="grid gap-4">
      <PageHeader title="Setup states" />
      <LoadingState />
      <ErrorState
        title="Setup token expired."
        body="Run cypra admin reset-bootstrap to issue a new token."
      />
      <SetupWizard token="cypra_setup_0123456789abcdef" step="admin" />
    </div>
  );
}

export function OverviewStates() {
  return (
    <div className="grid gap-4">
      <PageHeader title="Overview states" />
      <DashboardOverview />
      <LoadingState />
      <ErrorState
        title="Couldn't load tenant count."
        body="The tenant tile failed. Other tiles are still current."
      />
      <EmptyState
        title="You're a member of this tenant, but you don't have permissions to see anything yet."
        body="Ask the tenant owner."
      />
    </div>
  );
}

export function AccountStates() {
  return (
    <div className="grid gap-4">
      <PageHeader title="Account states" />
      <AccountProfile />
      <Toast
        variant="warn"
        message="This is your only sign-in method. Add another before removing it."
      />
      <MaskedSecret name="pat" value="cypra_pat_once" />
      <TextInput label="Validation" error="Token name is required." />
    </div>
  );
}

type CommandItem = { label: string } & (
  | { kind: "navigate"; path: string }
  | { kind: "action"; action: () => void | Promise<void> }
);

function instanceCommandItems(): CommandItem[] {
  return [
    { kind: "navigate", label: "Go to overview", path: "/dashboard" },
    { kind: "navigate", label: "Go to tenants", path: "/dashboard/tenants" },
    { kind: "navigate", label: "Go to instance admins", path: "/dashboard/instance/admins" },
    { kind: "navigate", label: "Go to instance audit", path: "/dashboard/instance/audit" },
    { kind: "navigate", label: "Go to diagnostics", path: "/dashboard/instance/diagnostics" },
    { kind: "navigate", label: "Go to account", path: "/dashboard/account" },
    { kind: "action", label: "Sign out", action: signOut },
  ];
}

function tenantCommandItems(slug: string): CommandItem[] {
  const base = `/dashboard/tenants/${slug}`;
  return [
    { kind: "navigate", label: "Go to overview", path: base },
    { kind: "navigate", label: "Go to projects", path: `${base}/projects` },
    { kind: "navigate", label: "Go to users", path: `${base}/users` },
    { kind: "navigate", label: "Go to auth providers", path: `${base}/auth-providers` },
    { kind: "navigate", label: "Go to signing keys", path: `${base}/signing-keys` },
    { kind: "navigate", label: "Go to audit", path: `${base}/audit` },
    { kind: "navigate", label: "Go to settings", path: `${base}/settings/branding` },
    { kind: "navigate", label: "Switch tenant (Tenants)", path: "/dashboard/tenants" },
    { kind: "navigate", label: "Go to account", path: "/dashboard/account" },
    { kind: "action", label: "Sign out", action: signOut },
  ];
}

export function CommandOverlay({
  open,
  onClose,
  scope = "instance",
  tenantSlug,
}: {
  open: boolean;
  onClose: () => void;
  scope?: "instance" | "tenant";
  tenantSlug?: string;
}) {
  const items =
    scope === "tenant" && tenantSlug ? tenantCommandItems(tenantSlug) : instanceCommandItems();
  const [query, setQuery] = useState("");
  useEffect(() => {
    if (!open) setQuery("");
  }, [open]);
  if (!open) return null;
  const q = query.trim().toLowerCase();
  const filtered = q ? items.filter((item) => item.label.toLowerCase().includes(q)) : items;
  const run = (item: CommandItem) => {
    onClose();
    if (item.kind === "navigate") {
      window.location.assign(item.path);
    } else {
      void item.action();
    }
  };
  return (
    <Modal title="Command palette" open={open} onClose={onClose} size="md">
      <TextInput
        label="Search"
        autoFocus
        data-autofocus="true"
        placeholder="Jump to route"
        value={query}
        onChange={(event) => setQuery(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === "Enter" && filtered[0]) {
            event.preventDefault();
            run(filtered[0]);
          } else if (event.key === "Escape") {
            event.preventDefault();
            onClose();
          }
        }}
      />
      <div className="mt-4 grid gap-2">
        {filtered.length === 0 ? (
          <p className="px-2 py-1 text-[13px] text-text-secondary">No matches.</p>
        ) : (
          filtered.map((item) => (
            <Button key={item.label} variant="ghost" onClick={() => run(item)}>
              {item.label}
            </Button>
          ))
        )}
      </div>
    </Modal>
  );
}

export function ShortcutOverlay({ open, onClose }: { open: boolean; onClose: () => void }) {
  if (!open) return null;
  return (
    <Modal title="Keyboard shortcuts" open={open} onClose={onClose} size="md">
      <dl className="grid grid-cols-2 gap-3 text-[13px]">
        <dt>cmd/ctrl + k</dt>
        <dd>Open command palette</dd>
        <dt>cmd/ctrl + /</dt>
        <dd>Open shortcuts</dd>
        <dt>g then o / u / p / t / m / a / s / n</dt>
        <dd>Go to overview, users, projects, auth methods, members, audit, settings, account</dd>
        <dt>/</dt>
        <dd>Focus search</dd>
        <dt>c</dt>
        <dd>Primary create</dd>
        <dt>escape</dt>
        <dd>Close overlay</dd>
      </dl>
    </Modal>
  );
}

export function FooterHelp({ onHelp }: { onHelp: () => void }) {
  return (
    <button
      type="button"
      className="fixed bottom-4 right-4 inline-flex h-10 w-10 items-center justify-center rounded-[var(--radius-pill)] border border-border-default bg-bg-elevated text-text-secondary shadow-[var(--shadow-md)]"
      aria-label="Show keyboard shortcuts"
      onClick={onHelp}
    >
      ?
    </button>
  );
}

export function VersionToast({
  bootVersion,
  currentVersion,
}: {
  bootVersion?: string;
  currentVersion?: string;
}) {
  if (!bootVersion || !currentVersion || bootVersion === currentVersion) return null;
  return (
    <div className="fixed bottom-4 left-4">
      <Toast
        variant="info"
        message="A new version is available - refresh to update"
        action={
          <Button size="sm" onClick={() => window.location.reload()}>
            Refresh
          </Button>
        }
      />
    </div>
  );
}

export function ErrorIcon() {
  return <Icons.XCircle className="h-4 w-4 text-status-error" />;
}
