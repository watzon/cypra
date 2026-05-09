import { useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { AuthError, getMe, getVersion } from "@/api";
import { MobileBlockedBanner, PrimitiveGallery, SidebarNav } from "@/components";
import { CreateTenantModal } from "@/modals";
import {
  AccountProfile,
  AccountStates,
  CommandOverlay,
  DashboardOverview,
  ErrorPage,
  FooterHelp,
  InstanceAdminsScreen,
  InstanceDiagnosticsScreen,
  OverviewStates,
  AuditLogScreen,
  SetupStateGallery,
  SetupWizard,
  ShortcutOverlay,
  TenantDetail,
  TenantList,
  VersionToast,
} from "@/screens";

type SetupStep = "admin" | "passkey" | "backup";

type Route =
  | { kind: "setup"; token: string; step: SetupStep }
  | { kind: "gallery" }
  | { kind: "dashboard" }
  | { kind: "account" }
  | { kind: "tenant-list" }
  | {
      kind: "tenant-detail";
      slug: string;
      tab: string;
      settingsTab?: string;
      authProviderTab?: string;
      detailSlug?: string;
    }
  | { kind: "instance-admins" }
  | { kind: "instance-diagnostics" }
  | { kind: "instance-audit" }
  | { kind: "error"; code: "403" | "404" | "500" | "503" }
  | { kind: "placeholder"; title: string };

const galleryPlaceholders: Record<string, string> = {
  "/__cypra/gallery/setup": "Setup state gallery",
  "/__cypra/gallery/account": "Account state gallery",
  "/__cypra/gallery/overview": "Overview state gallery",
};

function parseRoute(pathname: string): Route {
  if (pathname.startsWith("/setup/")) {
    const segments = pathname.split("/").filter(Boolean);
    // /setup/<token>            -> step "admin"
    // /setup/<token>/<step>     -> explicit step
    const token = segments[1] ?? "";
    const stepSegment = segments[2] ?? "admin";
    const step: SetupStep =
      stepSegment === "admin" || stepSegment === "passkey" || stepSegment === "backup"
        ? stepSegment
        : "admin";
    return { kind: "setup", token, step };
  }
  if (pathname === "/__cypra/gallery") return { kind: "gallery" };
  if (galleryPlaceholders[pathname])
    return { kind: "placeholder", title: galleryPlaceholders[pathname] };
  if (
    pathname === "/__cypra/403" ||
    pathname === "/__cypra/404" ||
    pathname === "/__cypra/500" ||
    pathname === "/__cypra/503"
  )
    return { kind: "error", code: pathname.slice(-3) as "403" | "404" | "500" | "503" };
  if (pathname === "/dashboard" || pathname === "/") return { kind: "dashboard" };
  if (pathname === "/dashboard/account") return { kind: "account" };
  if (pathname === "/dashboard/tenants") return { kind: "tenant-list" };
  if (pathname === "/dashboard/instance/admins") return { kind: "instance-admins" };
  if (pathname === "/dashboard/instance/diagnostics") return { kind: "instance-diagnostics" };
  if (pathname === "/dashboard/instance/audit") return { kind: "instance-audit" };
  if (pathname.startsWith("/dashboard/tenants/")) return parseTenantRoute(pathname);
  return { kind: "error", code: "404" };
}

function parseTenantRoute(pathname: string): Route {
  const segments = pathname.split("/").filter(Boolean);
  // segments: ["dashboard", "tenants", <slug>, <tab>, <sub...>]
  const slug = segments[2];
  const maybeTab = segments[3];
  const maybeSubTab = segments[4];
  const remainder = segments.slice(5).join("/");
  if (!slug) return { kind: "error", code: "404" };
  if (maybeTab === "settings") {
    return { kind: "tenant-detail", slug, tab: "settings", settingsTab: maybeSubTab || "branding" };
  }
  if (maybeTab === "auth-providers" || maybeTab === "auth-methods") {
    const sub = maybeSubTab || "overview";
    const authProviderTab = remainder ? `${sub}/${remainder}` : sub;
    return {
      kind: "tenant-detail",
      slug,
      tab: "auth-providers",
      authProviderTab,
    };
  }
  return { kind: "tenant-detail", slug, tab: maybeTab || "overview", detailSlug: maybeSubTab };
}

function routeScope(route: Route): { scope: "instance" | "tenant"; tenantSlug?: string } {
  if (route.kind === "tenant-detail") return { scope: "tenant", tenantSlug: route.slug };
  return { scope: "instance" };
}

function requiresAuth(route: Route): boolean {
  switch (route.kind) {
    case "dashboard":
    case "account":
    case "tenant-list":
    case "tenant-detail":
    case "instance-admins":
    case "instance-diagnostics":
    case "instance-audit":
      return true;
    default:
      return false;
  }
}

export function App() {
  const [route, setRoute] = useState<Route>(() => parseRoute(window.location.pathname));
  const [commandOpen, setCommandOpen] = useState(false);
  const [shortcutsOpen, setShortcutsOpen] = useState(false);
  const [mobileNavOpen, setMobileNavOpen] = useState(false);
  const [createTenantOpen, setCreateTenantOpen] = useState(false);
  const [bootVersion, setBootVersion] = useState<string>();
  const versionQuery = useQuery({
    queryKey: ["version"],
    queryFn: getVersion,
    refetchInterval: 60_000,
    retry: false,
  });

  const authGate = useQuery({
    queryKey: ["app-me"],
    queryFn: getMe,
    retry: false,
    enabled: requiresAuth(route),
  });
  useEffect(() => {
    if (authGate.error instanceof AuthError && authGate.error.status === 401) {
      window.location.assign("/login");
    }
  }, [authGate.error]);

  useEffect(() => {
    if (!bootVersion && versionQuery.data?.version) {
      setBootVersion(versionQuery.data.version);
    }
  }, [bootVersion, versionQuery.data?.version]);

  useEffect(() => {
    const onPop = () => setRoute(parseRoute(window.location.pathname));
    window.addEventListener("popstate", onPop);
    return () => window.removeEventListener("popstate", onPop);
  }, []);

  const seqRef = useRef<{ key: string; at: number } | null>(null);
  useEffect(() => {
    const isTypingTarget = (target: EventTarget | null) => {
      if (!(target instanceof HTMLElement)) return false;
      const tag = target.tagName;
      return tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT" || target.isContentEditable;
    };
    const tenantSlug = route.kind === "tenant-detail" ? route.slug : null;
    const tenantGotoMap: Record<string, string> = tenantSlug
      ? {
          o: `/dashboard/tenants/${tenantSlug}`,
          p: `/dashboard/tenants/${tenantSlug}/projects`,
          u: `/dashboard/tenants/${tenantSlug}/users`,
          t: `/dashboard/tenants/${tenantSlug}/auth-providers`,
          k: `/dashboard/tenants/${tenantSlug}/signing-keys`,
          a: `/dashboard/tenants/${tenantSlug}/audit`,
          s: `/dashboard/tenants/${tenantSlug}/settings/branding`,
          n: "/dashboard/account",
        }
      : {};
    const instanceGotoMap: Record<string, string> = {
      o: "/dashboard",
      t: "/dashboard/tenants",
      a: "/dashboard/instance/audit",
      d: "/dashboard/instance/diagnostics",
      i: "/dashboard/instance/admins",
      n: "/dashboard/account",
    };
    const gotoMap = tenantSlug ? tenantGotoMap : instanceGotoMap;
    const handler = (event: KeyboardEvent) => {
      const key = event.key.toLowerCase();
      if ((event.metaKey || event.ctrlKey) && key === "k") {
        event.preventDefault();
        setCommandOpen(true);
        return;
      }
      if ((event.metaKey || event.ctrlKey) && event.key === "/") {
        event.preventDefault();
        setShortcutsOpen(true);
        return;
      }
      if (event.shiftKey && event.key === "?") {
        event.preventDefault();
        setShortcutsOpen(true);
        return;
      }
      if (event.key === "Escape") {
        setCommandOpen(false);
        setShortcutsOpen(false);
        return;
      }
      if (isTypingTarget(event.target) || event.metaKey || event.ctrlKey || event.altKey) {
        return;
      }
      if (seqRef.current && Date.now() - seqRef.current.at < 1500) {
        const target = gotoMap[key];
        seqRef.current = null;
        if (target) {
          event.preventDefault();
          window.location.assign(target);
          return;
        }
      }
      if (key === "g") {
        seqRef.current = { key: "g", at: Date.now() };
        return;
      }
      seqRef.current = null;
      if (key === "/") {
        event.preventDefault();
        document.querySelector<HTMLInputElement>("input[type='search'], input")?.focus();
        return;
      }
      if (key === "c") {
        document.querySelector<HTMLButtonElement>("button[data-primary-create='true']")?.focus();
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [route]);

  if (route.kind === "setup") return <SetupWizard token={route.token} step={route.step} />;

  // While we resolve the auth gate (or during the redirect to /login), render
  // a minimal shell so we don't flash the dashboard chrome at a signed-out
  // user. The redirect itself happens in the useEffect above.
  if (requiresAuth(route)) {
    if (authGate.isLoading) {
      return <div className="min-h-screen bg-bg-canvas" aria-hidden="true" />;
    }
    if (authGate.error instanceof AuthError && authGate.error.status === 401) {
      return <div className="min-h-screen bg-bg-canvas" aria-hidden="true" />;
    }
  }

  const { scope, tenantSlug } = routeScope(route);

  return (
    <div className="min-h-screen bg-bg-canvas text-text-primary">
      <div className="flex">
        <SidebarNav
          className="hidden md:flex lg:hidden"
          collapsed
          scope={scope}
          tenantSlug={tenantSlug}
          onCreateTenant={() => setCreateTenantOpen(true)}
          label="Compact dashboard navigation"
        />
        <SidebarNav
          className="hidden lg:flex"
          scope={scope}
          tenantSlug={tenantSlug}
          onCreateTenant={() => setCreateTenantOpen(true)}
          label="Expanded dashboard navigation"
        />
        <main className="min-h-screen flex-1 p-4 md:p-6">
          <button
            type="button"
            className="mb-3 inline-flex h-10 items-center justify-center rounded-[var(--radius-md)] border border-border-default bg-bg-surface px-3 text-[13px] md:hidden"
            aria-expanded={mobileNavOpen}
            aria-controls="mobile-dashboard-nav"
            onClick={() => setMobileNavOpen(true)}
          >
            Open navigation
          </button>
          <MobileBlockedBanner />
          <div className="mx-auto mt-4 max-w-[1280px]">{renderRoute(route)}</div>
        </main>
      </div>
      {mobileNavOpen ? (
        <div
          className="fixed inset-0 z-40 bg-[var(--bg-overlay)] md:hidden"
          role="dialog"
          aria-modal="true"
          aria-label="Dashboard navigation"
        >
          <div
            id="mobile-dashboard-nav"
            className="flex h-full max-w-[280px] flex-col bg-bg-surface shadow-[var(--shadow-lg)]"
          >
            <button
              type="button"
              className="m-3 h-10 rounded-[var(--radius-md)] border border-border-default text-[13px] text-text-secondary"
              onClick={() => setMobileNavOpen(false)}
            >
              Close navigation
            </button>
            <SidebarNav
              scope={scope}
              tenantSlug={tenantSlug}
              onCreateTenant={() => setCreateTenantOpen(true)}
              label="Mobile dashboard navigation"
            />
          </div>
        </div>
      ) : null}
      <CommandOverlay
        open={commandOpen}
        onClose={() => setCommandOpen(false)}
        scope={scope}
        tenantSlug={tenantSlug}
      />
      <CreateTenantModal
        open={createTenantOpen}
        onClose={() => setCreateTenantOpen(false)}
        onSuccess={(tenant) => window.location.assign(`/dashboard/tenants/${tenant.slug}`)}
      />
      <ShortcutOverlay open={shortcutsOpen} onClose={() => setShortcutsOpen(false)} />
      <FooterHelp onHelp={() => setShortcutsOpen(true)} />
      <VersionToast bootVersion={bootVersion} currentVersion={versionQuery.data?.version} />
      <div className="sr-only" role="status" aria-live="polite">
        {routeTitle(route)}
      </div>
    </div>
  );
}

function routeTitle(route: Route) {
  if (route.kind === "setup") return "Set up Cypra";
  if (route.kind === "gallery") return "Primitive gallery";
  if (route.kind === "dashboard") return "Overview";
  if (route.kind === "account") return "Account";
  if (route.kind === "tenant-list") return "Tenants";
  if (route.kind === "tenant-detail") return route.slug;
  if (route.kind === "instance-admins") return "Instance admins";
  if (route.kind === "instance-diagnostics") return "Diagnostics";
  if (route.kind === "instance-audit") return "Instance audit";
  if (route.kind === "placeholder") return route.title;
  return `Error ${route.code}`;
}

function renderRoute(route: Route) {
  if (route.kind === "gallery") return <PrimitiveGallery />;
  if (route.kind === "dashboard") return <DashboardOverview />;
  if (route.kind === "account") return <AccountProfile />;
  if (route.kind === "tenant-list") return <TenantList />;
  if (route.kind === "instance-admins") return <InstanceAdminsScreen />;
  if (route.kind === "instance-diagnostics") return <InstanceDiagnosticsScreen />;
  if (route.kind === "instance-audit") return <AuditLogScreen scope="instance" />;
  if (route.kind === "tenant-detail")
    return (
      <TenantDetail
        slug={route.slug}
        tab={route.tab}
        settingsTab={route.settingsTab}
        authProviderTab={route.authProviderTab}
        detailSlug={route.detailSlug}
      />
    );
  if (route.kind === "error") return <ErrorPage code={route.code} />;
  if (route.kind === "placeholder" && route.title === "Setup state gallery")
    return <SetupStateGallery />;
  if (route.kind === "placeholder" && route.title === "Account state gallery")
    return <AccountStates />;
  if (route.kind === "placeholder" && route.title === "Overview state gallery")
    return <OverviewStates />;
  return <ErrorPage code="404" />;
}
