import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { getVersion } from "@/api";
import { MobileBlockedBanner, PrimitiveGallery, SidebarNav } from "@/components";
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
  PlaceholderPage,
  SetupStateGallery,
  SetupWizard,
  ShortcutOverlay,
  TenantDetail,
  TenantList,
  VersionToast,
} from "@/screens";

type Route =
  | { kind: "setup"; token: string }
  | { kind: "gallery" }
  | { kind: "dashboard" }
  | { kind: "account" }
  | { kind: "tenant-list" }
  | { kind: "tenant-detail"; slug: string; tab: string; settingsTab?: string; detailSlug?: string }
  | { kind: "error"; code: "403" | "404" | "500" | "503" }
  | { kind: "placeholder"; title: string };

const routeTitles: Record<string, string> = {
  "/dashboard/tenants": "Tenants",
  "/dashboard/users": "Users",
  "/dashboard/projects": "Projects",
  "/dashboard/auth-methods": "Auth methods",
  "/dashboard/members": "Members",
  "/dashboard/audit": "Audit",
  "/dashboard/settings": "Settings",
  "/dashboard/instance/admins": "Instance admins",
  "/dashboard/instance/storage": "Instance storage",
  "/dashboard/instance/audit": "Instance audit",
  "/dashboard/instance/diagnostics": "Diagnostics",
};

function parseRoute(pathname: string): Route {
  if (pathname.startsWith("/setup/"))
    return { kind: "setup", token: pathname.split("/").at(-1) ?? "" };
  if (pathname === "/__cypra/gallery") return { kind: "gallery" };
  if (pathname === "/__cypra/gallery/setup")
    return { kind: "placeholder", title: "Setup state gallery" };
  if (pathname === "/__cypra/gallery/account")
    return { kind: "placeholder", title: "Account state gallery" };
  if (pathname === "/__cypra/gallery/overview")
    return { kind: "placeholder", title: "Overview state gallery" };
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
  if (pathname.startsWith("/dashboard/tenants/")) return parseTenantRoute(pathname);
  if (routeTitles[pathname]) return { kind: "placeholder", title: routeTitles[pathname] };
  return { kind: "error", code: "404" };
}

function parseTenantRoute(pathname: string): Route {
  const [, , , slug, maybeTab, maybeSubTab] = pathname.split("/");
  if (!slug) return { kind: "error", code: "404" };
  if (maybeTab === "settings") {
    return { kind: "tenant-detail", slug, tab: "settings", settingsTab: maybeSubTab || "branding" };
  }
  return { kind: "tenant-detail", slug, tab: maybeTab || "overview", detailSlug: maybeSubTab };
}

export function App() {
  const [route, setRoute] = useState<Route>(() => parseRoute(window.location.pathname));
  const [commandOpen, setCommandOpen] = useState(false);
  const [shortcutsOpen, setShortcutsOpen] = useState(false);
  const bootVersion = useMemo(() => "dev", []);
  const versionQuery = useQuery({
    queryKey: ["version"],
    queryFn: getVersion,
    refetchInterval: 60_000,
    retry: false,
  });

  useEffect(() => {
    const onPop = () => setRoute(parseRoute(window.location.pathname));
    window.addEventListener("popstate", onPop);
    return () => window.removeEventListener("popstate", onPop);
  }, []);

  useEffect(() => {
    const handler = (event: KeyboardEvent) => {
      const key = event.key.toLowerCase();
      if ((event.metaKey || event.ctrlKey) && key === "k") {
        event.preventDefault();
        setCommandOpen(true);
      }
      if ((event.metaKey || event.ctrlKey) && event.key === "/") {
        event.preventDefault();
        setShortcutsOpen(true);
      }
      if (event.key === "Escape") {
        setCommandOpen(false);
        setShortcutsOpen(false);
      }
      if (key === "/" && !event.metaKey && !event.ctrlKey) {
        document.querySelector<HTMLInputElement>("input[type='search'], input")?.focus();
      }
      if (key === "c") {
        document.querySelector<HTMLButtonElement>("button[data-primary-create='true']")?.focus();
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, []);

  if (route.kind === "setup") return <SetupWizard token={route.token} />;

  return (
    <div className="min-h-screen bg-bg-canvas text-text-primary">
      <div className="flex">
        <SidebarNav />
        <main className="min-h-screen flex-1 p-4 md:p-6">
          <MobileBlockedBanner />
          <div className="mx-auto mt-4 max-w-[1280px]">{renderRoute(route)}</div>
        </main>
      </div>
      <CommandOverlay open={commandOpen} onClose={() => setCommandOpen(false)} />
      <ShortcutOverlay open={shortcutsOpen} onClose={() => setShortcutsOpen(false)} />
      <FooterHelp onHelp={() => setShortcutsOpen(true)} />
      <VersionToast bootVersion={bootVersion} currentVersion={versionQuery.data?.version} />
      <div className="sr-only" role="status" aria-live="polite">
        {route.kind}
      </div>
    </div>
  );
}

function renderRoute(route: Route) {
  if (route.kind === "gallery") return <PrimitiveGallery />;
  if (route.kind === "dashboard") return <DashboardOverview />;
  if (route.kind === "account") return <AccountProfile />;
  if (route.kind === "tenant-list") return <TenantList />;
  if (route.kind === "tenant-detail")
    return (
      <TenantDetail
        slug={route.slug}
        tab={route.tab}
        settingsTab={route.settingsTab}
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
  if (route.kind === "placeholder" && route.title === "Instance audit")
    return <AuditLogScreen scope="instance" />;
  if (route.kind === "placeholder" && route.title === "Instance admins")
    return <InstanceAdminsScreen />;
  if (route.kind === "placeholder" && route.title === "Diagnostics")
    return <InstanceDiagnosticsScreen />;
  if (route.kind === "placeholder") return <PlaceholderPage title={route.title} />;
  return <ErrorPage code="404" />;
}
