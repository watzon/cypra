import {
  useEffect,
  useId,
  useMemo,
  useState,
  type ButtonHTMLAttributes,
  type InputHTMLAttributes,
  type PropsWithChildren,
  type ReactNode,
  type SelectHTMLAttributes,
} from "react";

import { Icons, InstanceGlyph, OidcGlyph, PasskeyGlyph, TenantGlyph } from "@/icons";
import { cn, middleEllipsis } from "@/lib/utils";
import { useTheme, type ThemePreference } from "@/theme";

export type StatusVariant =
  | "active"
  | "overlap"
  | "sunsetting"
  | "revoked"
  | "pending"
  | "success"
  | "warn"
  | "error"
  | "info";

const statusClass: Record<StatusVariant, string> = {
  active: "text-key-state-active",
  overlap: "text-key-state-overlap",
  sunsetting: "text-key-state-sunsetting",
  revoked: "text-text-tertiary",
  pending: "text-status-pending",
  success: "text-status-success",
  warn: "text-status-warn",
  error: "text-status-error",
  info: "text-status-info",
};

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: "primary" | "secondary" | "ghost" | "destructive";
  size?: "sm" | "md" | "lg";
  loading?: boolean;
  kbd?: string;
  leading?: ReactNode;
  trailing?: ReactNode;
};

export function Button({
  className,
  children,
  variant = "secondary",
  size = "md",
  loading = false,
  kbd,
  leading,
  trailing,
  disabled,
  ...props
}: ButtonProps) {
  return (
    <button
      className={cn(
        "inline-flex items-center justify-center gap-2 rounded-[var(--radius-md)] border font-medium transition duration-[var(--dur-instant)] active:scale-[0.985] disabled:cursor-not-allowed disabled:opacity-100 disabled:saturate-50",
        size === "sm" && "h-7 px-3 text-[13px]",
        size === "md" && "h-9 px-4 text-[14px]",
        size === "lg" && "h-11 px-5 text-[14px]",
        variant === "primary" &&
          "border-accent-primary bg-accent-primary text-text-on-accent hover:bg-accent-primary-hi active:bg-accent-primary-lo",
        variant === "secondary" &&
          "border-border-default bg-bg-surface text-text-primary hover:bg-bg-elevated",
        variant === "ghost" &&
          "border-transparent bg-transparent text-text-secondary hover:bg-bg-elevated hover:text-text-primary",
        variant === "destructive" &&
          "border-status-error bg-status-error text-text-on-accent hover:opacity-90",
        className,
      )}
      disabled={Boolean(disabled) || loading}
      aria-busy={loading ? true : undefined}
      {...props}
    >
      {loading ? (
        <span
          className="h-3 w-3 rounded-full border border-current border-t-transparent"
          aria-hidden="true"
        />
      ) : (
        leading
      )}
      <span className={cn(loading && "sr-only")}>{children}</span>
      {kbd ? (
        <kbd className="ml-1 rounded bg-bg-code px-1.5 py-0.5 text-[12px] text-text-primary">
          {kbd}
        </kbd>
      ) : null}
      {trailing}
    </button>
  );
}

type IconButtonProps = Omit<ButtonProps, "children" | "leading" | "trailing" | "kbd"> & {
  label: string;
  icon: ReactNode;
};

export function IconButton({
  label,
  icon,
  className,
  size = "md",
  variant = "ghost",
  ...props
}: IconButtonProps) {
  return (
    <Button
      className={cn(
        "px-0",
        size === "sm" && "w-7",
        size === "md" && "w-8",
        size === "lg" && "w-10",
        className,
      )}
      size={size}
      variant={variant}
      title={label}
      aria-label={label}
      {...props}
    >
      {icon}
    </Button>
  );
}

type TextInputProps = InputHTMLAttributes<HTMLInputElement> & {
  label: string;
  helper?: string;
  error?: string;
  state?: "default" | "validating" | "success" | "readonly";
};

