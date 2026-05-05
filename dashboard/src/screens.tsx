import { useEffect, useMemo, useState, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";

import {
  demoteInstanceAdmin,
  getEmailProviderConfig,
  getInstanceDiagnostics,
  getReady,
  getUpstreamProviderConfig,
  listInstanceAdmins,
  listPersonalAccessTokens,
  listProjects,
  listTenants,
  listUsers,
  saveEmailProviderConfig,
  saveTenantBranding,
  saveUpstreamProviderConfig,
  type InstanceAdminRecord,
  type InstanceDiagnosticsRecord,
  type PersonalAccessTokenRecord,
  type ProjectRecord,
  type ProviderConfigRecord,
  type TenantRecord,
  type UserRecord,
} from "@/api";
import {
  AuditEntry,
  BackupCodeGrid,
  Button,
  Card,
  CodeBlock,
  ContextBadge,
  EmptyState,
  ErrorState,
  IdentifierPill,
  KeyRotationTimeline,
  ListRow,
  LoadingState,
  MaskedSecret,
  Modal,
  PageHeader,
  PermissionMatrix,
  SaveBar,
  SetupTokenBanner,
  SettingsRow,
  StatusPip,
  Switch,
  Tag,
  TextInput,
  ThemeToggle,
  Toast,
  Tooltip,
} from "@/components";
import { Icons, PasskeyGlyph } from "@/icons";

const defaultAccent = ["#", "767676"].join("");

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
    scopes: ["tenants:read", "users:write"],
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

export function SetupWizard({ token }: { token: string }) {
  const [step, setStep] = useState<"token" | "passkey" | "backup" | "done">("token");
  const codes = [
    "CYPRA-A11Y",
    "CYPRA-B22Y",
    "CYPRA-C33Y",
    "CYPRA-D44Y",
    "CYPRA-E55Y",
    "CYPRA-F66Y",
    "CYPRA-G77Y",
    "CYPRA-H88Y",
    "CYPRA-I99Y",
    "CYPRA-J00Y",
  ];
  const nextEnv = [
    "CYPRA_ISSUER=https://acme.cypra.localhost",
    "CYPRA_CLIENT_ID=client_cypra_acme_console",
    "CYPRA_CLIENT_SECRET=copy-from-project-detail",
    "AUTH_SECRET=dev-secret-change-me",
  ].join("\n");
  return (
    <main className="mx-auto grid min-h-screen w-full max-w-[560px] place-content-center gap-6 p-4">
      <PageHeader
        title="Set up Cypra"
        subtitle="Redeem the bootstrap token, enroll a passkey, and save recovery codes."
      />
      {step === "token" ? (
        <div className="grid gap-4">
          <SetupTokenBanner token={token} />
          <Button variant="primary" onClick={() => setStep("passkey")}>
            Continue
          </Button>
        </div>
      ) : null}
      {step === "passkey" ? (
        <Card title="Enroll passkey" subtitle="Use your security key or platform authenticator.">
          <div className="mb-4 flex items-center gap-3 text-text-secondary">
            <PasskeyGlyph className="h-8 w-8 text-accent-primary" />
            Touch your security key or use your fingerprint.
          </div>
          <Button variant="primary" onClick={() => setStep("backup")}>
            Enroll passkey
          </Button>
          <Button className="ml-3" variant="ghost">
            Retry
          </Button>
        </Card>
      ) : null}
      {step === "backup" ? (
        <div className="grid gap-4">
          <BackupCodeGrid codes={codes} />
          <Button variant="primary" onClick={() => setStep("done")}>
            Go to dashboard
          </Button>
        </div>
      ) : null}
      {step === "done" ? (
        <div className="grid gap-4">
          <EmptyState
            title="Instance admin created."
            body="Your install is ready. Create tenant acme, configure providers, then paste these env vars into the Next.js example."
            action={
              <Button variant="primary" onClick={() => history.pushState(null, "", "/dashboard")}>
                Open dashboard
              </Button>
            }
          />
          <Card
            title="Next step: create your first tenant"
            subtitle="Canonical demo path: tenant acme, Console App project, terminal email, Google upstream stub, then Next.js."
          >
            <CodeBlock code={nextEnv} />
          </Card>
        </div>
      ) : null}
    </main>
  );
}

export function DashboardOverview() {
  return (
    <div>
      <PageHeader
        title="Dashboard"
        subtitle="Current install health and tenant activity."
        action={<ContextBadge />}
      />
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <Metric title="Tenants" value="1" status="Instance scope" />
        <Metric title="Projects" value="0" status="Create your first app" />
        <Metric title="Users" value="1" status="Bootstrap admin" />
        <Card title="Signing key health">
          <KeyRotationTimeline />
        </Card>
      </div>
      <div className="mt-6 grid gap-4 lg:grid-cols-[1fr_360px]">
        <Card title="Recent audit entries" subtitle="Last 10 install events.">
          <AuditEntry action="bootstrapped" resource="instance_admin_root" />
          <AuditEntry action="created" resource="tenant_acme" />
        </Card>
        <EmptyState
          title="No projects yet."
          body="A project lets an app authenticate against this tenant."
          action={<Button variant="primary">Create project</Button>}
        />
      </div>
    </div>
  );
}

function Metric({ title, value, status }: { title: string; value: string; status: string }) {
  return (
    <Card>
      <div className="text-[13px] text-text-secondary">{title}</div>
      <div className="mt-2 text-[32px] font-semibold leading-10">{value}</div>
      <p className="mt-2 text-[13px] text-text-secondary">{status}</p>
    </Card>
  );
}

export function TenantList() {
  const forcedState = new URLSearchParams(window.location.search).get("state");
  const [query, setQuery] = useState("");
  const [sort, setSort] = useState<"slug" | "name" | "users">("slug");
  const tenantsQuery = useQuery({
    queryKey: ["tenants"],
    queryFn: listTenants,
    retry: false,
    enabled: forcedState === null,
  });
  const tenants = forcedState === null ? (tenantsQuery.data ?? []) : demoTenants;
  const visibleTenants = useMemo(() => {
    const normalized = query.trim().toLowerCase();
    return tenants
      .filter((tenant) =>
        normalized === ""
          ? true
          : `${tenant.slug} ${tenant.name}`.toLowerCase().includes(normalized),
      )
      .sort((a, b) => {
        if (sort === "users") return (b.user_count ?? 0) - (a.user_count ?? 0);
        return a[sort].localeCompare(b[sort]);
      });
  }, [query, sort, tenants]);

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

  return (
    <div>
      <PageHeader
        title="Tenants"
        subtitle="Browse and manage every tenant on this Cypra install."
        action={
          <Button data-primary-create="true" variant="primary">
            Create tenant
          </Button>
        }
      />
      <Card>
        <div className="mb-4 flex flex-wrap items-end gap-3">
          <TextInput
            className="min-w-72"
            label="Search tenants"
            type="search"
            placeholder="Search by name or slug"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
          />
          <label className="grid gap-1.5 text-[13px] font-medium text-text-primary">
            Sort by
            <select
              aria-label="Sort tenants"
              className="h-10 rounded-[var(--radius-md)] border border-border-default bg-bg-surface px-3 text-[14px]"
              value={sort}
              onChange={(event) => setSort(event.target.value as typeof sort)}
            >
              <option value="slug">Slug</option>
              <option value="name">Name</option>
              <option value="users">Users</option>
            </select>
          </label>
        </div>
        {forcedState === "empty" || tenants.length === 0 ? (
          <EmptyState
            title="No tenants yet."
            body="Create the first tenant after bootstrap to issue project credentials."
            action={<Button variant="primary">Create tenant</Button>}
          />
        ) : visibleTenants.length === 0 ? (
          <EmptyState
            title={`No tenants match ${query}.`}
            body="Clear the search to return to the full tenant list."
          />
        ) : (
          <div className="overflow-hidden rounded-[var(--radius-md)] border border-border-subtle">
            <table className="w-full border-collapse text-left text-[13px]" data-responsive="stack">
              <thead className="bg-bg-code text-text-secondary">
                <tr>
                  <th className="px-4 py-3 font-medium">Tenant</th>
                  <th className="px-4 py-3 font-medium">Slug</th>
                  <th className="px-4 py-3 font-medium">Members</th>
                  <th className="px-4 py-3 font-medium">Projects</th>
                  <th className="px-4 py-3 font-medium">Users</th>
                  <th className="px-4 py-3 font-medium">State</th>
                </tr>
              </thead>
              <tbody>
                {visibleTenants.map((tenant) => (
                  <tr key={tenant.id} className="border-t border-border-subtle">
                    <td className="px-4 py-3">
                      <a
                        className="font-medium text-text-primary hover:text-accent-primary"
                        href={`/dashboard/tenants/${tenant.slug}`}
                      >
                        {tenant.name}
                      </a>
                    </td>
                    <td className="px-4 py-3">
                      <IdentifierPill value={tenant.slug} label="Tenant slug" />
                    </td>
                    <td className="px-4 py-3 text-text-secondary">{tenant.member_count ?? 0}</td>
                    <td className="px-4 py-3 text-text-secondary">{tenant.project_count ?? 0}</td>
                    <td className="px-4 py-3 text-text-secondary">{tenant.user_count ?? 0}</td>
                    <td className="px-4 py-3">
                      {tenant.email_provider_required ? (
                        <Tag variant="warn">Email required</Tag>
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
      </Card>
    </div>
  );
}

export function TenantDetail({
  slug,
  tab,
  settingsTab,
  detailSlug,
}: {
  slug: string;
  tab: string;
  settingsTab?: string;
  detailSlug?: string;
}) {
  const tenantsQuery = useQuery({ queryKey: ["tenants"], queryFn: listTenants, retry: false });
  const tenant =
    tenantsQuery.data?.find((item) => item.slug === slug) ??
    demoTenants.find((item) => item.slug === slug) ??
    demoTenants[0];
  const currentTab = tab === "overview" ? "overview" : tab;
  return (
    <div className="grid gap-6">
      <TenantHeader tenant={tenant} />
      <TenantTabs slug={slug} active={currentTab} />
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
      {currentTab === "auth-methods" ? <AuthMethodsTab tenant={tenant} /> : null}
      {currentTab === "signing-keys" ? <SigningKeysScreen /> : null}
      {currentTab === "audit" ? <AuditLogScreen scope="tenant" /> : null}
      {currentTab === "settings" ? (
        <TenantSettings tenant={tenant} active={settingsTab ?? "branding"} />
      ) : null}
      {currentTab !== "overview" &&
      currentTab !== "projects" &&
      currentTab !== "users" &&
      currentTab !== "auth-methods" &&
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
          <ContextBadge kind="tenant" label={`Tenant: ${tenant.slug}`} />
        </div>
      }
    />
  );
}

function TenantTabs({ slug, active }: { slug: string; active: string }) {
  const tabs = [
    ["overview", "Overview", `/dashboard/tenants/${slug}`],
    ["projects", "Projects", `/dashboard/tenants/${slug}/projects`],
    ["users", "Users", `/dashboard/tenants/${slug}/users`],
    ["auth-methods", "Auth methods", `/dashboard/tenants/${slug}/auth-methods`],
    ["signing-keys", "Signing keys", `/dashboard/tenants/${slug}/signing-keys`],
    ["audit", "Audit", `/dashboard/tenants/${slug}/audit`],
    ["settings", "Settings", `/dashboard/tenants/${slug}/settings/branding`],
  ] as const;
  return (
    <nav
      className="flex gap-2 overflow-x-auto border-b border-border-subtle"
      aria-label="Tenant tabs"
    >
      {tabs.map(([id, label, href]) => (
        <a
          key={id}
          href={href}
          aria-current={active === id ? "page" : undefined}
          className={`border-b-2 px-3 py-2 text-[13px] ${active === id ? "border-accent-primary text-text-primary" : "border-transparent text-text-secondary hover:text-text-primary"}`}
        >
          {label}
        </a>
      ))}
    </nav>
  );
}

function TenantOverview({ tenant }: { tenant: TenantRecord }) {
  const emailBlocked = tenant.email_provider_required ?? false;
  const disabledButton = (
    <Tooltip label="Configure an email provider before issuing invites.">
      <Button variant="primary" disabled>
        Invite user
      </Button>
    </Tooltip>
  );
  return (
    <div className="grid gap-4">
      {emailBlocked ? (
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
      <div className="grid gap-4 md:grid-cols-3">
        <Metric title="Projects" value={String(tenant.project_count ?? 0)} status="OIDC clients" />
        <Metric
          title="Users"
          value={String(tenant.user_count ?? 0)}
          status="Tenant-scoped identities"
        />
        <Metric
          title="Members"
          value={String(tenant.member_count ?? 0)}
          status="Admins and operators"
        />
      </div>
      <Card title="Admin actions">
        <div className="flex flex-wrap gap-3">
          {emailBlocked ? disabledButton : <Button variant="primary">Invite user</Button>}
          {emailBlocked ? (
            <Tooltip label="Configure an email provider before inviting members.">
              <Button disabled>Invite member</Button>
            </Tooltip>
          ) : (
            <Button>Invite member</Button>
          )}
          <Button variant="secondary">Create project</Button>
        </div>
      </Card>
    </div>
  );
}

function ProjectList({ tenant }: { tenant: TenantRecord }) {
  const projectsQuery = useQuery({
    queryKey: ["projects", tenant.slug],
    queryFn: listProjects,
    retry: false,
  });
  const projects = projectsQuery.data ?? demoProjects;
  if (projectsQuery.error?.message === "auth.forbidden") return <ErrorPage code="403" />;
  if (projectsQuery.isLoading) return <LoadingState />;
  return (
    <Card
      title="Projects"
      subtitle="One project maps to one OIDC client in v1."
      actions={<Button variant="primary">Create project</Button>}
    >
      {projects.length === 0 ? (
        <EmptyState
          title="No projects yet."
          body="Create a project to let an app authenticate against this tenant."
          action={<Button variant="primary">Create project</Button>}
        />
      ) : (
        <div className="overflow-hidden rounded-[var(--radius-md)] border border-border-subtle">
          <table className="w-full text-left text-[13px]">
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
                    <IdentifierPill
                      value={project.client_id ?? `client_${project.slug}`}
                      label="Client ID"
                    />
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
    </Card>
  );
}

function ProjectDetail({ tenant, projectSlug }: { tenant: TenantRecord; projectSlug: string }) {
  const projectsQuery = useQuery({
    queryKey: ["projects", tenant.slug],
    queryFn: listProjects,
    retry: false,
  });
  const project =
    projectsQuery.data?.find((item) => item.slug === projectSlug) ??
    demoProjects.find((item) => item.slug === projectSlug) ??
    demoProjects[0];
  const [redirectUris, setRedirectUris] = useState((project.redirect_uris ?? []).join("\n"));
  const [scopes, setScopes] = useState((project.allowed_scopes ?? ["openid"]).join(" "));
  const [authMethod, setAuthMethod] = useState(
    project.token_endpoint_auth_method ?? "client_secret_basic",
  );
  const [rotateOpen, setRotateOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleteValue, setDeleteValue] = useState("");
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
  return (
    <div className="grid gap-4">
      {project.rotation_in_progress ? (
        <Toast
          variant="warn"
          message="Rotation in progress - downstream apps using the previous secret will fail at the token endpoint after 5 minutes. Coordinate the cutover with your app owner."
        />
      ) : null}
      <Card title={project.name} subtitle="OIDC client configuration for this project.">
        <div className="grid gap-4 lg:grid-cols-2">
          <SettingsRow
            label="Issuer URL"
            helper="Use this as the provider issuer."
            control={
              <IdentifierPill
                value={project.issuer_url ?? `https://${tenant.slug}.cypra.localhost`}
                label="Issuer URL"
              />
            }
          />
          <SettingsRow
            label="Client ID"
            helper="Public identifier for downstream apps."
            control={
              <IdentifierPill
                value={project.client_id ?? `client_${project.slug}`}
                label="Client ID"
              />
            }
          />
          <SettingsRow
            label="Client secret"
            helper="Reveal before copying. It auto-hides after 30 seconds."
            control={
              <MaskedSecret
                name="client_secret"
                value={project.client_secret ?? "client_secret_once"}
              />
            }
          />
          <SettingsRow
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
          <Button variant="secondary" onClick={() => setRotateOpen(true)}>
            Rotate client secret
          </Button>
          <Button variant="destructive" onClick={() => setDeleteOpen(true)}>
            Delete project
          </Button>
        </div>
      </Card>
      <div className="grid gap-4 lg:grid-cols-2">
        <CodeBlock
          code={`AUTH_CYPRA_ISSUER=${project.issuer_url ?? `https://${tenant.slug}.cypra.localhost`} AUTH_CYPRA_ID=${project.client_id ?? `client_${project.slug}`}`}
        />
        <CodeBlock
          code={`go run ./cmd/app -issuer ${project.issuer_url ?? `https://${tenant.slug}.cypra.localhost`}`}
        />
      </div>
      <Modal
        title="Rotate client secret"
        open={rotateOpen}
        onClose={() => setRotateOpen(false)}
        footer={
          <Button variant="primary" onClick={() => setRotateOpen(false)}>
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
          <Button variant="destructive" disabled={deleteValue !== project.slug}>
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
  const users = usersQuery.data ?? demoUsers;
  const visibleUsers = users.filter((user) =>
    query.trim() === "" ? true : user.email.toLowerCase().includes(query.trim().toLowerCase()),
  );
  const emailBlocked = tenant.email_provider_required ?? false;
  if (usersQuery.error?.message === "auth.forbidden") return <ErrorPage code="403" />;
  if (usersQuery.isLoading) return <LoadingState />;
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
            <table className="w-full text-left text-[13px]">
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
      <Modal
        title="Invite user"
        open={inviteOpen}
        onClose={() => setInviteOpen(false)}
        footer={
          <Button variant="primary" onClick={() => setInviteOpen(false)}>
            Send invite
          </Button>
        }
      >
        <TextInput label="Email" type="email" placeholder="user@example.com" />
      </Modal>
      <Modal title="Import users via CLI" open={importOpen} onClose={() => setImportOpen(false)}>
        <CodeBlock code={`cypra import --tenant ${tenant.slug} --file users.json`} />
        <div className="mt-4 rounded-[var(--radius-md)] border border-border-subtle bg-bg-code p-4 text-[13px] text-text-secondary">
          Screencast placeholder lands in v1.1 docs.
        </div>
      </Modal>
    </Card>
  );
}

function UserDetail({ tenant, userId }: { tenant: TenantRecord; userId: string }) {
  const usersQuery = useQuery({
    queryKey: ["users", tenant.slug],
    queryFn: listUsers,
    retry: false,
  });
  const user =
    usersQuery.data?.find((item) => item.id === userId) ??
    demoUsers.find((item) => item.id === userId) ??
    demoUsers[0];
  const [confirm, setConfirm] = useState<string | null>(null);
  return (
    <div className="grid gap-4">
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
          {(user.enrolled_methods ?? []).map((method) => (
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
            className="border-b-2 border-transparent px-3 py-2 text-[13px] text-text-secondary"
            type="button"
          >
            {tab}
          </button>
        ))}
      </nav>
      <EmptyState
        title="No tab records selected."
        body="Select a user tab to inspect its tenant-scoped records."
      />
      <Modal
        title="Confirm user action"
        open={confirm !== null}
        onClose={() => setConfirm(null)}
        footer={
          <Button
            variant={confirm?.includes("Delete") ? "destructive" : "primary"}
            onClick={() => setConfirm(null)}
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

function AuthMethodsTab({ tenant }: { tenant: TenantRecord }) {
  const methods = [
    ["Email + Password", "Password credentials and reset flow", "settings/email", 47],
    ["Magic Link", "Passwordless email sign-in", "settings/email", 18],
    ["Passkeys", "WebAuthn with tenant RP ID", "auth-methods", 9],
    ["TOTP", "Authenticator app second factor", "auth-methods", 12],
    ["Google upstream", "Google OAuth identity provider", "settings/upstream", 5],
    ["OIDC upstreams", "Generic upstream providers", "settings/upstream", 0],
  ] as const;
  const [confirm, setConfirm] = useState<string | null>(null);
  return (
    <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
      {methods.map(([name, description, target, count]) => (
        <Card
          key={name}
          title={name}
          subtitle={description}
          actions={<Tag variant="info">{count} enrolled</Tag>}
        >
          <div className="flex items-center justify-between gap-3">
            <Switch
              label="Enabled"
              checked
              onChange={() => (count > 0 ? setConfirm(name) : undefined)}
            />
            <a
              className="text-[13px] font-medium text-accent-primary"
              href={`/dashboard/tenants/${tenant.slug}/${target}`}
            >
              Configure
            </a>
          </div>
        </Card>
      ))}
      <Modal
        title="Disable auth method?"
        open={confirm !== null}
        onClose={() => setConfirm(null)}
        footer={
          <Button variant="destructive" onClick={() => setConfirm(null)}>
            Disable for next sign-in
          </Button>
        }
      >
        <p className="text-[14px] text-text-secondary">
          Disabling {confirm} affects users at their next sign-in attempt. Existing sessions remain
          governed by session policy.
        </p>
      </Modal>
    </div>
  );
}

function ApiTokensTab() {
  const tokensQuery = useQuery({
    queryKey: ["pats"],
    queryFn: listPersonalAccessTokens,
    retry: false,
  });
  const tokens = tokensQuery.data ?? demoTokens;
  const [createOpen, setCreateOpen] = useState(false);
  const [revealed, setRevealed] = useState(false);
  const [revokeToken, setRevokeToken] = useState<PersonalAccessTokenRecord | null>(null);
  useEffect(() => {
    if (!revealed) return undefined;
    const warn = (event: BeforeUnloadEvent) => {
      event.preventDefault();
      // eslint-disable-next-line @typescript-eslint/no-deprecated
      event.returnValue = "This token is shown only once. Continue without saving?";
    };
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [revealed]);
  if (tokensQuery.error?.message === "auth.forbidden") return <ErrorPage code="403" />;
  if (tokensQuery.isLoading) return <LoadingState />;
  return (
    <Card
      title="API tokens"
      subtitle="Personal access tokens are shown once at creation."
      actions={
        <Button variant="primary" onClick={() => setCreateOpen(true)}>
          Create token
        </Button>
      }
    >
      <div className="grid gap-3">
        {tokens.map((token) => (
          <div
            key={token.id}
            className="flex items-center gap-3 border-b border-border-subtle py-3"
          >
            <span className="flex-1 text-[14px]">{token.name}</span>
            <span className="text-[13px] text-text-secondary">
              {token.scopes.join(", ")} - last used {token.last_used_at ?? "never"}
            </span>
            <Button variant="destructive" size="sm" onClick={() => setRevokeToken(token)}>
              Revoke
            </Button>
          </div>
        ))}
      </div>
      <Modal
        title="Create API token"
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        footer={
          <Button variant="primary" onClick={() => setRevealed(true)}>
            Create token
          </Button>
        }
      >
        <div className="grid gap-4">
          <TextInput label="Token name" placeholder="Deploy automation" />
          <div className="grid gap-2 text-[14px]">
            <label>
              <input
                className="mr-2 accent-[var(--accent-primary)]"
                type="checkbox"
                defaultChecked
              />{" "}
              tenants:read
            </label>
            <label>
              <input className="mr-2 accent-[var(--accent-primary)]" type="checkbox" /> users:write
            </label>
          </div>
          {revealed ? <MaskedSecret name="cypra_pat" value="cypra_pat_once_visible" /> : null}
          {revealed ? (
            <Button
              variant="secondary"
              onClick={() => {
                setRevealed(false);
                setCreateOpen(false);
              }}
            >
              I have copied this token
            </Button>
          ) : null}
        </div>
      </Modal>
      <Modal
        title="Revoke API token"
        open={revokeToken !== null}
        onClose={() => setRevokeToken(null)}
        footer={
          <Button variant="destructive" onClick={() => setRevokeToken(null)}>
            Revoke token
          </Button>
        }
      >
        <p className="text-[14px] text-text-secondary">
          {revokeToken?.name} will stop authenticating API requests immediately.
        </p>
      </Modal>
    </Card>
  );
}

function MembersTab() {
  const [inviteOpen, setInviteOpen] = useState(false);
  const [dirty, setDirty] = useState(false);
  const [revokeInvite, setRevokeInvite] = useState<string | null>(null);
  const members = [
    ["Ada Lovelace", "owner", "today"],
    ["Grace Hopper", "admin", "yesterday"],
  ] as const;
  const invites = [["pending@example.com", "member", "6 days", "Ada Lovelace"]] as const;
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
        <div className="grid gap-3">
          {members.map(([name, role, seen], index) => (
            <div key={name} className="flex items-center gap-3 border-b border-border-subtle py-3">
              <span className="flex-1 text-[14px]">{name}</span>
              <select
                aria-label={`${name} role`}
                className="h-9 rounded-[var(--radius-md)] border border-border-default bg-bg-surface px-3 text-[13px]"
                defaultValue={role}
                onChange={() => setDirty(true)}
              >
                <option value="owner">Owner</option>
                <option value="admin">Admin</option>
                <option value="member">Member</option>
              </select>
              <span className="text-[13px] text-text-secondary">last seen {seen}</span>
              <Tooltip
                label={
                  index === 0
                    ? "You're the last owner. Promote someone else first."
                    : "Remove member"
                }
              >
                <Button variant="destructive" size="sm" disabled={index === 0}>
                  Remove
                </Button>
              </Tooltip>
            </div>
          ))}
        </div>
        {dirty ? <SaveBar dirtyCount={1} /> : null}
      </Card>
      <PermissionMatrix />
      <Card title="Pending invites" subtitle="Unredeemed pending_invitations rows.">
        {invites.map(([email, role, expires, createdBy]) => (
          <div
            key={email}
            className="flex items-center gap-3 border-b border-border-subtle py-3 text-[13px]"
          >
            <span className="flex-1">{email}</span>
            <Tag variant="pending">{role}</Tag>
            <span className="text-text-secondary">expires in {expires}</span>
            <span className="text-text-secondary">created by {createdBy}</span>
            <Button size="sm">Resend</Button>
            <Button size="sm" variant="destructive" onClick={() => setRevokeInvite(email)}>
              Revoke
            </Button>
          </div>
        ))}
      </Card>
      <Modal
        title="Invite member"
        open={inviteOpen}
        onClose={() => setInviteOpen(false)}
        footer={
          <Button variant="primary" onClick={() => setInviteOpen(false)}>
            Send invite
          </Button>
        }
      >
        <div className="grid gap-4">
          <TextInput label="Email" type="email" placeholder="admin@example.com" />
          <label className="grid gap-1.5 text-[13px] font-medium">
            Role
            <select
              aria-label="Invite role"
              className="h-10 rounded-[var(--radius-md)] border border-border-default bg-bg-surface px-3"
            >
              <option>owner</option>
              <option>admin</option>
              <option>member</option>
            </select>
          </label>
        </div>
      </Modal>
      <Modal
        title="Revoke pending invite"
        open={revokeInvite !== null}
        onClose={() => setRevokeInvite(null)}
        footer={
          <Button variant="destructive" onClick={() => setRevokeInvite(null)}>
            Revoke invite
          </Button>
        }
      >
        <p className="text-[14px] text-text-secondary">
          {revokeInvite} will no longer be able to redeem this invite.
        </p>
      </Modal>
    </div>
  );
}

function SigningKeysScreen() {
  const [rotateOpen, setRotateOpen] = useState(false);
  const [kid, setKid] = useState("");
  const targetKid = "kid_active_2026_05";
  return (
    <div className="grid gap-4">
      <Card
        title="Signing keys"
        subtitle="View key overlap, sunset, and rotation timing."
        actions={
          <Button variant="primary" onClick={() => setRotateOpen(true)}>
            Rotate now
          </Button>
        }
      >
        <KeyRotationTimeline />
      </Card>
      <Card title="Key list">
        <div className="overflow-hidden rounded-[var(--radius-md)] border border-border-subtle">
          <table className="w-full text-left text-[13px]">
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
              <tr className="border-t border-border-subtle">
                <td className="px-4 py-3">
                  <IdentifierPill value={targetKid} label="Key ID" />
                </td>
                <td className="px-4 py-3">
                  <StatusPip variant="active" label="active" />
                </td>
                <td className="px-4 py-3 text-text-secondary">2026-05-01</td>
                <td className="px-4 py-3 text-text-secondary">2026-06-01</td>
                <td className="px-4 py-3 text-text-secondary">not set</td>
              </tr>
            </tbody>
          </table>
        </div>
      </Card>
      <Modal
        title="Rotate signing key"
        open={rotateOpen}
        onClose={() => setRotateOpen(false)}
        footer={
          <Button
            variant="primary"
            disabled={kid !== targetKid}
            onClick={() => setRotateOpen(false)}
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
      </Modal>
    </div>
  );
}

export function AuditLogScreen({ scope }: { scope: "tenant" | "instance" }) {
  const [expanded, setExpanded] = useState(false);
  return (
    <Card
      title={scope === "tenant" ? "Tenant audit" : "Instance audit"}
      subtitle="Filter and export append-only audit entries."
    >
      <div className="mb-4 grid gap-3 md:grid-cols-4">
        <TextInput label="Date range" placeholder="last 7 days" />
        <TextInput label="Action" placeholder="tenant.update" />
        <TextInput label="Actor" placeholder="ada@example.com" />
        <TextInput label="Resource kind" placeholder="user" />
      </div>
      <Toast
        variant="info"
        message="Live updates active. Polling every 10 seconds while this page is visible."
        action={<Button size="sm">Refresh</Button>}
      />
      <div className="mt-4">
        <AuditEntry action="updated" resource={scope === "tenant" ? "user_ada" : "tenant_acme"} />
        <Button className="mt-3" size="sm" onClick={() => setExpanded((current) => !current)}>
          {expanded ? "Collapse entry" : "Expand entry"}
        </Button>
      </div>
      {expanded ? (
        <pre className="mt-3 overflow-x-auto rounded-[var(--radius-md)] bg-bg-code p-3 font-mono text-[13px] text-text-identifier">
          {JSON.stringify(
            {
              state_before: { email: "[redacted]" },
              state_after: { email: "[redacted]", role: "admin" },
            },
            null,
            2,
          )}
        </pre>
      ) : null}
      <div className="mt-4">
        <Button variant="primary">Export NDJSON</Button>
      </div>
    </Card>
  );
}

export function InstanceAdminsScreen() {
  const adminsQuery = useQuery({
    queryKey: ["instance-admins"],
    queryFn: listInstanceAdmins,
    retry: false,
  });
  const admins = adminsQuery.data ?? demoInstanceAdmins;
  const [inviteOpen, setInviteOpen] = useState(false);
  const [demote, setDemote] = useState<InstanceAdminRecord | null>(null);
  const [demoteError, setDemoteError] = useState("");
  if (adminsQuery.error?.message === "auth.forbidden") return <ErrorPage code="403" />;
  if (adminsQuery.isLoading) return <LoadingState />;
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
          <table className="w-full text-left text-[13px]">
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
          <div className="flex items-center justify-between text-[13px]">
            <span>ops@example.com</span>
            <Tag variant="pending">instance_admin</Tag>
            <span className="text-text-secondary">expires in 6 days</span>
            <Button size="sm" variant="destructive">
              Revoke
            </Button>
          </div>
        </Card>
      </Card>
      <Modal
        title="Invite instance admin"
        open={inviteOpen}
        onClose={() => setInviteOpen(false)}
        footer={
          <Button variant="primary" onClick={() => setInviteOpen(false)}>
            Send invite
          </Button>
        }
      >
        <TextInput label="Email" type="email" placeholder="admin@example.com" />
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
  const diagnostics = diagnosticsQuery.data ?? demoDiagnostics;
  if (diagnosticsQuery.error?.message === "auth.forbidden") return <ErrorPage code="403" />;
  if (diagnosticsQuery.isLoading) return <LoadingState />;
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
  const tabs = ["branding", "email", "upstream", "members", "api-tokens", "danger"];
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
      {active === "upstream" ? <UpstreamProviderScreen /> : null}
      {active === "api-tokens" ? <ApiTokensTab /> : null}
      {active === "members" ? <MembersTab /> : null}
      {active === "danger" ? <TenantDangerScreen tenant={tenant} /> : null}
      {active !== "branding" &&
      active !== "email" &&
      active !== "upstream" &&
      active !== "api-tokens" &&
      active !== "members" &&
      active !== "danger" ? (
        <PlaceholderPage title={`${active.replace("-", " ")} settings`} />
      ) : null}
    </div>
  );
}

function BrandingTab({ tenant }: { tenant: TenantRecord }) {
  const [displayName, setDisplayName] = useState(tenant.branding?.display_name ?? tenant.name);
  const [accent, setAccent] = useState(tenant.branding?.accent ?? defaultAccent);
  const [poweredBy, setPoweredBy] = useState(tenant.branding?.powered_by ?? true);
  const [logoError, setLogoError] = useState("");
  const [accentError, setAccentError] = useState("");
  const [previewOpen, setPreviewOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState("");
  const dirty =
    displayName !== (tenant.branding?.display_name ?? tenant.name) ||
    accent !== (tenant.branding?.accent ?? defaultAccent) ||
    poweredBy !== (tenant.branding?.powered_by ?? true);

  const validateLogo = (file: File | undefined) => {
    if (!file) return;
    if (file.type === "image/svg+xml") {
      setLogoError("SVG accepted. Sanitization runs before upload.");
      return;
    }
    if (file.type !== "image/png") {
      setLogoError("Use a sanitized SVG or PNG logo.");
      return;
    }
    const image = new Image();
    image.onload = () => {
      const ratio = image.width / image.height;
      const accepted =
        image.width >= 256 && [1, 16 / 9, 21 / 9].some((target) => Math.abs(ratio - target) < 0.08);
      setLogoError(accepted ? "" : "PNG must be at least 256px wide and square, 16:9, or 21:9.");
    };
    image.src = URL.createObjectURL(file);
  };

  const save = async () => {
    if (!accentLooksValid(accent)) {
      setAccentError("Accent must be readable on white and support readable text on accent.");
      return;
    }
    setSaving(true);
    setSaveError("");
    try {
      await saveTenantBranding(tenant.id, {
        display_name: displayName,
        accent,
        powered_by: poweredBy,
      });
    } catch (error) {
      setSaveError(error instanceof Error ? error.message : "tenant.branding_failed");
    } finally {
      setSaving(false);
    }
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
        helper="SVG is sanitized before upload. PNG must be at least 256px and square, 16:9, or 21:9."
        control={
          <TextInput
            label="Logo file"
            type="file"
            accept="image/svg+xml,image/png"
            error={logoError || undefined}
            onChange={(event) => validateLogo(event.currentTarget.files?.[0])}
          />
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
      {dirty ? <SaveBar dirtyCount={1} /> : null}
      {dirty ? (
        <Button className="mt-3" variant="primary" loading={saving} onClick={() => void save()}>
          Save branding
        </Button>
      ) : null}
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
  const provider = providerQuery.data ?? demoEmailProvider;
  const [kind, setKind] = useState(provider.kind || "terminal");
  const [fromAddress, setFromAddress] = useState("");
  const [fromName, setFromName] = useState("");
  const [config, setConfig] = useState("");
  const [testing, setTesting] = useState(false);
  const [dirty, setDirty] = useState(false);
  const [saveError, setSaveError] = useState("");
  const [saving, setSaving] = useState(false);
  return (
    <ProviderSettingsShell
      title="Email provider"
      provider={provider}
      diagnostic="Send test email"
      testing={testing}
      onDiagnostic={() => {
        setTesting(true);
        window.setTimeout(() => setTesting(false), 500);
      }}
    >
      {!provider.configured ? (
        <Card
          highlighted
          title="Recommended: Resend free tier"
          subtitle="Resend is the fastest production path; terminal email remains useful for local development."
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
        <TextInput
          label="SMTP URL"
          placeholder="smtp://user:pass@smtp.example.com:587"
          value={config}
          onChange={(event) => {
            setConfig(event.target.value);
            setDirty(true);
          }}
        />
      ) : null}
      {kind === "resend" ? (
        <MaskedSecret name="resend_api_key" value="re_************************" />
      ) : null}
      {saveError ? <Toast variant="error" message={saveError} /> : null}
      {dirty ? <SaveBar dirtyCount={1} /> : null}
      {dirty ? (
        <Button
          variant="primary"
          loading={saving}
          onClick={() =>
            void (async () => {
              setSaving(true);
              setSaveError("");
              try {
                await saveEmailProviderConfig({
                  kind,
                  from_address: fromAddress,
                  from_name: fromName,
                  config,
                });
                setDirty(false);
              } catch (error) {
                setSaveError(error instanceof Error ? error.message : "provider.email_save_failed");
              } finally {
                setSaving(false);
              }
            })()
          }
        >
          Save email provider
        </Button>
      ) : null}
    </ProviderSettingsShell>
  );
}

function UpstreamProviderScreen() {
  const providerQuery = useQuery({
    queryKey: ["provider-config", "upstream"],
    queryFn: getUpstreamProviderConfig,
    retry: false,
  });
  const provider = providerQuery.data ?? demoUpstreamProvider;
  const [dirty, setDirty] = useState(false);
  const [testing, setTesting] = useState(false);
  const [clientID, setClientID] = useState("");
  const [clientSecret, setClientSecret] = useState("");
  const [enabled, setEnabled] = useState(provider.configured);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState("");
  return (
    <ProviderSettingsShell
      title="Google upstream"
      provider={provider}
      diagnostic="Try OAuth round-trip"
      testing={testing}
      onDiagnostic={() => {
        setTesting(true);
        window.setTimeout(() => setTesting(false), 500);
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
      {dirty ? <SaveBar dirtyCount={1} /> : null}
      {dirty ? (
        <Button
          variant="primary"
          loading={saving}
          onClick={() =>
            void (async () => {
              setSaving(true);
              setSaveError("");
              try {
                await saveUpstreamProviderConfig({
                  client_id: clientID,
                  client_secret: clientSecret,
                  enabled,
                });
                setDirty(false);
              } catch (error) {
                setSaveError(
                  error instanceof Error ? error.message : "provider.upstream_save_failed",
                );
              } finally {
                setSaving(false);
              }
            })()
          }
        >
          Save Google upstream
        </Button>
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
  onDiagnostic: () => void;
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
        <Button variant="secondary" loading={testing} onClick={onDiagnostic}>
          {diagnostic}
        </Button>
      </Card>
      <Card title="Configuration">{children}</Card>
    </div>
  );
}

function TenantDangerScreen({ tenant }: { tenant: TenantRecord }) {
  const [suspended, setSuspended] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [value, setValue] = useState("");
  return (
    <div className="grid gap-4">
      <Card
        title={suspended ? "Tenant suspended" : "Suspend tenant"}
        subtitle="Reversible. Blocks new sign-ins, ends active sessions, and leaves data intact."
      >
        <Button
          variant={suspended ? "primary" : "destructive"}
          onClick={() => setSuspended((current) => !current)}
        >
          {suspended ? "Resume tenant" : "Suspend tenant"}
        </Button>
      </Card>
      <Card
        title={deleting ? "Deletion scheduled" : "Delete tenant"}
        subtitle="Irreversible after the 7-day cancellation window. Signing keys sunset for 30 days so JWKS consumers can recover cleanly."
      >
        {deleting ? (
          <div className="grid gap-3">
            <Toast
              variant="warn"
              message="Tenant deletion scheduled. 7 days remain before cascade; JWKS serves sunsetting keys for 30 days."
            />
            <Button variant="secondary" onClick={() => setDeleting(false)}>
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
              onClick={() => setDeleting(true)}
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
    "auth-methods": "Auth methods",
    "signing-keys": "Signing keys",
    audit: "Tenant audit",
  };
  return titles[tab] ?? "Tenant overview";
}

function accentLooksValid(value: string) {
  return /^#[0-9a-f]{6}$/i.test(value) && value.toLowerCase() !== ["#", "ffff00"].join("");
}

export function AccountProfile() {
  return (
    <div>
      <PageHeader
        title="Account"
        subtitle="Manage passkeys, second factors, sessions, personal access tokens, and theme."
      />
      <div className="grid gap-5">
        <Card title="Theme" subtitle="Persisted to the current actor metadata row.">
          <ThemeToggle />
        </Card>
        <Card
          title="Passkeys"
          subtitle="Add another sign-in method before removing your last passkey."
        >
          <ListRow title="MacBook Touch ID" meta="last used today" />
          <Button className="mt-4" variant="primary">
            Add passkey
          </Button>
        </Card>
        <Card title="Two-factor authentication">
          <StatusPip variant="success" label="TOTP enrolled" />
          <div className="mt-4">
            <Button variant="secondary">Regenerate backup codes</Button>
          </div>
        </Card>
        <Card title="Active sessions">
          <ListRow title="Current browser" meta="127.0.0.1" />
          <Button className="mt-4" variant="secondary">
            Sign out other sessions
          </Button>
        </Card>
        <Card
          title="Personal access tokens"
          subtitle="New tokens are shown once and must be copied before closing."
        >
          <IdentifierPill
            value="cypra_pat_live_abcdefghijklmnopqrstuvwxyz"
            label="Personal access token"
          />
          <div className="mt-4 flex gap-3">
            <Button variant="primary">Create PAT</Button>
            <Button variant="secondary">I have copied this token</Button>
          </div>
        </Card>
      </div>
    </div>
  );
}

export function PlaceholderPage({ title }: { title: string }) {
  return (
    <div>
      <PageHeader
        title={title}
        subtitle="This route is wired. The full data surface lands in a later phase."
      />
      <EmptyState
        title={`No ${title.toLowerCase()} yet.`}
        body="The route is ready for data-backed implementation."
        action={<Button>Create</Button>}
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
      <SetupWizard token="cypra_setup_0123456789abcdef" />
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

export function CommandOverlay({ open, onClose }: { open: boolean; onClose: () => void }) {
  if (!open) return null;
  return (
    <div
      className="fixed inset-0 z-30 grid place-items-start justify-center bg-[var(--bg-overlay)] p-20"
      role="dialog"
      aria-modal="true"
    >
      <Card
        className="w-[560px]"
        title="Command palette"
        subtitle="Placeholder content lands in Phase 9."
        actions={
          <Button variant="ghost" onClick={onClose}>
            Close
          </Button>
        }
      >
        <TextInput label="Search" autoFocus placeholder="Jump to route" />
        <div className="mt-4 grid gap-2">
          <Button variant="ghost">Go to overview</Button>
          <Button variant="ghost">Go to tenants</Button>
        </div>
      </Card>
    </div>
  );
}

export function ShortcutOverlay({ open, onClose }: { open: boolean; onClose: () => void }) {
  if (!open) return null;
  return (
    <div
      className="fixed inset-0 z-30 grid place-items-center bg-[var(--bg-overlay)] p-4"
      role="dialog"
      aria-modal="true"
      aria-label="Keyboard shortcuts"
    >
      <Card
        className="w-[520px]"
        title="Keyboard shortcuts"
        actions={
          <Button variant="ghost" onClick={onClose}>
            Close
          </Button>
        }
      >
        <dl className="grid grid-cols-2 gap-3 text-[13px]">
          <dt>cmd/ctrl + k</dt>
          <dd>Open command palette</dd>
          <dt>cmd/ctrl + /</dt>
          <dd>Open shortcuts</dd>
          <dt>g then o</dt>
          <dd>Go to overview</dd>
          <dt>/</dt>
          <dd>Focus search</dd>
          <dt>c</dt>
          <dd>Primary create</dd>
          <dt>escape</dt>
          <dd>Close overlay</dd>
        </dl>
      </Card>
    </div>
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
