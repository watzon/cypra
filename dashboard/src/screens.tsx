import { useEffect, useState, type ReactNode } from "react";

import { getReady } from "@/api";
import {
  AuditEntry,
  BackupCodeGrid,
  Button,
  Card,
  ContextBadge,
  EmptyState,
  ErrorState,
  IdentifierPill,
  KeyRotationTimeline,
  ListRow,
  LoadingState,
  MaskedSecret,
  PageHeader,
  SetupTokenBanner,
  StatusPip,
  TextInput,
  ThemeToggle,
  Toast,
} from "@/components";
import { Icons, PasskeyGlyph } from "@/icons";

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
        <EmptyState
          title="Instance admin created."
          body="Your install is ready. Continue to the dashboard."
          action={
            <Button variant="primary" onClick={() => history.pushState(null, "", "/dashboard")}>
              Open dashboard
            </Button>
          }
        />
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