export function TextInput({
  label,
  helper,
  error,
  state = "default",
  className,
  id,
  ...props
}: TextInputProps) {
  const generatedId = useId();
  const inputId = id ?? generatedId;
  const helperId = `${inputId}-helper`;
  const describedBy = error !== undefined || helper !== undefined ? helperId : undefined;
  return (
    <label className="grid gap-1.5 text-[13px] font-medium text-text-primary" htmlFor={inputId}>
      <span>{label}</span>
      <span className="relative">
        <input
          id={inputId}
          aria-label={label}
          className={cn(
            "h-10 w-full rounded-[var(--radius-md)] border border-border-default bg-bg-surface px-3 text-[14px] text-text-primary transition duration-[var(--dur-instant)] placeholder:text-text-tertiary read-only:bg-bg-code",
            error && "border-status-error",
            state === "success" && "border-status-success",
            className,
          )}
          aria-invalid={error !== undefined ? true : undefined}
          aria-describedby={describedBy}
          aria-busy={state === "validating" ? true : undefined}
          readOnly={state === "readonly" || Boolean(props.readOnly)}
          {...props}
        />
        {state === "validating" ? (
          <Icons.Clock
            className="absolute right-3 top-3 h-4 w-4 text-status-pending"
            aria-hidden="true"
          />
        ) : null}
      </span>
      {error !== undefined || helper !== undefined ? (
        <span
          id={helperId}
          className={cn(
            "text-[12px] font-normal",
            error ? "text-status-error" : "text-text-secondary",
          )}
        >
          {error ?? helper}
        </span>
      ) : null}
    </label>
  );
}

type SelectProps = SelectHTMLAttributes<HTMLSelectElement> & {
  label: string;
  helper?: string;
  options: { value: string; label: string }[];
};

export function Select({ label, helper, options, id, className, ...props }: SelectProps) {
  const generatedId = useId();
  const selectId = id ?? generatedId;
  return (
    <label className="grid gap-1.5 text-[13px] font-medium text-text-primary" htmlFor={selectId}>
      <span>{label}</span>
      <select
        id={selectId}
        className={cn(
          "h-10 rounded-[var(--radius-md)] border border-border-default bg-bg-surface px-3 text-[14px]",
          className,
        )}
        {...props}
      >
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
      {helper ? (
        <span className="text-[12px] font-normal text-text-secondary">{helper}</span>
      ) : null}
    </label>
  );
}

export function Checkbox({
  label,
  helper,
  ...props
}: InputHTMLAttributes<HTMLInputElement> & { label: string; helper?: string }) {
  return (
    <label className="flex items-start gap-3 text-[14px] text-text-primary">
      <input type="checkbox" className="mt-0.5 h-5 w-5 accent-[var(--accent-primary)]" {...props} />
      <span className="grid gap-0.5">
        <span>{label}</span>
        {helper ? <span className="text-[12px] text-text-secondary">{helper}</span> : null}
      </span>
    </label>
  );
}

export function Radio({
  label,
  ...props
}: InputHTMLAttributes<HTMLInputElement> & { label: string }) {
  return (
    <label className="flex items-center gap-3 text-[14px]">
      <input type="radio" className="h-5 w-5 accent-[var(--accent-primary)]" {...props} />
      {label}
    </label>
  );
}

export function Switch({
  label,
  checked = false,
  pending = false,
  onChange,
}: {
  label: string;
  checked?: boolean;
  pending?: boolean;
  onChange?: (checked: boolean) => void;
}) {
  return (
    <button
      className="inline-flex items-center gap-3 text-[14px]"
      type="button"
      role="switch"
      aria-checked={checked}
      aria-busy={pending ? true : undefined}
      onClick={() => !pending && onChange?.(!checked)}
    >
      <span
        className={cn(
          "flex h-5 w-9 items-center rounded-[var(--radius-pill)] border border-border-default p-0.5",
          checked ? "bg-accent-primary" : "bg-bg-code",
        )}
      >
        <span
          className={cn(
            "grid h-4 w-4 place-items-center rounded-[var(--radius-pill)] bg-bg-surface transition duration-[var(--dur-fast)]",
            checked && "translate-x-4",
          )}
        >
          {pending ? <Icons.Clock className="h-3 w-3" /> : null}
        </span>
      </span>
      {label}
    </button>
  );
}

export function Tag({
  children,
  variant = "neutral",
}: PropsWithChildren<{
  variant?: "neutral" | "accent" | "context-instance" | "context-tenant" | StatusVariant;
}>) {
  const color =
    variant in statusClass
      ? statusClass[variant as StatusVariant]
      : variant === "accent"
        ? "text-accent-primary"
        : variant === "context-instance"
          ? "text-context-instance"
          : variant === "context-tenant"
            ? "text-context-tenant"
            : "text-text-secondary";
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-[var(--radius-pill)] border border-border-default bg-bg-code px-2 py-0.5 text-[11px] font-semibold uppercase tracking-[0.06em]",
        "text-text-primary",
      )}
    >
      <span className={color}>
        <StatusSigil variant={variant in statusClass ? (variant as StatusVariant) : "info"} />
      </span>
      {children}
    </span>
  );
}

export function Tooltip({ label, children }: PropsWithChildren<{ label: string }>) {
  return (
    <span className="group relative inline-flex">
      <span aria-describedby={label}>{children}</span>
      <span
        role="tooltip"
        className="pointer-events-none absolute bottom-full left-1/2 z-20 mb-2 hidden -translate-x-1/2 whitespace-nowrap rounded-[var(--radius-sm)] border border-border-emphasis bg-bg-elevated px-2 py-1 text-[13px] text-text-primary shadow-[var(--shadow-md)] group-hover:block group-focus-within:block"
      >
        {label}
      </span>
    </span>
  );
}

export function Toast({
  variant,
  message,
  action,
}: {
  variant: StatusVariant;
  message: string;
  action?: ReactNode;
}) {
  const live = variant === "error" || variant === "warn" ? "assertive" : "polite";
  return (
    <div
      role="status"
      aria-live={live}
      className="flex max-w-sm items-center gap-3 rounded-[var(--radius-md)] border border-border-emphasis bg-bg-elevated p-3 text-[13px] shadow-[var(--shadow-lg)]"
    >
      <StatusSigil variant={variant} />
      <span className="flex-1">{message}</span>
      {action}
    </div>
  );
}

export function Modal({
  title,
  open,
  children,
  footer,
  onClose,
}: PropsWithChildren<{ title: string; open: boolean; footer?: ReactNode; onClose: () => void }>) {
  if (!open) {
    return null;
  }
  return (
    <div
      className="fixed inset-0 z-40 grid place-items-center bg-[var(--bg-overlay)] p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="modal-title"
    >
      <section className="w-full max-w-xl rounded-[var(--radius-lg)] border border-border-emphasis bg-bg-elevated shadow-[var(--shadow-lg)]">
        <header className="flex items-center justify-between border-b border-border-subtle p-5">
          <h2 id="modal-title" className="text-[20px] font-semibold leading-7">
            {title}
          </h2>
          <IconButton
            label="Close"
            icon={<Icons.XCircle className="h-4 w-4" />}
            onClick={onClose}
          />
        </header>
        <div className="p-5">{children}</div>
        {footer ? (
          <footer className="flex justify-end gap-3 border-t border-border-subtle p-5">
            {footer}
          </footer>
        ) : null}
      </section>
    </div>
  );
}

export function Popover({ label, children }: PropsWithChildren<{ label: string }>) {
  return (
    <details className="relative inline-block">
      <summary className="cursor-pointer list-none rounded-[var(--radius-md)] border border-border-default bg-bg-surface px-3 py-2 text-[13px]">
        {label}
      </summary>
      <div className="absolute right-0 z-20 mt-2 min-w-52 rounded-[var(--radius-md)] border border-border-default bg-bg-elevated p-2 shadow-[var(--shadow-md)]">
        {children}
      </div>
    </details>
  );
}

export function Tabs({
  tabs,
}: {
  tabs: { id: string; label: string; content: ReactNode; disabled?: boolean }[];
}) {
  const [active, setActive] = useState(tabs[0]?.id ?? "");
  const current = tabs.find((tab) => tab.id === active) ?? tabs[0];
  return (
    <div className="grid gap-4">
      <div className="flex gap-2 overflow-x-auto border-b border-border-subtle" role="tablist">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            type="button"
            role="tab"
            aria-selected={tab.id === current.id}
            disabled={tab.disabled}
            className={cn(
              "border-b-2 px-3 py-2 text-[13px] text-text-secondary disabled:text-text-disabled",
              tab.id === current.id
                ? "border-accent-primary text-text-primary"
                : "border-transparent",
            )}
            onClick={() => setActive(tab.id)}
          >
            {tab.label}
          </button>
        ))}
      </div>
      <div role="tabpanel">{current.content}</div>
    </div>
  );
}

export function Card({
  title,
  subtitle,
  children,
  className,
  actions,
  highlighted = false,
}: PropsWithChildren<{
  title?: string;
  subtitle?: string;
  actions?: ReactNode;
  highlighted?: boolean;
  className?: string;
}>) {
  return (
    <section
      className={cn(
        "rounded-[var(--radius-md)] border border-border-subtle bg-bg-surface p-5 shadow-[var(--shadow-sm)]",
        highlighted && "bg-accent-primary-mu",
        className,
      )}
    >
      {title !== undefined || actions !== undefined ? (
        <header className="mb-4 flex items-start justify-between gap-4">
          <div>
            {title ? <h2 className="text-[20px] font-semibold leading-7">{title}</h2> : null}
            {subtitle ? (
              <p className="mt-1 text-[13px] leading-5 text-text-secondary">{subtitle}</p>
            ) : null}
          </div>
          {actions}
        </header>
      ) : null}
      {children}
    </section>
  );
}

export function Avatar({
  name,
  sub,
  size = "md",
}: {
  name?: string;
  sub: string;
  size?: "xs" | "sm" | "md" | "lg" | "xl";
}) {
  const sizes = {
    xs: "h-5 w-5 text-[10px]",
    sm: "h-6 w-6 text-[11px]",
    md: "h-8 w-8 text-[12px]",
    lg: "h-10 w-10 text-[13px]",
    xl: "h-16 w-16 text-[16px]",
  };
  const initials = name
    ?.split(" ")
    .map((part) => part[0])
    .join("")
    .slice(0, 2)
    .toUpperCase();
  const label = initials ?? sub.slice(-4);
  return (
    <span
      className={cn(
        "grid place-items-center rounded-[var(--radius-pill)] bg-bg-code font-mono text-text-secondary",
        sizes[size],
      )}
    >
      {label}
    </span>
  );
}

export function Skeleton({
  shape = "block",
  className,
}: {
  shape?: "line" | "circle" | "block";
  className?: string;
}) {
  return (
    <span
      className={cn(
        "block border border-border-subtle bg-bg-code",
        shape === "line" && "h-4 rounded-[var(--radius-sm)]",
        shape === "circle" && "h-8 w-8 rounded-[var(--radius-pill)]",
        shape === "block" && "h-24 rounded-[var(--radius-md)]",
        className,
      )}
    />
  );
}

export function SegmentedControl({
  options,
  value,
  onChange,
}: {
  options: string[];
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <div className="inline-flex rounded-[var(--radius-md)] border border-border-default bg-bg-code p-1">
      {options.map((option) => (
        <button
          key={option}
          type="button"
          className={cn(
            "rounded-[var(--radius-sm)] px-3 py-1.5 text-[13px] text-text-secondary",
            option === value && "bg-accent-primary-mu text-text-primary",
          )}
          onClick={() => onChange(option)}
        >
          {option}
        </button>
      ))}
    </div>
  );
}

export function IdentifierPill({
  value,
  label = "Identifier",
  disabled = false,
}: {
  value: string;
  label?: string;
  disabled?: boolean;
}) {
  const [copied, setCopied] = useState(false);
  const copy = () => {
    if (disabled) {
      return;
    }
    void navigator.clipboard.writeText(value).then(() => {
      setCopied(true);
      window.setTimeout(() => setCopied(false), 900);
    });
  };
  return (
    <span
      className={cn(
        "inline-flex items-center gap-2 rounded-[var(--radius-sm)] bg-bg-code px-2 py-1 font-mono text-[13px] text-text-identifier",
        copied && "bg-accent-primary-mu",
        disabled && "opacity-50",
      )}
      aria-label={`${label}: ${value}`}
    >
      <span>{middleEllipsis(value)}</span>
      <IconButton
        label={copied ? "Copied" : `Copy ${label}`}
        icon={<Icons.Copy className="h-3.5 w-3.5" />}
        size="sm"
        onClick={copy}
        disabled={disabled}
      />
    </span>
  );
}

export function MaskedSecret({ name, value }: { name: string; value: string }) {
  const [revealed, setRevealed] = useState(false);
  const [message, setMessage] = useState("");
  useEffect(() => {
    if (!revealed) {
      return;
    }
    const timer = window.setTimeout(() => setRevealed(false), 30_000);
    return () => window.clearTimeout(timer);
  }, [revealed]);
  const copy = () => {
    if (!revealed) {
      setMessage("Reveal first to copy");
      return;
    }
    void navigator.clipboard.writeText(value).then(() => setMessage(`Copied ${name}`));
  };
  return (
    <span className="inline-flex items-center gap-2 rounded-[var(--radius-sm)] bg-bg-code px-2 py-1 font-mono text-[13px] text-text-identifier">
      <span>
        {revealed ? (
          value
        ) : (
          <>
            <span>{name}_</span>
            <span className="text-secret-mask">**********</span>
          </>
        )}
      </span>
      <IconButton
        label={revealed ? "Hide secret" : "Reveal secret"}
        icon={
          revealed ? (
            <Icons.EyeOff className="h-3.5 w-3.5" />
          ) : (
            <Icons.Eye className="h-3.5 w-3.5" />
          )
        }
        size="sm"
        onClick={() => setRevealed((current) => !current)}
      />
      <IconButton
        label="Copy secret"
        icon={<Icons.Copy className="h-3.5 w-3.5" />}
        size="sm"
        onClick={copy}
      />
      <span className="sr-only" aria-live="polite">
        {message}
      </span>
    </span>
  );
}

export function TenantSwitcher() {
  return (
    <Popover label="Acme Operations">
      <div className="grid gap-2">
        <TextInput label="Search tenants" placeholder="acme" />
        <div className="rounded-[var(--radius-md)] px-2 py-2 text-left text-[13px] hover:bg-bg-code">
          <TenantGlyph className="mr-2 inline h-4 w-4 text-context-tenant" />
          Acme Operations <IdentifierPill value="acme" label="Tenant slug" />
        </div>
        <Button size="sm" variant="ghost">
          Create tenant
        </Button>
      </div>
    </Popover>
  );
}

export function ContextBadge({
  kind = "instance",
  label,
}: {
  kind?: "instance" | "tenant";
  label?: string;
}) {
  const Glyph = kind === "instance" ? InstanceGlyph : TenantGlyph;
  return (
    <span
      className={cn(
        "inline-flex items-center gap-2 rounded-[var(--radius-pill)] bg-bg-code px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.06em]",
        kind === "instance" ? "text-context-instance" : "text-context-tenant",
      )}
    >
      <Glyph className="h-4 w-4" />
      {label ?? (kind === "instance" ? "Instance admin" : "Tenant: acme")}
    </span>
  );
}

export function StatusPip({ variant, label }: { variant: StatusVariant; label: string }) {
  return (
    <span className="inline-flex items-center gap-2 text-[13px] text-text-primary">
      <span className={statusClass[variant]}>
        <StatusSigil variant={variant} />
      </span>
      {label}
    </span>
  );
}

function StatusSigil({ variant }: { variant: StatusVariant }) {
  if (variant === "pending") return <Icons.Clock className="h-3.5 w-3.5" aria-hidden="true" />;
  if (variant === "success")
    return <Icons.CheckCircle2 className="h-3.5 w-3.5" aria-hidden="true" />;
  if (variant === "warn" || variant === "overlap")
    return (
      <span
        className="h-0 w-0 border-x-[5px] border-b-[9px] border-x-transparent border-b-current"
        aria-hidden="true"
      />
    );
  if (variant === "sunsetting" || variant === "info")
    return (
      <span
        className="h-2 w-2 rounded-[var(--radius-pill)] border border-current"
        aria-hidden="true"
      />
    );
  if (variant === "revoked") return <span className="h-2 w-2 bg-current" aria-hidden="true" />;
  return <span className="h-2 w-2 rounded-[var(--radius-pill)] bg-current" aria-hidden="true" />;
}

export function KeyRotationTimeline() {
  return (
    <div className="grid gap-3">
      <div className="flex h-8 overflow-hidden rounded-[var(--radius-pill)] border border-border-subtle">
        <span className="w-1/2 bg-key-state-active" />
        <span className="w-1/3 bg-key-state-overlap" />
        <span className="flex-1 bg-key-state-sunsetting" />
      </div>
      <div className="flex justify-between text-[12px] text-text-secondary">
        <IdentifierPill value="kid_active_2026_05" label="Key ID" />
        <span>Next rotation in 28 days</span>
      </div>
    </div>
  );
}

export function AuditEntry({ action, resource }: { action: string; resource: string }) {
  return (
    <article className="grid gap-2 border-b border-border-subtle py-3 text-[13px] sm:flex sm:items-center sm:gap-3">
      <Avatar name="Cypra Admin" sub="00000000-0000-0000-0000-00000000cafe" size="sm" />
      <span className="flex-1">
        <span className="font-medium">Cypra Admin</span> <Tag variant="info">{action}</Tag>
      </span>
      <IdentifierPill value={resource} label="Resource ID" />
      <time className="text-text-secondary">2 minutes ago</time>
    </article>
  );
}

export function BackupCodeGrid({ codes }: { codes: string[] }) {
  const [confirmed, setConfirmed] = useState(false);
  useEffect(() => {
    if (confirmed) return undefined;
    const warn = (event: BeforeUnloadEvent) => {
      event.preventDefault();
      // eslint-disable-next-line @typescript-eslint/no-deprecated
      event.returnValue = "Your backup codes are shown only once. Continue without saving?";
    };
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [confirmed]);
  return (
    <Card title="Backup codes" subtitle="These codes are shown once. Save them before leaving.">
      <div className="grid grid-cols-2 gap-3">
        {codes.map((code) => (
          <IdentifierPill key={code} value={code} label="Backup code" />
        ))}
      </div>
      <div className="mt-4 flex gap-3">
        <Button variant="secondary">Download .txt</Button>
        <Button variant="primary" onClick={() => setConfirmed(true)}>
          {confirmed ? "Saved" : "I've saved these"}
        </Button>
      </div>
    </Card>
  );
}

export function PermissionMatrix() {
  return (
    <Card title="Permission matrix">
      <div className="grid grid-cols-4 gap-px overflow-hidden rounded-[var(--radius-md)] border border-border-default bg-border-default text-[13px]">
        {[
          "Role",
          "Read",
          "Write",
          "Admin",
          "Owner",
          "Yes",
          "Yes",
          "Yes",
          "Member",
          "Yes",
          "No",
          "No",
        ].map((cell, index) => (
          <span key={`${cell}-${String(index)}`} className="bg-bg-surface p-2">
            {cell}
          </span>
        ))}
      </div>
      <SaveBar dirtyCount={3} />
    </Card>
  );
}

export function SetupTokenBanner({ token }: { token: string }) {
  return (
    <Card
      highlighted
      title="Setup token"
      subtitle="This token is redacted on export. Copy it before continuing."
    >
      <IdentifierPill value={token} label="Setup token" />
      <div className="mt-4">
        <Button variant="primary">I have copied this - show next step</Button>
      </div>
    </Card>
  );
}

export function ProviderConfigCard({ kind }: { kind: "email" | "upstream" | "storage" }) {
  return (
    <Card
      title={`${kind} provider`}
      subtitle="Configuration health and current provider state."
      actions={<StatusPip variant="warn" label="Required" />}
    >
      <Button variant="secondary">Configure</Button>
    </Card>
  );
}

export function MobileBlockedBanner() {
  const [dismissed, setDismissed] = useState(false);
  if (dismissed) return null;
  return (
    <div className="flex items-center gap-3 rounded-[var(--radius-md)] border border-border-subtle bg-bg-surface p-3 text-[13px] text-text-secondary md:hidden">
      <Icons.Monitor className="h-4 w-4" />
      Cypra dashboard is optimized for desktop. Some features may be cramped.
      <IconButton
        label="Dismiss mobile banner"
        icon={<Icons.XCircle className="h-4 w-4" />}
        size="sm"
        onClick={() => setDismissed(true)}
      />
    </div>
  );
}

export function CodeBlock({ code }: { code: string }) {
  const [curl, setCurl] = useState(false);
  const shown = curl ? `curl ${code}` : code;
  return (
    <Card>
      <div className="mb-2 flex items-center justify-between">
        <span className="text-[12px] text-text-tertiary">shell</span>
        <Switch label="Copy as cURL" checked={curl} onChange={setCurl} />
      </div>
      <pre
        className="overflow-x-auto rounded-[var(--radius-md)] bg-bg-code p-3 font-mono text-[13px] text-text-identifier"
        tabIndex={0}
      >
        <code>{shown}</code>
      </pre>
    </Card>
  );
}

export function PageHeader({
  title,
  subtitle,
  action,
  loading = false,
}: {
  title: string;
  subtitle?: string;
  action?: ReactNode;
  loading?: boolean;
}) {
  return (
    <header className="mb-6 flex items-start justify-between gap-4">
      {loading ? (
        <div className="grid w-80 gap-2">
          <Skeleton shape="line" />
          <Skeleton shape="line" className="w-2/3" />
        </div>
      ) : (
        <div>
          <h1 className="text-[24px] font-semibold leading-8">{title}</h1>
          {subtitle ? (
            <p className="mt-1 max-w-2xl text-[14px] leading-[22px] text-text-secondary">
              {subtitle}
            </p>
          ) : null}
        </div>
      )}
      {action}
    </header>
  );
}

export function Breadcrumb({ segments }: { segments: string[] }) {
  return (
    <nav aria-label="Breadcrumb" className="mb-3 text-[12px] text-text-secondary">
      {segments.map((segment, index) => (
        <span key={segment}>
          {index > 0 ? " / " : null}
          {index === segments.length - 1 ? (
            <span className="text-text-primary">{segment}</span>
          ) : (
            segment
          )}
        </span>
      ))}
    </nav>
  );
}

export function EmptyState({
  title,
  body,
  action,
}: {
  title: string;
  body: string;
  action?: ReactNode;
}) {
  return (
    <div className="grid place-items-center rounded-[var(--radius-md)] border border-border-subtle bg-bg-surface p-10 text-center">
      <Icons.AlertTriangle className="mb-3 h-6 w-6 text-text-tertiary" />
      <h2 className="text-[20px] font-semibold leading-7">{title}</h2>
      <p className="mt-2 max-w-md text-[14px] text-text-secondary">{body}</p>
      {action ? <div className="mt-5">{action}</div> : null}
    </div>
  );
}

export function ErrorState({
  title,
  body,
  retry,
}: {
  title: string;
  body: string;
  retry?: () => void;
}) {
  return (
    <div className="rounded-[var(--radius-md)] border border-border-subtle bg-bg-surface p-6">
      <Icons.XCircle className="mb-3 h-6 w-6 text-status-error" />
      <h2 className="text-[20px] font-semibold leading-7">{title}</h2>
      <p className="mt-2 text-[14px] text-text-secondary">{body}</p>
      {retry ? (
        <Button className="mt-4" onClick={retry}>
          Retry
        </Button>
      ) : null}
    </div>
  );
}

export function LoadingState() {
  return (
    <div className="grid gap-4">
      <Skeleton shape="line" className="w-1/3" />
      <div className="grid gap-4 md:grid-cols-3">
        <Skeleton />
        <Skeleton />
        <Skeleton />
      </div>
    </div>
  );
}

export function ConfirmationDialog({ resource }: { resource: string }) {
  const [value, setValue] = useState("");
  return (
    <Card title="Confirm destructive action" subtitle={`Type ${resource} to continue.`}>
      <TextInput
        label="Resource identifier"
        value={value}
        onChange={(event) => setValue(event.target.value)}
      />
      <Button className="mt-4" variant="destructive" disabled={value.trim() !== resource}>
        Delete
      </Button>
    </Card>
  );
}

export function SettingsRow({
  label,
  helper,
  control,
}: {
  label: string;
  helper: string;
  control: ReactNode;
}) {
  return (
    <div className="flex items-center justify-between gap-6 border-b border-border-subtle py-4">
      <div>
        <div className="text-[14px] font-medium">{label}</div>
        <p className="mt-1 text-[13px] text-text-secondary">{helper}</p>
      </div>
      {control}
    </div>
  );
}

export function ListRow({ title, meta }: { title: string; meta: string }) {
  return (
    <div className="flex items-center gap-3 border-b border-border-subtle py-3">
      <Avatar name={title} sub={meta} size="sm" />
      <span className="flex-1 text-[14px]">{title}</span>
      <span className="text-[13px] text-text-secondary">{meta}</span>
      <IconButton
        label="Remove"
        icon={<Icons.Trash2 className="h-4 w-4" />}
        variant="destructive"
      />
    </div>
  );
}

export function SaveBar({ dirtyCount }: { dirtyCount: number }) {
  return (
    <div className="sticky bottom-4 mt-4 flex items-center justify-between rounded-[var(--radius-md)] border border-border-emphasis bg-bg-elevated p-3 shadow-[var(--shadow-lg)]">
      <span className="text-[13px] text-text-secondary">{dirtyCount} unsaved changes</span>
      <span className="flex gap-2">
        <Button variant="secondary" size="sm">
          Discard
        </Button>
        <Button variant="primary" size="sm">
          Save
        </Button>
      </span>
    </div>
  );
}

export function SidebarNav({
  collapsed = false,
  className,
  label = "Dashboard navigation",
}: {
  collapsed?: boolean;
  className?: string;
  label?: string;
}) {
  const items = ["Overview", "Users", "Projects", "Auth methods", "Members", "Audit", "Settings"];
  return (
    <aside
      className={cn(
        "flex min-h-screen flex-col border-r border-border-subtle bg-bg-surface p-3",
        collapsed ? "w-14" : "w-60",
        className,
      )}
      aria-label={label}
    >
      <div className="mb-4 flex items-center gap-2 px-2">
        <OidcGlyph className="h-5 w-5 text-accent-primary" />
        {collapsed ? null : <strong>Cypra</strong>}
      </div>
      {collapsed ? null : (
        <div className="mb-4">
          <ContextBadge />
          <div className="mt-3">
            <TenantSwitcher />
          </div>
        </div>
      )}
      <nav className="grid gap-1" aria-label={label}>
        {items.map((item) => (
          <a
            key={item}
            className="rounded-[var(--radius-md)] px-3 py-2 text-[13px] text-text-secondary hover:bg-bg-code hover:text-text-primary"
            href={
              item === "Overview"
                ? "/dashboard"
                : `/dashboard/${item.toLowerCase().replaceAll(" ", "-")}`
            }
          >
            {collapsed ? item[0] : item}
          </a>
        ))}
        <Tooltip label="Missing instance permission">
          <span className="inline-flex cursor-not-allowed items-center gap-2 rounded-[var(--radius-md)] px-3 py-2 text-[13px] text-text-secondary">
            <Icons.Lock className="h-4 w-4" />
            {collapsed ? null : "Diagnostics"}
          </span>
        </Tooltip>
      </nav>
      <div className="mt-auto grid gap-3 px-2">
        <ThemeToggle />
        {collapsed ? null : (
          <a className="text-[13px] text-text-secondary" href="/dashboard/account">
            Account
          </a>
        )}
      </div>
    </aside>
  );
}

export function ThemeToggle() {
  const { preference, setPreference } = useTheme();
  const options: ThemePreference[] = ["system", "dark", "light"];
  return (
    <SegmentedControl
      options={options}
      value={preference}
      onChange={(value) => setPreference(value as ThemePreference)}
    />
  );
}

export function PrimitiveGallery() {
  const [seg, setSeg] = useState("One");
  const codes = useMemo(
    () => [
      "CYPRA-1111",
      "CYPRA-2222",
      "CYPRA-3333",
      "CYPRA-4444",
      "CYPRA-5555",
      "CYPRA-6666",
      "CYPRA-7777",
      "CYPRA-8888",
      "CYPRA-9999",
      "CYPRA-0000",
    ],
    [],
  );
  return (
    <div className="grid gap-6">
      <PageHeader
        title="Primitive gallery"
        subtitle="Every primitive renders in both modes for visual regression."
      />
      <div className="grid gap-4 lg:grid-cols-2">
        <Card title="Buttons">
          <div className="flex flex-wrap gap-2">
            <Button variant="primary">Primary</Button>
            <Button>Secondary</Button>
            <Button variant="ghost">Ghost</Button>
            <Button variant="destructive">Delete</Button>
            <Button loading>Saving</Button>
            <IconButton label="Rotate" icon={<Icons.RotateCw className="h-4 w-4" />} />
          </div>
        </Card>
        <Card title="Inputs">
          <div className="grid gap-3">
            <TextInput
              label="Email"
              placeholder="admin@example.com"
              helper="Visible label, helper text."
            />
            <TextInput label="Slug" value="acme" state="success" readOnly />
            <Select
              label="Role"
              options={[
                { value: "owner", label: "Owner" },
                { value: "member", label: "Member" },
              ]}
            />
            <Checkbox label="Require MFA" />
            <Radio label="Password" name="method" />
            <Switch label="Enabled" checked />
          </div>
        </Card>
        <Card title="Identity">
          <div className="flex flex-wrap items-center gap-3">
            <Tag variant="success">Healthy</Tag>
            <IdentifierPill value="client_cypra_acme_0123456789abcdef" label="Client ID" />
            <MaskedSecret name="client_secret" value="client_secret_real_value" />
            <Avatar name="Ada Lovelace" sub="00000000-0000-0000-0000-00000000ada1" />
          </div>
        </Card>
        <Card title="Status">
          <div className="grid gap-2">
            {Object.keys(statusClass).map((variant) => (
              <StatusPip key={variant} variant={variant as StatusVariant} label={variant} />
            ))}
          </div>
        </Card>
        <Card title="Navigation">
          <Tabs
            tabs={[
              {
                id: "a",
                label: "First",
                content: (
                  <EmptyState
                    title="No projects yet."
                    body="A project lets an app authenticate against this tenant."
                    action={<Button>Create project</Button>}
                  />
                ),
              },
              { id: "b", label: "Second", content: <LoadingState /> },
            ]}
          />
        </Card>
        <Card title="Composite">
          <div className="grid gap-4">
            <KeyRotationTimeline />
            <AuditEntry action="created" resource="tenant_acme_001" />
            <BackupCodeGrid codes={codes} />
            <ProviderConfigCard kind="email" />
            <MobileBlockedBanner />
            <CodeBlock code="https://cypra.localhost/.well-known/openid-configuration" />
            <PermissionMatrix />
            <ConfirmationDialog resource="acme" />
            <SettingsRow
              label="Branding preview"
              helper="Open a hosted-login preview before saving."
              control={<Button>Preview</Button>}
            />
            <ListRow title="Ada Lovelace" meta="owner" />
            <SaveBar dirtyCount={2} />
          </div>
        </Card>
        <Card title="Segmented">
          <SegmentedControl options={["One", "Two", "Three"]} value={seg} onChange={setSeg} />
        </Card>
        <Card title="Glyphs">
          <div className="flex gap-4 text-accent-primary">
            <TenantGlyph className="h-8 w-8" />
            <InstanceGlyph className="h-8 w-8" />
            <PasskeyGlyph className="h-8 w-8" />
            <OidcGlyph className="h-8 w-8" />
          </div>
        </Card>
      </div>
      <div
        data-mode="light"
        className="rounded-[var(--radius-lg)] bg-bg-canvas p-4 text-text-primary"
      >
        <PageHeader
          title="Light mode parity"
          subtitle="Token-bound rendering inside a light-mode island."
        />
        <StatusPip variant="success" label="Light mode healthy" />
      </div>
      <div
        data-mode="dark"
        className="rounded-[var(--radius-lg)] bg-bg-canvas p-4 text-text-primary"
      >
        <PageHeader
          title="Dark mode parity"
          subtitle="Token-bound rendering inside a dark-mode island."
        />
        <StatusPip variant="pending" label="Dark mode pending" />
      </div>
    </div>
  );
}
