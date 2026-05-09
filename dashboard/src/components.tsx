import {
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
  type ButtonHTMLAttributes,
  type ComponentType,
  type InputHTMLAttributes,
  type KeyboardEvent,
  type PropsWithChildren,
  type ReactNode,
  type SelectHTMLAttributes,
  type SVGProps,
} from "react";

import { useQuery } from "@tanstack/react-query";

import { listTenants, signOut } from "@/api";
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
        "inline-flex items-center justify-center whitespace-nowrap rounded-[var(--radius-md)] border font-medium transition duration-[var(--dur-instant)] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-border-focus active:scale-[0.985] disabled:cursor-not-allowed disabled:opacity-50",
        size === "sm" && "h-7 gap-2 px-3 text-[13px]",
        size === "md" && "h-9 gap-2 px-4 text-[14px]",
        size === "lg" && "h-11 gap-2.5 px-5 text-[14px]",
        variant === "primary" &&
          "border-accent-primary bg-accent-primary text-text-on-accent hover:bg-accent-primary-hi active:bg-accent-primary-lo",
        variant === "secondary" &&
          "border-border-default bg-bg-surface text-text-primary hover:bg-bg-elevated",
        variant === "ghost" &&
          "border-transparent bg-transparent text-text-secondary hover:bg-bg-elevated hover:text-text-primary",
        variant === "destructive" &&
          "border-status-error bg-status-error text-white hover:opacity-90",
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
      <span className={cn("min-w-0 overflow-hidden text-ellipsis", loading && "sr-only")}>
        {children}
      </span>
      {kbd ? (
        <kbd className="rounded bg-bg-code px-1.5 py-0.5 text-[12px] text-text-primary">{kbd}</kbd>
      ) : null}
      {trailing}
    </button>
  );
}

type IconButtonProps = Omit<ButtonHTMLAttributes<HTMLButtonElement>, "children"> & {
  label: string;
  icon: ReactNode;
  size?: "sm" | "md" | "lg";
  variant?: "ghost" | "subtle" | "accent" | "destructive";
  loading?: boolean;
};

export function IconButton({
  label,
  icon,
  className,
  size = "md",
  variant = "ghost",
  loading = false,
  disabled,
  ...props
}: IconButtonProps) {
  return (
    <button
      className={cn(
        "inline-flex items-center justify-center rounded-[var(--radius-md)] border transition duration-[var(--dur-instant)] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-border-focus active:scale-[0.985] disabled:cursor-not-allowed disabled:opacity-50",
        size === "sm" && "h-7 w-7",
        size === "md" && "h-8 w-8",
        size === "lg" && "h-10 w-10",
        variant === "ghost" &&
          "border-transparent bg-transparent text-text-secondary hover:bg-bg-elevated hover:text-text-primary active:bg-accent-primary-mu active:text-accent-primary",
        variant === "subtle" &&
          "border-border-default bg-bg-surface text-text-secondary hover:bg-bg-elevated hover:text-text-primary",
        variant === "accent" &&
          "border-transparent bg-accent-primary-mu text-accent-primary hover:text-accent-primary-hi",
        variant === "destructive" &&
          "border-transparent bg-transparent text-status-error hover:bg-status-error/10",
        className,
      )}
      title={label}
      aria-label={label}
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
        icon
      )}
    </button>
  );
}

type TextInputProps = Omit<InputHTMLAttributes<HTMLInputElement>, "size"> & {
  label: string;
  helper?: string;
  error?: string;
  state?: "default" | "validating" | "success" | "readonly";
  size?: "sm" | "md";
  variant?: "default" | "borderless";
};

export function TextInput({
  label,
  helper,
  error,
  state = "default",
  size = "md",
  variant = "default",
  className,
  id,
  disabled,
  ...props
}: TextInputProps) {
  const generatedId = useId();
  const inputId = id ?? generatedId;
  const helperId = `${inputId}-helper`;
  const describedBy = error !== undefined || helper !== undefined ? helperId : undefined;
  const readonly = state === "readonly" || Boolean(props.readOnly);
  return (
    <label className="grid gap-1.5 text-[13px] font-medium text-text-primary" htmlFor={inputId}>
      <span>{label}</span>
      <span className="relative">
        <input
          id={inputId}
          aria-label={label}
          className={cn(
            "w-full rounded-[var(--radius-md)] border text-text-primary transition duration-[var(--dur-instant)] placeholder:text-text-tertiary",
            "focus:outline-2 focus:outline-offset-0 focus:outline-border-focus",
            size === "sm" && "h-8 px-2.5 text-[13px]",
            size === "md" && "h-10 px-3 text-[14px]",
            variant === "default" &&
              "border-border-default bg-bg-surface hover:border-border-emphasis",
            variant === "borderless" && "border-transparent bg-transparent",
            readonly && "border-border-subtle bg-bg-code hover:border-border-subtle",
            disabled &&
              "border-border-default bg-bg-surface opacity-50 hover:border-border-default",
            error && "border-status-error hover:border-status-error",
            state === "success" && "border-status-success hover:border-status-success",
            className,
          )}
          aria-invalid={error !== undefined ? true : undefined}
          aria-describedby={describedBy}
          aria-busy={state === "validating" ? true : undefined}
          readOnly={readonly}
          disabled={disabled}
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

interface ImageUploadProps {
  label?: string;
  helper?: string;
  error?: string;
  accept?: string;
  maxSizeBytes?: number;
  value?: File | null;
  initialPreviewUrl?: string;
  disabled?: boolean;
  onChange?: (file: File | null) => void;
  id?: string;
}

export function ImageUpload({
  label,
  helper,
  error,
  accept = "image/png,image/svg+xml,image/jpeg,image/webp",
  maxSizeBytes,
  value,
  initialPreviewUrl,
  disabled = false,
  onChange,
  id,
}: ImageUploadProps) {
  const generatedId = useId();
  const inputId = id ?? generatedId;
  const helperId = `${inputId}-helper`;
  const inputRef = useRef<HTMLInputElement>(null);
  const [dragOver, setDragOver] = useState(false);
  const [internalError, setInternalError] = useState<string | undefined>();
  const [objectUrl, setObjectUrl] = useState<string | undefined>();

  useEffect(() => {
    if (!value) {
      setObjectUrl(undefined);
      return;
    }
    const url = URL.createObjectURL(value);
    setObjectUrl(url);
    return () => URL.revokeObjectURL(url);
  }, [value]);

  const previewUrl = objectUrl ?? initialPreviewUrl;
  const shownError = error ?? internalError;
  const helperText = shownError ?? helper;

  const acceptList = useMemo(
    () =>
      accept
        .split(",")
        .map((entry) => entry.trim())
        .filter(Boolean),
    [accept],
  );

  const acceptsFile = (file: File) => {
    if (acceptList.length === 0) return true;
    return acceptList.some((pattern) => {
      if (pattern.endsWith("/*")) {
        const prefix = pattern.slice(0, -1);
        return file.type.startsWith(prefix);
      }
      if (pattern.startsWith(".")) {
        return file.name.toLowerCase().endsWith(pattern.toLowerCase());
      }
      return file.type === pattern;
    });
  };

  const handleFile = (file: File | undefined) => {
    if (!file) return;
    if (!acceptsFile(file)) {
      setInternalError("This file type isn't supported.");
      onChange?.(null);
      return;
    }
    if (maxSizeBytes && file.size > maxSizeBytes) {
      setInternalError(`File must be smaller than ${formatBytes(maxSizeBytes)}.`);
      onChange?.(null);
      return;
    }
    setInternalError(undefined);
    onChange?.(file);
  };

  const openPicker = () => {
    if (disabled) return;
    inputRef.current?.click();
  };

  const clear = () => {
    if (disabled) return;
    setInternalError(undefined);
    if (inputRef.current) inputRef.current.value = "";
    onChange?.(null);
  };

  const onKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (disabled) return;
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      openPicker();
    }
  };

  return (
    <div className="grid w-full gap-1.5 text-[13px] font-medium text-text-primary">
      {label ? (
        <label htmlFor={inputId} className="cursor-default">
          {label}
        </label>
      ) : null}
      <input
        ref={inputRef}
        id={inputId}
        type="file"
        accept={accept}
        className="sr-only"
        aria-label={label ? undefined : "Upload image"}
        aria-describedby={helperText ? helperId : undefined}
        aria-invalid={shownError ? true : undefined}
        disabled={disabled}
        onChange={(event) => handleFile(event.currentTarget.files?.[0])}
      />
      {previewUrl ? (
        <div
          className={cn(
            "flex min-w-0 items-center gap-3 overflow-hidden rounded-[var(--radius-md)] border border-border-default bg-bg-surface p-3",
            shownError && "border-status-error",
            disabled && "opacity-50",
          )}
        >
          <div className="grid h-14 w-14 flex-shrink-0 place-items-center overflow-hidden rounded-[var(--radius-sm)] border border-border-subtle bg-bg-canvas">
            <img
              src={previewUrl}
              alt={value?.name ?? "Selected image"}
              className="h-full w-full object-contain"
            />
          </div>
          <div className="min-w-0 flex-1">
            <div className="truncate text-[13px] font-medium text-text-primary">
              {value?.name ?? "Current image"}
            </div>
            <div className="text-[12px] font-normal text-text-secondary">
              {value ? formatBytes(value.size) : "Saved logo"}
            </div>
          </div>
          <span className="flex-shrink-0">
            <IconButton
              label="Remove image"
              icon={<Icons.Trash2 className="h-4 w-4" />}
              variant="destructive"
              onClick={clear}
              disabled={disabled}
            />
          </span>
        </div>
      ) : (
        <div
          role="button"
          tabIndex={disabled ? -1 : 0}
          aria-disabled={disabled || undefined}
          aria-invalid={shownError ? true : undefined}
          aria-describedby={helperText ? helperId : undefined}
          onClick={openPicker}
          onKeyDown={onKeyDown}
          onDragOver={(event) => {
            if (disabled) return;
            event.preventDefault();
            setDragOver(true);
          }}
          onDragLeave={() => setDragOver(false)}
          onDrop={(event) => {
            event.preventDefault();
            setDragOver(false);
            if (disabled) return;
            handleFile(event.dataTransfer.files[0]);
          }}
          className={cn(
            "flex flex-col items-center justify-center gap-1.5 rounded-[var(--radius-md)] border border-dashed px-4 py-6 text-center outline-none transition duration-[var(--dur-instant)]",
            "focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-border-focus",
            !disabled && "cursor-pointer hover:border-border-emphasis hover:bg-bg-elevated",
            dragOver
              ? "border-accent-primary bg-accent-primary-mu"
              : "border-border-default bg-bg-surface",
            shownError && "border-status-error",
            disabled && "opacity-50",
          )}
        >
          <Icons.ImageUp className="h-5 w-5 text-text-secondary" aria-hidden="true" />
          <div className="text-[13px] font-medium text-text-primary">
            <span className="text-accent-primary">Click to upload</span>
            <span className="text-text-secondary"> or drag and drop</span>
          </div>
          <div className="text-[12px] font-normal text-text-tertiary">
            {describeAccept(acceptList)}
            {maxSizeBytes ? ` • up to ${formatBytes(maxSizeBytes)}` : ""}
          </div>
        </div>
      )}
      {helperText ? (
        <span
          id={helperId}
          className={cn(
            "text-[12px] font-normal",
            shownError ? "text-status-error" : "text-text-secondary",
          )}
        >
          {helperText}
        </span>
      ) : null}
    </div>
  );
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${String(bytes)} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function describeAccept(accept: string[]): string {
  if (accept.length === 0) return "Any image";
  const exts = accept
    .map((entry) => {
      if (entry === "image/svg+xml") return "SVG";
      if (entry === "image/png") return "PNG";
      if (entry === "image/jpeg") return "JPG";
      if (entry === "image/webp") return "WEBP";
      if (entry === "image/gif") return "GIF";
      if (entry === "image/*") return "Image";
      if (entry.startsWith(".")) return entry.slice(1).toUpperCase();
      const slash = entry.indexOf("/");
      return slash >= 0 ? entry.slice(slash + 1).toUpperCase() : entry.toUpperCase();
    })
    .filter((entry, index, all) => all.indexOf(entry) === index);
  return exts.join(", ");
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
      <span className="relative">
        <select
          id={selectId}
          className={cn(
            "h-10 w-full appearance-none rounded-[var(--radius-md)] border border-border-default bg-bg-surface pl-3 pr-9 text-[14px] text-text-primary focus:outline-2 focus:outline-offset-0 focus:outline-border-focus",
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
        <Icons.ChevronDown
          className="pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-text-secondary"
          aria-hidden="true"
        />
      </span>
      {helper ? (
        <span className="text-[12px] font-normal text-text-secondary">{helper}</span>
      ) : null}
    </label>
  );
}

type ComboboxState = "default" | "loading" | "empty" | "error" | "readonly";

interface ComboboxOption {
  value: string;
  label: string;
  group?: string;
}

export function Combobox({
  label,
  options,
  value,
  onChange,
  placeholder = "Search...",
  helper,
  state = "default",
  error,
  defaultOpen = false,
}: {
  label: string;
  options: ComboboxOption[];
  value?: string;
  onChange?: (value: string) => void;
  placeholder?: string;
  helper?: string;
  state?: ComboboxState;
  error?: string;
  defaultOpen?: boolean;
}) {
  const id = useId();
  const [open, setOpen] = useState(defaultOpen);
  const [query, setQuery] = useState("");
  const [activeIndex, setActiveIndex] = useState(0);
  const selected = options.find((option) => option.value === value);
  const filtered = options.filter((option) =>
    option.label.toLowerCase().includes(query.trim().toLowerCase()),
  );
  const disabled = state === "readonly";
  const empty = state === "empty" || (query.trim() !== "" && filtered.length === 0);
  const shownOptions = state === "loading" || state === "error" || empty ? [] : filtered;
  const hasOptions = shownOptions.length > 0;

  const commit = (option: ComboboxOption | undefined) => {
    if (!option) return;
    onChange?.(option.value);
    setQuery(option.label);
    setOpen(false);
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (disabled) return;
    if (event.key === "ArrowDown") {
      event.preventDefault();
      setOpen(true);
      setActiveIndex((current) => Math.min(current + 1, Math.max(shownOptions.length - 1, 0)));
    }
    if (event.key === "ArrowUp") {
      event.preventDefault();
      setOpen(true);
      setActiveIndex((current) => Math.max(current - 1, 0));
    }
    if (event.key === "Home") {
      event.preventDefault();
      setActiveIndex(0);
    }
    if (event.key === "End") {
      event.preventDefault();
      setActiveIndex(Math.max(shownOptions.length - 1, 0));
    }
    if (event.key === "Enter" && open) {
      event.preventDefault();
      commit(shownOptions[activeIndex]);
    }
    if (event.key === "Escape") {
      setOpen(false);
    }
  };

  const groups = shownOptions.reduce<Record<string, ComboboxOption[]>>((result, option) => {
    const group = option.group ?? "Options";
    result[group] = [...(result[group] ?? []), option];
    return result;
  }, {});

  return (
    <div className="relative grid gap-1.5 text-[13px] font-medium text-text-primary">
      <label htmlFor={id}>{label}</label>
      <input
        id={id}
        role="combobox"
        aria-expanded={open}
        aria-controls={`${id}-listbox`}
        aria-autocomplete="list"
        aria-invalid={state === "error" || error ? true : undefined}
        aria-busy={state === "loading" ? true : undefined}
        readOnly={disabled}
        disabled={disabled}
        className={cn(
          "h-10 w-full rounded-[var(--radius-md)] border border-border-default bg-bg-surface px-3 text-[14px] text-text-primary placeholder:text-text-tertiary",
          (state === "error" || error) && "border-status-error",
          disabled && "bg-bg-code text-text-secondary",
        )}
        placeholder={selected?.label ?? placeholder}
        value={query}
        onChange={(event) => {
          setQuery(event.target.value);
          setOpen(true);
          setActiveIndex(0);
        }}
        onFocus={() => setOpen(!disabled)}
        onKeyDown={handleKeyDown}
      />
      {helper || error ? (
        <span
          className={cn(
            "text-[12px] font-normal",
            error ? "text-status-error" : "text-text-secondary",
          )}
        >
          {error ?? helper}
        </span>
      ) : null}
      {open ? (
        <div
          id={`${id}-listbox`}
          role={hasOptions ? "listbox" : "status"}
          className="absolute left-0 right-0 top-[calc(100%+0.5rem)] z-30 max-h-64 overflow-auto rounded-[var(--radius-md)] border border-border-default bg-bg-elevated p-2 shadow-[var(--shadow-md)]"
        >
          {state === "loading" ? (
            <div className="grid gap-2">
              <Skeleton shape="line" />
              <Skeleton shape="line" />
              <Skeleton shape="line" />
            </div>
          ) : null}
          {state === "error" ? (
            <div className="grid gap-2 rounded-[var(--radius-md)] bg-bg-code p-3 text-[13px] text-status-error">
              Options failed to load.
              <Button size="sm" variant="secondary">
                Retry
              </Button>
            </div>
          ) : null}
          {empty ? (
            <div className="rounded-[var(--radius-md)] bg-bg-code p-3 text-center text-[13px] text-text-secondary">
              No matches for{" "}
              {query.trim() ? <span className="font-mono">{query}</span> : "this filter"}
            </div>
          ) : null}
          {Object.entries(groups).map(([group, groupOptions]) => (
            <div key={group} className="grid gap-1" role="group" aria-label={group}>
              <div
                aria-hidden="true"
                className="px-2 py-1 text-[11px] font-semibold uppercase tracking-[0.06em] text-text-tertiary"
              >
                {group}
              </div>
              {groupOptions.map((option) => {
                const optionIndex = shownOptions.findIndex((item) => item.value === option.value);
                const active = optionIndex === activeIndex;
                return (
                  <button
                    key={option.value}
                    type="button"
                    role="option"
                    aria-selected={option.value === value}
                    className={cn(
                      "rounded-[var(--radius-sm)] px-2 py-2 text-left text-[13px] text-text-primary",
                      active && "bg-accent-primary-mu",
                      option.value === value && "font-semibold",
                    )}
                    onMouseEnter={() => setActiveIndex(optionIndex)}
                    onClick={() => commit(option)}
                  >
                    {option.label}
                  </button>
                );
              })}
            </div>
          ))}
        </div>
      ) : null}
    </div>
  );
}

export function Checkbox({
  label,
  helper,
  ...props
}: InputHTMLAttributes<HTMLInputElement> & { label: string; helper?: string }) {
  return (
    <label className="flex items-start gap-3 text-[14px] text-text-primary">
      <input
        type="checkbox"
        className="mt-0.5 h-5 w-5 rounded-[var(--radius-sm)] accent-[var(--accent-primary)] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-border-focus disabled:opacity-50"
        {...props}
      />
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
      <input
        type="radio"
        className="h-5 w-5 accent-[var(--accent-primary)] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-border-focus disabled:opacity-50"
        {...props}
      />
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
      className="group inline-flex items-center gap-3 text-[14px] disabled:cursor-not-allowed disabled:opacity-50"
      type="button"
      role="switch"
      aria-checked={checked}
      aria-busy={pending ? true : undefined}
      onClick={() => !pending && onChange?.(!checked)}
    >
      <span
        className={cn(
          "flex h-5 w-9 items-center rounded-[var(--radius-pill)] p-0.5 transition duration-[var(--dur-fast)] group-focus-visible:outline-2 group-focus-visible:outline-offset-2 group-focus-visible:outline-border-focus",
          pending ? "bg-accent-primary-mu" : checked ? "bg-accent-primary" : "bg-border-emphasis",
        )}
      >
        <span
          className={cn(
            "grid h-4 w-4 place-items-center rounded-[var(--radius-pill)] bg-bg-canvas transition duration-[var(--dur-fast)]",
            checked && !pending && "translate-x-4",
          )}
        >
          {pending ? <Icons.Clock className="h-2.5 w-2.5 text-status-pending" /> : null}
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
            : variant === "neutral"
              ? "text-text-primary"
              : "text-text-secondary";
  const background = variant === "accent" ? "bg-accent-primary-mu" : "bg-bg-code";
  return (
    <span
      className={cn(
        "inline-flex h-6 items-center gap-1.5 whitespace-nowrap rounded-[var(--radius-pill)] px-2.5 text-[11px] font-semibold uppercase tracking-[0.06em]",
        background,
        color,
      )}
    >
      <StatusSigil variant={variant in statusClass ? (variant as StatusVariant) : "info"} />
      {children}
    </span>
  );
}

export function Tooltip({ label, children }: PropsWithChildren<{ label: string }>) {
  const tooltipId = useId();
  return (
    <span className="group relative inline-flex">
      <span aria-describedby={tooltipId}>{children}</span>
      <span
        id={tooltipId}
        role="tooltip"
        className="pointer-events-none absolute bottom-full left-1/2 z-20 mb-2 hidden -translate-x-1/2 whitespace-nowrap rounded-[var(--radius-sm)] border border-border-emphasis bg-bg-elevated px-2.5 py-1.5 text-[13px] text-text-primary shadow-[var(--shadow-md)] group-hover:block group-focus-within:block"
      >
        {label}
      </span>
    </span>
  );
}

export function InlineAlert({
  variant = "error",
  title,
  message,
  action,
}: {
  variant?: "error" | "warn" | "info" | "success";
  title?: string;
  message: ReactNode;
  action?: ReactNode;
}) {
  const accent = {
    error: {
      border: "border-status-error",
      icon: "text-status-error",
      Icon: Icons.XCircle,
      role: "alert" as const,
    },
    warn: {
      border: "border-status-warn",
      icon: "text-status-warn",
      Icon: Icons.AlertTriangle,
      role: "status" as const,
    },
    info: {
      border: "border-status-info",
      icon: "text-status-info",
      Icon: Icons.Info,
      role: "status" as const,
    },
    success: {
      border: "border-status-success",
      icon: "text-status-success",
      Icon: Icons.CheckCircle2,
      role: "status" as const,
    },
  }[variant];
  return (
    <div
      role={accent.role}
      className={cn(
        "flex w-full items-start gap-3 rounded-[var(--radius-md)] border-y border-r border-l-[3px] bg-bg-elevated px-4 py-3",
        accent.border,
      )}
    >
      <accent.Icon className={cn("mt-0.5 h-4 w-4 flex-shrink-0", accent.icon)} aria-hidden="true" />
      <div className="grid flex-1 gap-1">
        {title ? <p className="text-[13px] font-semibold text-text-primary">{title}</p> : null}
        <div className="text-[13px] leading-[1.5] text-text-secondary">{message}</div>
      </div>
      {action ? <div className="flex-shrink-0">{action}</div> : null}
    </div>
  );
}

export function Toast({
  variant,
  message,
  action,
  onDismiss,
}: {
  variant: StatusVariant;
  message: string;
  action?: ReactNode;
  onDismiss?: () => void;
}) {
  const live = variant === "error" || variant === "warn" ? "assertive" : "polite";
  return (
    <div
      role="status"
      aria-live={live}
      className="flex w-[380px] items-center gap-3 rounded-[var(--radius-md)] border border-border-emphasis bg-bg-elevated px-4 py-3 text-[14px] text-text-primary shadow-[var(--shadow-lg)]"
    >
      <span className={statusClass[variant]}>
        <StatusSigil variant={variant} />
      </span>
      <span className="flex-1">{message}</span>
      {action}
      {onDismiss ? (
        <IconButton
          label="Dismiss"
          icon={<Icons.XCircle className="h-3.5 w-3.5" />}
          size="sm"
          onClick={onDismiss}
        />
      ) : null}
    </div>
  );
}

export function Modal({
  title,
  headerIcon,
  open,
  children,
  footer,
  size = "md",
  onClose,
}: PropsWithChildren<{
  title: string;
  headerIcon?: ReactNode;
  open: boolean;
  footer?: ReactNode;
  size?: "sm" | "md" | "lg";
  onClose: () => void;
}>) {
  useEffect(() => {
    if (!open) return;
    const handler = (event: globalThis.KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [open, onClose]);
  if (!open) {
    return null;
  }
  const widthClass =
    size === "sm" ? "max-w-[400px]" : size === "lg" ? "max-w-[720px]" : "max-w-[560px]";
  return (
    <div
      className="fixed inset-0 z-40 grid place-items-center bg-[var(--bg-overlay)] p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="modal-title"
    >
      <section
        className={cn(
          "w-full rounded-[var(--radius-lg)] border border-border-emphasis bg-bg-elevated shadow-[var(--shadow-lg)]",
          widthClass,
        )}
      >
        <header className="flex items-start justify-between gap-3 px-5 py-4">
          <div className="flex items-start gap-3">
            {headerIcon ? <span className="flex-shrink-0 pt-0.5">{headerIcon}</span> : null}
            <h2 id="modal-title" className="text-[20px] font-semibold leading-[1.4]">
              {title}
            </h2>
          </div>
          <IconButton
            label="Close"
            icon={<Icons.XCircle className="h-4 w-4" />}
            onClick={onClose}
          />
        </header>
        <div className="px-5 py-2">{children}</div>
        {footer ? (
          <footer className="flex items-center justify-end gap-3 border-t border-border-subtle px-5 py-4">
            {footer}
          </footer>
        ) : null}
      </section>
    </div>
  );
}

export function Drawer({
  title,
  open,
  width = 480,
  children,
  footer,
  onClose,
}: PropsWithChildren<{
  title: string;
  open: boolean;
  width?: number;
  footer?: ReactNode;
  onClose: () => void;
}>) {
  useEffect(() => {
    if (!open) return;
    const handler = (event: globalThis.KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [open, onClose]);
  if (!open) return null;
  return (
    <div
      className="fixed inset-0 z-40 flex"
      role="dialog"
      aria-modal="true"
      aria-labelledby="drawer-title"
    >
      <button
        type="button"
        aria-label="Close drawer"
        className="flex-1 bg-[var(--bg-overlay)]"
        onClick={onClose}
      />
      <section
        className="ml-auto flex h-full flex-col border-l border-border-emphasis bg-bg-elevated shadow-[var(--shadow-lg)]"
        style={{ width }}
      >
        <header className="flex items-center justify-between gap-3 border-b border-border-subtle px-5 py-4">
          <h2 id="drawer-title" className="text-[20px] font-semibold leading-[1.4]">
            {title}
          </h2>
          <IconButton
            label="Close"
            icon={<Icons.XCircle className="h-4 w-4" />}
            onClick={onClose}
          />
        </header>
        <div className="min-h-0 flex-1 overflow-y-auto p-5">{children}</div>
        {footer ? (
          <footer className="flex items-center justify-end gap-3 border-t border-border-subtle px-5 py-4">
            {footer}
          </footer>
        ) : null}
      </section>
    </div>
  );
}

export function Popover({
  label,
  side = "left",
  defaultOpen = false,
  children,
}: PropsWithChildren<{ label: string; side?: "left" | "right"; defaultOpen?: boolean }>) {
  const panelId = useId();
  const [open, setOpen] = useState(defaultOpen);
  return (
    <span className="relative inline-block">
      <button
        type="button"
        aria-expanded={open}
        aria-controls={panelId}
        className="rounded-[var(--radius-md)] border border-border-default bg-bg-surface px-3 py-2 text-[13px]"
        onClick={() => setOpen((current) => !current)}
        onKeyDown={(event) => {
          if (event.key === "Escape") setOpen(false);
          if (event.key === "ArrowDown") setOpen(true);
        }}
      >
        {label}
      </button>
      {open ? (
        <div
          id={panelId}
          className={cn(
            "absolute z-20 mt-2 min-w-52 rounded-[var(--radius-md)] border border-border-default bg-bg-elevated p-2 shadow-[var(--shadow-md)]",
            side === "right" ? "right-0" : "left-0",
          )}
          onKeyDown={(event) => {
            if (event.key === "Escape") setOpen(false);
          }}
        >
          {children}
        </div>
      ) : null}
    </span>
  );
}

export function DropdownMenu({
  label,
  items,
  defaultOpen = false,
}: {
  label: string;
  items: { label: string; destructive?: boolean; disabled?: boolean; onSelect?: () => void }[];
  defaultOpen?: boolean;
}) {
  const menuId = useId();
  const [open, setOpen] = useState(defaultOpen);
  const [activeIndex, setActiveIndex] = useState(0);
  const itemRefs = useRef<(HTMLButtonElement | null)[]>([]);
  const enabledItems = items
    .map((item, index) => ({ ...item, index }))
    .filter((item) => !item.disabled);

  const focusItem = (index: number) => {
    const nextIndex =
      enabledItems.length === 0 ? 0 : (enabledItems[index]?.index ?? enabledItems[0].index);
    setActiveIndex(nextIndex);
    window.setTimeout(() => itemRefs.current[nextIndex]?.focus(), 0);
  };

  const openMenu = () => {
    setOpen(true);
    focusItem(0);
  };

  const handleMenuKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    const enabledIndex = Math.max(
      enabledItems.findIndex((item) => item.index === activeIndex),
      0,
    );
    if (event.key === "Escape") {
      event.preventDefault();
      setOpen(false);
    }
    if (event.key === "ArrowDown") {
      event.preventDefault();
      focusItem((enabledIndex + 1) % Math.max(enabledItems.length, 1));
    }
    if (event.key === "ArrowUp") {
      event.preventDefault();
      focusItem((enabledIndex - 1 + enabledItems.length) % Math.max(enabledItems.length, 1));
    }
    if (event.key === "Home") {
      event.preventDefault();
      focusItem(0);
    }
    if (event.key === "End") {
      event.preventDefault();
      focusItem(Math.max(enabledItems.length - 1, 0));
    }
    if (/^[a-z0-9]$/i.test(event.key)) {
      const match = enabledItems.find((item) =>
        item.label.toLowerCase().startsWith(event.key.toLowerCase()),
      );
      if (match) {
        event.preventDefault();
        setActiveIndex(match.index);
        window.setTimeout(() => itemRefs.current[match.index]?.focus(), 0);
      }
    }
  };

  return (
    <span className="relative inline-block">
      <button
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={menuId}
        className="rounded-[var(--radius-md)] border border-border-default bg-bg-surface px-3 py-2 text-[13px]"
        onClick={() => (open ? setOpen(false) : openMenu())}
        onKeyDown={(event) => {
          if (event.key === "ArrowDown" || event.key === "Enter" || event.key === " ") {
            event.preventDefault();
            openMenu();
          }
          if (event.key === "Escape") setOpen(false);
        }}
      >
        {label}
      </button>
      {open ? (
        <div
          id={menuId}
          role="menu"
          className="absolute left-0 z-30 mt-2 grid min-w-52 gap-1 rounded-[var(--radius-md)] border border-border-default bg-bg-elevated p-2 shadow-[var(--shadow-md)]"
          onKeyDown={handleMenuKeyDown}
        >
          {items.map((item, index) => (
            <button
              key={item.label}
              ref={(node) => {
                itemRefs.current[index] = node;
              }}
              type="button"
              role="menuitem"
              disabled={item.disabled}
              tabIndex={index === activeIndex ? 0 : -1}
              className={cn(
                "rounded-[var(--radius-sm)] px-2 py-2 text-left text-[13px] text-text-primary disabled:text-text-disabled",
                index === activeIndex && "bg-accent-primary-mu",
                item.destructive && "text-status-error",
              )}
              onMouseEnter={() => setActiveIndex(index)}
              onClick={() => {
                item.onSelect?.();
                setOpen(false);
              }}
            >
              {item.label}
            </button>
          ))}
        </div>
      ) : null}
    </span>
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
              "border-b-2 px-4 py-3 text-[13px] font-medium focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-border-focus disabled:cursor-not-allowed disabled:opacity-50",
              tab.id === current.id
                ? "border-accent-primary text-accent-primary"
                : "border-transparent text-text-secondary hover:text-text-primary",
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
  variant,
  disabled = false,
}: PropsWithChildren<{
  title?: string;
  subtitle?: string;
  actions?: ReactNode;
  highlighted?: boolean;
  variant?: "standard" | "interactive" | "highlighted" | "disabled";
  disabled?: boolean;
  className?: string;
}>) {
  const resolved = variant ?? (disabled ? "disabled" : highlighted ? "highlighted" : "standard");
  return (
    <section
      className={cn(
        "min-w-0 rounded-[var(--radius-md)] border bg-bg-surface p-6 shadow-[var(--shadow-sm)]",
        resolved === "standard" && "border-border-subtle",
        resolved === "interactive" &&
          "border-border-subtle transition duration-[var(--dur-instant)] hover:border-border-emphasis focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-border-focus",
        resolved === "highlighted" && "border-accent-primary bg-accent-primary-mu",
        resolved === "disabled" && "border-border-subtle opacity-50",
        className,
      )}
    >
      {title !== undefined || actions !== undefined ? (
        <header className="mb-3 flex items-start justify-between gap-4">
          <div>
            {title ? <h2 className="text-[20px] font-semibold leading-7">{title}</h2> : null}
            {subtitle ? (
              <p className="mt-1 text-[14px] leading-[1.571] text-text-secondary">{subtitle}</p>
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
    xs: "h-5 w-5 text-[9px]",
    sm: "h-6 w-6 text-[10px]",
    md: "h-8 w-8 text-[13px]",
    lg: "h-10 w-10 text-[14px]",
    xl: "h-16 w-16 text-[24px]",
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
        shape === "line" && "h-3.5 w-[200px] rounded-[var(--radius-sm)]",
        shape === "circle" && "h-10 w-10 rounded-[var(--radius-pill)]",
        shape === "block" && "h-20 w-[120px] rounded-[var(--radius-md)]",
        className,
      )}
    />
  );
}

export function SegmentedControl({
  options,
  value,
  onChange,
  label,
  disabledOptions = [],
}: {
  options: string[];
  value: string;
  onChange: (value: string) => void;
  label?: string;
  disabledOptions?: string[];
}) {
  const enabledOptions = options.filter((option) => !disabledOptions.includes(option));
  const moveSelection = (direction: 1 | -1) => {
    const current = Math.max(enabledOptions.indexOf(value), 0);
    const next =
      enabledOptions[(current + direction + enabledOptions.length) % enabledOptions.length];
    if (next) onChange(next);
  };
  return (
    <div
      className="inline-flex gap-1 rounded-[var(--radius-md)] bg-bg-code p-1"
      role="radiogroup"
      aria-label={label}
      onKeyDown={(event) => {
        if (event.key === "ArrowRight" || event.key === "ArrowDown") {
          event.preventDefault();
          moveSelection(1);
        }
        if (event.key === "ArrowLeft" || event.key === "ArrowUp") {
          event.preventDefault();
          moveSelection(-1);
        }
      }}
    >
      {options.map((option) => {
        const checked = option === value;
        const disabled = disabledOptions.includes(option);
        return (
          <button
            key={option}
            type="button"
            role="radio"
            aria-checked={checked}
            disabled={disabled}
            className={cn(
              "rounded-[var(--radius-sm)] px-3 py-1 text-[13px] font-medium text-text-secondary transition duration-[var(--dur-instant)] hover:text-text-primary focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-border-focus disabled:cursor-not-allowed disabled:opacity-50",
              checked && "bg-accent-primary-mu text-accent-primary hover:text-accent-primary",
            )}
            onClick={() => onChange(option)}
          >
            {option}
          </button>
        );
      })}
    </div>
  );
}

export function Toolbar({
  label,
  children,
  className,
}: PropsWithChildren<{ label: string; className?: string }>) {
  return (
    <div
      role="toolbar"
      aria-label={label}
      className={cn(
        "flex flex-wrap items-center gap-2 rounded-[var(--radius-md)] border border-border-default bg-bg-code p-2",
        className,
      )}
    >
      {children}
    </div>
  );
}

export function IdentifierPill({
  value,
  label = "Identifier",
  disabled = false,
  iconOnly = false,
}: {
  value: string;
  label?: string;
  disabled?: boolean;
  iconOnly?: boolean;
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
  if (iconOnly) {
    return (
      <IconButton
        label={`Copy ${label}: ${value}`}
        icon={<Icons.Copy className="h-3.5 w-3.5" />}
        variant="subtle"
        size="sm"
        onClick={copy}
        disabled={disabled}
      />
    );
  }
  return (
    <span
      className={cn(
        "inline-flex h-7 items-center gap-1.5 rounded-[var(--radius-sm)] bg-bg-code pl-2 pr-1 font-mono text-[13px] text-text-identifier",
        copied && "bg-accent-primary-mu",
        disabled && "opacity-50",
      )}
      aria-label={`${label}: ${value}`}
    >
      <span>{middleEllipsis(value)}</span>
      {copied ? (
        <span aria-live="polite" className="text-[12px] text-text-primary">
          Copied
        </span>
      ) : null}
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
    <span className="relative inline-flex h-8 items-center gap-2 rounded-[var(--radius-sm)] bg-bg-code py-2 pl-2.5 pr-1 font-mono text-[13px] text-text-identifier">
      <span>
        {revealed ? (
          value
        ) : (
          <>
            <span className="text-text-secondary">{name}_</span>
            <span className="text-secret-mask">••••••••••••</span>
          </>
        )}
      </span>
      {message && !revealed ? (
        <span
          role="status"
          aria-live="polite"
          className="pointer-events-none absolute -top-2 right-2 -translate-y-full rounded-[var(--radius-sm)] border border-status-error bg-bg-elevated px-2.5 py-1.5 text-[12px] font-medium text-text-primary shadow-[var(--shadow-md)]"
        >
          {message}
        </span>
      ) : null}
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
      {revealed && message ? (
        <span className="sr-only" aria-live="polite">
          {message}
        </span>
      ) : null}
    </span>
  );
}

interface TenantSwitcherTenant {
  name: string;
  slug: string;
}

type TenantSwitcherState = "default" | "loading" | "empty" | "error";

export function TenantSwitcher({
  label = "Acme Operations",
  slug = "acme",
  tenants,
  state,
  errorMessage,
  hasError = false,
  onCreateTenant,
  onSelect,
  onRetry,
}: {
  label?: string;
  slug?: string;
  tenants?: TenantSwitcherTenant[];
  state?: TenantSwitcherState;
  errorMessage?: string;
  hasError?: boolean;
  onCreateTenant?: () => void;
  onSelect?: (tenant: TenantSwitcherTenant) => void;
  onRetry?: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const panelId = useId();
  const tenantsQuery = useQuery({
    queryKey: ["tenants"],
    queryFn: listTenants,
    enabled: tenants === undefined && (open || slug !== ""),
    retry: false,
  });
  const fetchedList: TenantSwitcherTenant[] | undefined = tenantsQuery.data?.map((tenant) => ({
    name: tenant.branding?.display_name ?? tenant.name,
    slug: tenant.slug,
  }));
  const list = tenants ?? fetchedList ?? [];
  const resolvedState: TenantSwitcherState =
    state ??
    (tenants === undefined && tenantsQuery.isLoading
      ? "loading"
      : tenants === undefined && tenantsQuery.isError
        ? "error"
        : list.length === 0
          ? "empty"
          : "default");
  const filtered = query
    ? list.filter(
        (tenant) =>
          tenant.name.toLowerCase().includes(query.toLowerCase()) ||
          tenant.slug.toLowerCase().includes(query.toLowerCase()),
      )
    : list;
  const showSearch = list.length >= 10;
  return (
    <span className="relative inline-block w-full">
      <button
        type="button"
        aria-expanded={open}
        aria-controls={panelId}
        onClick={() => setOpen((current) => !current)}
        onKeyDown={(event) => {
          if (event.key === "Escape") setOpen(false);
          if (event.key === "ArrowDown") setOpen(true);
        }}
        className={cn(
          "flex h-10 w-full items-center gap-2 rounded-[var(--radius-md)] bg-bg-code px-3 text-[13px] transition duration-[var(--dur-instant)]",
          "hover:border-border-emphasis hover:bg-bg-elevated",
          "focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-border-focus",
          hasError
            ? "border-l-[3px] border-y border-r border-status-error"
            : "border border-border-default",
        )}
      >
        <Icons.SquareDot
          className="h-3.5 w-3.5 flex-shrink-0 text-accent-primary"
          aria-hidden="true"
        />
        <span className="min-w-0 flex-1 truncate text-left font-medium text-text-primary">
          {list.find((tenant) => tenant.slug === slug)?.name ?? label}
        </span>
        <Icons.ChevronsUpDown
          className="h-3.5 w-3.5 flex-shrink-0 text-text-tertiary"
          aria-hidden="true"
        />
      </button>
      {open ? (
        <div
          id={panelId}
          className="absolute left-0 z-20 mt-2 w-[280px] rounded-[var(--radius-md)] border border-border-default bg-bg-elevated p-1.5 shadow-[var(--shadow-lg)]"
          onKeyDown={(event) => {
            if (event.key === "Escape") setOpen(false);
          }}
        >
          {resolvedState === "error" ? (
            <div className="grid gap-2 px-2 py-3">
              <p className="text-[13px] text-status-error">
                {errorMessage ?? "Couldn't load tenants."}
              </p>
              <Button
                variant="secondary"
                size="sm"
                onClick={() => {
                  if (onRetry) {
                    onRetry();
                  } else {
                    void tenantsQuery.refetch();
                  }
                }}
              >
                Retry
              </Button>
            </div>
          ) : resolvedState === "loading" ? (
            <div className="grid gap-2 px-2 py-3">
              <Skeleton shape="line" className="h-3.5 w-full" />
              <Skeleton shape="line" className="h-3.5 w-3/4" />
              <Skeleton shape="line" className="h-3.5 w-2/3" />
            </div>
          ) : resolvedState === "empty" || filtered.length === 0 ? (
            <div className="grid place-items-center gap-1.5 px-4 py-6 text-center">
              <p className="text-[13px] font-medium text-text-primary">No tenants</p>
              <p className="text-[12px] text-text-secondary">
                {query ? "No matches for your search." : "Create your first tenant to get started."}
              </p>
            </div>
          ) : (
            <>
              {showSearch ? (
                <div className="px-1 pb-1.5">
                  <TextInput
                    label="Search tenants"
                    placeholder="acme"
                    size="sm"
                    value={query}
                    onChange={(event) => setQuery(event.target.value)}
                  />
                </div>
              ) : null}
              <ul role="listbox" className="grid gap-0.5">
                {filtered.map((tenant) => (
                  <li key={tenant.slug}>
                    <button
                      type="button"
                      role="option"
                      aria-selected={tenant.slug === slug}
                      onClick={() => {
                        if (onSelect) {
                          onSelect(tenant);
                        } else {
                          window.location.assign(`/dashboard/tenants/${tenant.slug}`);
                        }
                        setOpen(false);
                      }}
                      className={cn(
                        "flex w-full items-center gap-2 rounded-[var(--radius-sm)] px-2 py-1.5 text-left transition duration-[var(--dur-instant)] hover:bg-accent-primary-mu hover:text-accent-primary focus-visible:bg-accent-primary-mu focus-visible:outline-none",
                        tenant.slug === slug && "bg-accent-primary-mu text-accent-primary",
                      )}
                    >
                      <TenantGlyph className="h-3.5 w-3.5 flex-shrink-0" />
                      <span className="min-w-0 flex-1 truncate text-[13px] font-medium">
                        {tenant.name}
                      </span>
                      {tenant.slug === slug ? (
                        <svg
                          viewBox="0 0 12 12"
                          className="h-3 w-3 flex-shrink-0"
                          aria-hidden="true"
                          fill="none"
                          stroke="currentColor"
                          strokeWidth="2"
                          strokeLinecap="round"
                          strokeLinejoin="round"
                        >
                          <polyline points="2.5 6.5 5 9 9.5 3.5" />
                        </svg>
                      ) : null}
                    </button>
                  </li>
                ))}
              </ul>
            </>
          )}
          {onCreateTenant ? (
            <>
              <div className="my-1.5 h-px bg-border-subtle" />
              <Button size="sm" variant="ghost" className="w-full" onClick={onCreateTenant}>
                Create tenant
              </Button>
            </>
          ) : null}
        </div>
      ) : null}
    </span>
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
        "inline-flex h-6 items-center gap-1.5 rounded-[var(--radius-pill)] border bg-bg-elevated px-2.5 text-[11px] font-semibold uppercase tracking-[0.06em]",
        kind === "instance"
          ? "border-context-instance text-context-instance"
          : "border-context-tenant text-context-tenant",
      )}
    >
      <Glyph className="h-3.5 w-3.5" />
      {label ?? (kind === "instance" ? "Instance admin" : "Tenant: acme")}
    </span>
  );
}

export function StatusPip({ variant, label }: { variant: StatusVariant; label: string }) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 text-[13px] font-medium",
        statusClass[variant],
      )}
    >
      <StatusSigil variant={variant} />
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

interface KeyRotationSegment {
  kid: string;
  state: "sunsetting" | "overlap" | "active";
  width: number;
}

export function KeyRotationTimeline({
  keyID = "kid_active_2026_05",
  segments,
  nextRotationDays = 28,
  onCreateKey,
}: {
  keyID?: string;
  segments?: KeyRotationSegment[];
  nextRotationDays?: number;
  onCreateKey?: () => void;
}) {
  if (keyID === "no-active-key") {
    return (
      <div className="grid place-items-center gap-3 rounded-[var(--radius-md)] border border-border-subtle bg-bg-surface p-6 text-center">
        <svg
          viewBox="0 0 24 24"
          className="h-6 w-6 text-status-error"
          aria-hidden="true"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.5"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <circle cx="8" cy="15" r="3" />
          <path d="m10.5 12.5 7-7m-3 1 2.5 2.5M12 15h.01" />
        </svg>
        <h3 className="text-[20px] font-semibold leading-[1.4] text-text-primary">
          No active signing key
        </h3>
        <p className="text-[13px] text-text-secondary">
          Cypra cannot mint OIDC tokens for this tenant.
        </p>
        {onCreateKey ? (
          <Button variant="primary" size="sm" onClick={onCreateKey}>
            Create signing key
          </Button>
        ) : null}
      </div>
    );
  }

  const trackSegments: KeyRotationSegment[] = segments ?? [
    { kid: "k_42", state: "sunsetting", width: 30 },
    { kid: "k_43", state: "overlap", width: 20 },
    { kid: keyID.split("_").pop() ?? "k_44", state: "active", width: 50 },
  ];

  return (
    <div className="grid gap-3 rounded-[var(--radius-md)] border border-border-subtle bg-bg-surface p-4">
      <div className="flex h-6 overflow-hidden rounded-[var(--radius-sm)]">
        {trackSegments.map((segment, index) => (
          <span
            key={`${segment.kid}-${String(index)}`}
            className={cn(
              "flex items-center justify-center gap-1.5",
              segment.state === "sunsetting" && "bg-key-state-sunsetting",
              segment.state === "overlap" && "bg-key-state-overlap",
              segment.state === "active" && "bg-key-state-active",
            )}
            style={{ width: `${String(segment.width)}%` }}
          >
            {segment.state === "sunsetting" ? (
              <span className="h-2 w-2 rounded-full border-[1.5px] border-bg-surface" />
            ) : null}
            {segment.state === "overlap" ? (
              <span
                className="h-0 w-0 border-x-[4.5px] border-b-[8px] border-x-transparent border-b-bg-surface"
                aria-hidden="true"
              />
            ) : null}
            {segment.state === "active" ? (
              <>
                <span className="h-2 w-2 rounded-full bg-bg-surface" />
                <span className="text-[11px] font-semibold text-bg-surface">current</span>
              </>
            ) : null}
          </span>
        ))}
      </div>
      <div className="flex" aria-hidden="true">
        {trackSegments.map((segment, index) => (
          <span
            key={`${segment.kid}-label-${String(index)}`}
            className="text-center font-mono text-[11px] text-text-tertiary"
            style={{ width: `${String(segment.width)}%` }}
          >
            {segment.kid}
          </span>
        ))}
      </div>
      <div className="flex items-center justify-between">
        <IdentifierPill value={keyID} label="Key ID" />
        <span className="text-[12px] text-text-secondary">
          Next rotation in {nextRotationDays} days
        </span>
      </div>
    </div>
  );
}

export function AuditEntry({
  action,
  resource,
  actor = "Megan Patel",
  actorId = "00000000-0000-0000-0000-00000000cafe",
  timestamp = "2m ago",
  expandable = false,
  details,
  redacted = false,
}: {
  action: string;
  resource: string;
  actor?: string;
  actorId?: string;
  timestamp?: string;
  expandable?: boolean;
  details?: ReactNode;
  redacted?: boolean;
}) {
  const [expanded, setExpanded] = useState(false);
  const Header = (
    <div className="flex items-center gap-4 px-4 py-3 text-[13px]">
      <time className="w-20 font-mono text-text-secondary">{timestamp}</time>
      <span className="flex w-48 items-center gap-2">
        <Avatar name={actor} sub={actorId} size="sm" />
        <span className="font-medium text-text-primary">{actor}</span>
      </span>
      <span
        className={cn(
          "rounded-[var(--radius-sm)] px-2 py-0.5 text-[11px] font-semibold tracking-[0.06em]",
          redacted ? "bg-bg-code text-text-tertiary" : "bg-accent-primary-mu text-accent-primary",
        )}
      >
        {redacted ? "redacted" : action}
      </span>
      <span className="flex-1" />
      {redacted ? (
        <span className="text-[12px] text-text-tertiary">
          <Icons.Lock className="mr-1 inline h-3.5 w-3.5" aria-hidden="true" />
          access logs scrubbed
        </span>
      ) : (
        <IdentifierPill value={resource} label="Resource ID" />
      )}
      {expandable && !redacted ? (
        <IconButton
          label={expanded ? "Collapse details" : "Expand details"}
          icon={
            <Icons.ChevronDown
              className={cn("h-3.5 w-3.5 transition-transform", expanded && "rotate-180")}
            />
          }
          variant="ghost"
          size="sm"
          onClick={() => setExpanded((current) => !current)}
        />
      ) : null}
    </div>
  );
  return (
    <article className="rounded-[var(--radius-md)] border border-border-subtle bg-bg-surface transition duration-[var(--dur-instant)] hover:border-border-default hover:bg-bg-elevated">
      {Header}
      {expandable && expanded && details ? (
        <div className="border-t border-border-subtle px-4 py-3 text-[13px] text-text-secondary">
          {details}
        </div>
      ) : null}
    </article>
  );
}

export function BackupCodeGrid({
  codes,
  onConfirmedChange,
}: {
  codes: string[];
  onConfirmedChange?: (confirmed: boolean) => void;
}) {
  const [confirmed, setConfirmed] = useState(false);
  const confirmSaved = () => {
    setConfirmed(true);
    onConfirmedChange?.(true);
  };
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
  useEffect(() => {
    if (confirmed) return undefined;
    const currentURL = window.location.href;
    const guardState = { cypraBackupCodeGuard: true };
    history.pushState(guardState, "", currentURL);
    const warnNavigation = () => {
      if (window.confirm("Your backup codes are shown only once. Continue without saving?")) {
        return;
      }
      history.pushState(guardState, "", currentURL);
    };
    window.addEventListener("popstate", warnNavigation);
    return () => window.removeEventListener("popstate", warnNavigation);
  }, [confirmed]);
  const downloadCodes = () => {
    const blob = new Blob([codes.join("\n")], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = "cypra-backup-codes.txt";
    document.body.appendChild(anchor);
    anchor.click();
    anchor.remove();
    URL.revokeObjectURL(url);
  };
  const copyAll = () => {
    void navigator.clipboard.writeText(codes.join("\n"));
  };
  const copyOne = (code: string) => {
    void navigator.clipboard.writeText(code);
  };
  if (confirmed) {
    return (
      <div className="flex items-center gap-3 rounded-[var(--radius-md)] border border-accent-primary-mu bg-bg-surface px-5 py-4">
        <Icons.CheckCircle2 className="h-5 w-5 text-status-success" aria-hidden="true" />
        <span className="text-[13px] font-medium text-text-primary">
          Backup codes saved. You can leave this screen safely.
        </span>
      </div>
    );
  }
  return (
    <div className="grid gap-4 rounded-[var(--radius-md)] border border-border-subtle bg-bg-surface p-6">
      <div className="grid grid-cols-2 gap-3">
        {codes.map((code) => (
          <button
            key={code}
            type="button"
            onClick={() => copyOne(code)}
            className="flex items-center justify-between rounded-[var(--radius-sm)] bg-bg-code px-2.5 py-2 transition duration-[var(--dur-instant)] hover:bg-bg-elevated focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-border-focus"
          >
            <code className="font-mono text-[13px] text-text-identifier">{code}</code>
            <Icons.Copy className="h-3.5 w-3.5 text-text-secondary" aria-hidden="true" />
          </button>
        ))}
      </div>
      <div className="flex items-center gap-2">
        <Button
          variant="secondary"
          size="sm"
          leading={<Icons.Copy className="h-3.5 w-3.5" />}
          onClick={copyAll}
        >
          Copy all
        </Button>
        <Button
          variant="secondary"
          size="sm"
          leading={
            <svg
              viewBox="0 0 14 14"
              className="h-3.5 w-3.5"
              aria-hidden="true"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.5"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="M7 1.5v8.5M3.5 6.5L7 10l3.5-3.5M2 12h10" />
            </svg>
          }
          onClick={downloadCodes}
        >
          Download .txt
        </Button>
        <span className="flex-1" />
        <Button variant="primary" onClick={confirmSaved}>
          I have saved these
        </Button>
      </div>
    </div>
  );
}

type PermissionCellState = "checked" | "unchecked" | "disabled";

function PermissionCell({ state }: { state: PermissionCellState }) {
  if (state === "checked") {
    return (
      <span className="grid h-[18px] w-[18px] place-items-center rounded-[var(--radius-sm)] bg-accent-primary">
        <svg
          viewBox="0 0 12 12"
          className="h-3 w-3 text-text-on-accent"
          aria-hidden="true"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <polyline points="2.5 6.5 5 9 9.5 3.5" />
        </svg>
      </span>
    );
  }
  if (state === "disabled") {
    return (
      <span className="block h-[18px] w-[18px] rounded-[var(--radius-sm)] border border-border-subtle bg-bg-code" />
    );
  }
  return (
    <span className="block h-[18px] w-[18px] rounded-[var(--radius-sm)] border-[1.5px] border-border-emphasis" />
  );
}

interface PermissionMatrixRow {
  role: string;
  cells: PermissionCellState[];
}

type PermissionMatrixSaveState = "clean" | "dirty" | "saving" | "saved" | "error";

const defaultPermissions = ["Read", "Write", "Invite", "Rotate", "Billing"];
const defaultRows: PermissionMatrixRow[] = [
  { role: "Owner", cells: ["checked", "checked", "checked", "checked", "checked"] },
  { role: "Admin", cells: ["checked", "checked", "checked", "checked", "disabled"] },
  { role: "Member", cells: ["checked", "unchecked", "unchecked", "unchecked", "disabled"] },
  { role: "Viewer", cells: ["checked", "unchecked", "unchecked", "unchecked", "disabled"] },
];

export function PermissionMatrix({
  permissions = defaultPermissions,
  rows = defaultRows,
  dirtyCount = 0,
  saveState,
  saveMessage,
  onSave,
  onDiscard,
  onRetry,
  saving = false,
  readonly = false,
  denied = false,
}: {
  permissions?: string[];
  rows?: PermissionMatrixRow[];
  dirtyCount?: number;
  saveState?: PermissionMatrixSaveState;
  saveMessage?: string;
  onSave?: () => void | Promise<void>;
  onDiscard?: () => void;
  onRetry?: () => void;
  saving?: boolean;
  readonly?: boolean;
  denied?: boolean;
} = {}) {
  if (denied) {
    return (
      <div className="grid place-items-center gap-3 rounded-[var(--radius-md)] border border-border-subtle bg-bg-surface p-12 text-center">
        <Icons.Lock className="h-6 w-6 text-text-tertiary" aria-hidden="true" />
        <h2 className="text-[20px] font-semibold leading-[1.4] text-text-primary">
          403 — permission denied
        </h2>
        <p className="text-[13px] text-text-secondary">
          You need tenant admin to view permissions.
        </p>
      </div>
    );
  }

  const resolvedState: PermissionMatrixSaveState =
    saveState ?? (saving ? "saving" : dirtyCount > 0 ? "dirty" : "clean");

  return (
    <div className="grid gap-4">
      {readonly ? (
        <div className="flex items-center gap-3 rounded-[var(--radius-md)] border border-status-info bg-bg-surface px-4 py-3">
          <svg
            viewBox="0 0 16 16"
            className="h-4 w-4 text-status-info"
            aria-hidden="true"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.5"
          >
            <circle cx="8" cy="8" r="6.5" />
            <path d="M8 7v4.5M8 4.5v.5" strokeLinecap="round" />
          </svg>
          <span className="text-[13px] text-text-primary">
            You can view this matrix but not edit it.
          </span>
        </div>
      ) : null}

      <div className="overflow-hidden rounded-[var(--radius-md)] border border-border-default bg-bg-surface">
        <div
          className="grid items-center bg-bg-code px-4 py-3"
          style={{ gridTemplateColumns: `160px repeat(${String(permissions.length)}, 120px)` }}
        >
          <span className="text-[11px] font-semibold uppercase tracking-[0.06em] text-text-tertiary">
            Role
          </span>
          {permissions.map((permission) => (
            <span
              key={permission}
              className="text-center text-[11px] font-semibold uppercase tracking-[0.06em] text-text-tertiary"
            >
              {permission}
            </span>
          ))}
        </div>
        {rows.map((row) => (
          <div
            key={row.role}
            className="grid items-center border-t border-border-subtle px-4 py-3"
            style={{ gridTemplateColumns: `160px repeat(${String(permissions.length)}, 120px)` }}
          >
            <span className="text-[13px] font-medium text-text-primary">{row.role}</span>
            {row.cells.map((state, index) => (
              <span key={`${row.role}-${String(index)}`} className="grid place-items-center">
                <PermissionCell state={state} />
              </span>
            ))}
          </div>
        ))}
      </div>

      {!readonly && (resolvedState !== "clean" || onSave || onDiscard) ? (
        <PermissionMatrixSaveBar
          state={resolvedState}
          dirtyCount={dirtyCount}
          message={saveMessage}
          onSave={onSave}
          onDiscard={onDiscard}
          onRetry={onRetry}
        />
      ) : null}
    </div>
  );
}

function PermissionMatrixSaveBar({
  state,
  dirtyCount,
  message,
  onSave,
  onDiscard,
  onRetry,
}: {
  state: PermissionMatrixSaveState;
  dirtyCount: number;
  message?: string;
  onSave?: () => void | Promise<void>;
  onDiscard?: () => void;
  onRetry?: () => void;
}) {
  if (state === "saved") {
    return (
      <div className="flex items-center gap-3 rounded-[var(--radius-md)] border border-accent-primary-mu bg-bg-elevated px-4 py-3">
        <Icons.CheckCircle2 className="h-3.5 w-3.5 text-status-success" aria-hidden="true" />
        <span className="text-[13px] font-medium text-text-primary">
          {message ?? "All changes saved"}
        </span>
      </div>
    );
  }
  if (state === "error") {
    return (
      <div className="flex items-center gap-4 rounded-[var(--radius-md)] border-l-[3px] border-status-error bg-bg-elevated px-4 py-3">
        <Icons.XCircle className="h-3.5 w-3.5 text-status-error" aria-hidden="true" />
        <span className="flex-1 text-[13px] font-medium text-text-primary">
          {message ?? "Save failed — your draft is preserved."}
        </span>
        {onRetry ? (
          <Button variant="destructive" size="sm" onClick={onRetry}>
            Retry
          </Button>
        ) : null}
      </div>
    );
  }
  return (
    <div className="flex items-center gap-4 rounded-[var(--radius-md)] border border-border-emphasis bg-bg-elevated px-4 py-3 shadow-[var(--shadow-lg)]">
      <span className="h-3.5 w-3.5 rounded-full bg-status-pending" aria-hidden="true" />
      <span className="flex-1 text-[13px] font-medium text-text-primary">
        {state === "saving"
          ? "Saving…"
          : `${String(dirtyCount)} ${dirtyCount === 1 ? "row" : "rows"} changed`}
      </span>
      <Button variant="secondary" size="sm" onClick={onDiscard} disabled={state === "saving"}>
        Discard
      </Button>
      <Button
        variant="primary"
        size="sm"
        loading={state === "saving"}
        onClick={() => void onSave?.()}
      >
        Save
      </Button>
    </div>
  );
}

export function SetupTokenBanner({ token }: { token: string }) {
  return (
    <div className="grid gap-4 rounded-[var(--radius-md)] border border-border-emphasis bg-bg-surface p-6">
      <header className="flex items-center gap-2">
        <Icons.Lock className="h-4 w-4 text-accent-primary" aria-hidden="true" />
        <h3 className="text-[13px] font-semibold text-text-primary">Bootstrap token</h3>
      </header>
      <div
        className="flex items-center gap-3 rounded-[var(--radius-md)] bg-bg-code px-5 py-3.5"
        aria-label={`Setup token: ${token}`}
      >
        <code className="flex-1 truncate font-mono text-[14px] text-text-identifier">{token}</code>
        <IconButton
          label="Copy bootstrap token"
          icon={<Icons.Copy className="h-3.5 w-3.5" />}
          variant="subtle"
          size="sm"
          onClick={() => void navigator.clipboard.writeText(token)}
        />
      </div>
      <p className="flex items-start gap-2 text-[12px] leading-[1.5] text-status-warn">
        <Icons.AlertTriangle className="mt-0.5 h-3.5 w-3.5 flex-shrink-0" aria-hidden="true" />
        <span>
          Redacted from exports and audit logs. Shown only on this screen — copy it now or restart
          setup.
        </span>
      </p>
      <div className="flex justify-end">
        <Button variant="primary">I have copied this — show next step</Button>
      </div>
    </div>
  );
}

type ProviderConfigCardState =
  | "configured-healthy"
  | "unconfigured"
  | "configured-failing"
  | "testing"
  | "testing-while-dirty";

export function ProviderConfigCard({
  kind,
  name,
  status,
  statusLabel,
  state = "configured-healthy",
  errorMessage,
  onConfigure,
  actionLabel,
  children,
}: PropsWithChildren<{
  kind: "email" | "upstream" | "storage";
  name?: string;
  status?: StatusVariant;
  statusLabel?: string;
  state?: ProviderConfigCardState;
  errorMessage?: string;
  onConfigure?: () => void;
  actionLabel?: string;
}>) {
  const Glyph =
    kind === "email" ? Icons.AlertTriangle : kind === "upstream" ? Icons.Lock : Icons.Monitor;
  const heading = name ?? `${kind.charAt(0).toUpperCase()}${kind.slice(1)} provider`;
  const { resolvedStatus, resolvedLabel, resolvedAction } = (() => {
    if (status && statusLabel) {
      return {
        resolvedStatus: status,
        resolvedLabel: statusLabel,
        resolvedAction: actionLabel ?? "Configure",
      };
    }
    switch (state) {
      case "unconfigured":
        return {
          resolvedStatus: "warn" as StatusVariant,
          resolvedLabel: "Required",
          resolvedAction: actionLabel ?? "Configure",
        };
      case "configured-failing":
        return {
          resolvedStatus: "error" as StatusVariant,
          resolvedLabel: "Failing",
          resolvedAction: actionLabel ?? "Investigate",
        };
      case "testing":
        return {
          resolvedStatus: "pending" as StatusVariant,
          resolvedLabel: "Testing",
          resolvedAction: actionLabel ?? "View test",
        };
      case "testing-while-dirty":
        return {
          resolvedStatus: "pending" as StatusVariant,
          resolvedLabel: "Testing · unsaved",
          resolvedAction: actionLabel ?? "View test",
        };
      default:
        return {
          resolvedStatus: "active" as StatusVariant,
          resolvedLabel: "Active",
          resolvedAction: actionLabel ?? "Configure",
        };
    }
  })();
  const isFailing = state === "configured-failing";
  return (
    <div
      className={cn(
        "grid gap-4 rounded-[var(--radius-md)] bg-bg-surface p-5",
        isFailing
          ? "border-l-[3px] border-y border-r border-status-error"
          : "border border-border-default",
      )}
    >
      <header className="flex items-center gap-3">
        <Glyph className="h-5 w-5 text-text-primary" aria-hidden="true" />
        <span className="text-[13px] font-semibold text-text-primary">{heading}</span>
        <span className="flex-1" />
        <StatusPip variant={resolvedStatus} label={resolvedLabel} />
      </header>
      {isFailing && errorMessage ? (
        <p className="text-[13px] text-status-error">{errorMessage}</p>
      ) : null}
      {children ? <div className="grid gap-1.5 text-[13px]">{children}</div> : null}
      {onConfigure ? (
        <div className="flex justify-end">
          <Button variant="secondary" onClick={onConfigure}>
            {resolvedAction}
          </Button>
        </div>
      ) : null}
    </div>
  );
}

export function MobileBlockedBanner() {
  const [dismissed, setDismissed] = useState(false);
  if (dismissed) return null;
  return (
    <div className="flex items-center gap-3 rounded-[var(--radius-md)] border border-border-subtle bg-bg-surface px-3 py-4 text-[13px] leading-[1.5] text-text-secondary md:hidden">
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

export function CodeBlock({
  code,
  language = "shell",
  curlToggle = false,
}: {
  code: string;
  language?: string;
  curlToggle?: boolean;
}) {
  const [curl, setCurl] = useState(false);
  const shown = curlToggle && curl ? `curl ${code}` : code;
  const copy = () => {
    void navigator.clipboard.writeText(shown);
  };
  return (
    <div className="overflow-hidden rounded-[var(--radius-md)] border border-border-subtle bg-bg-code">
      <div className="flex items-center justify-between border-b border-border-subtle px-4 py-2.5">
        <span className="text-[12px] text-text-tertiary">{language}</span>
        <div className="flex items-center gap-1.5">
          {curlToggle ? (
            <button
              type="button"
              aria-pressed={curl}
              onClick={() => setCurl((current) => !current)}
              className={cn(
                "rounded-[var(--radius-sm)] border border-border-default bg-bg-surface px-2 py-1 text-[12px] font-medium",
                curl ? "text-accent-primary" : "text-text-secondary",
              )}
            >
              cURL
            </button>
          ) : null}
          <IconButton
            label="Copy code"
            icon={<Icons.Copy className="h-3.5 w-3.5" />}
            size="sm"
            onClick={copy}
          />
        </div>
      </div>
      <pre
        className="overflow-x-auto p-4 font-mono text-[13px] leading-[1.538] text-text-identifier"
        tabIndex={0}
      >
        <code>{shown}</code>
      </pre>
    </div>
  );
}

export function PageHeader({
  title,
  subtitle,
  action,
  breadcrumb,
  loading = false,
}: {
  title: string;
  subtitle?: string;
  action?: ReactNode;
  breadcrumb?: ReactNode;
  loading?: boolean;
}) {
  return (
    <header className="mb-6 flex flex-col gap-4 rounded-[var(--radius-md)] border border-border-subtle bg-bg-surface px-8 py-6">
      {breadcrumb}
      <div className="flex items-start justify-between gap-4">
        {loading ? (
          <div className="grid w-80 gap-2">
            <Skeleton shape="line" />
            <Skeleton shape="line" className="w-2/3" />
          </div>
        ) : (
          <div className="grid gap-1">
            <h1 className="text-[24px] font-semibold leading-[1.333]">{title}</h1>
            {subtitle ? (
              <p className="max-w-2xl text-[14px] leading-[1.571] text-text-secondary">
                {subtitle}
              </p>
            ) : null}
          </div>
        )}
        {action}
      </div>
    </header>
  );
}

export function Breadcrumb({ segments }: { segments: string[] }) {
  return (
    <nav
      aria-label="Breadcrumb"
      className="flex items-center gap-1.5 text-[13px] text-text-secondary"
    >
      {segments.map((segment, index) => (
        <span key={segment} className="flex items-center gap-1.5">
          {index > 0 ? <span className="text-text-tertiary">/</span> : null}
          {index === segments.length - 1 ? (
            <span className="font-medium text-text-primary">{segment}</span>
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
  icon,
}: {
  title: string;
  body?: string;
  action?: ReactNode;
  icon?: ReactNode;
}) {
  return (
    <div className="grid place-items-center gap-3 rounded-[var(--radius-md)] border border-border-subtle bg-bg-surface p-12 text-center">
      <span className="text-text-tertiary">
        {icon ?? <Icons.AlertTriangle className="h-6 w-6" />}
      </span>
      <h2 className="text-[20px] font-semibold leading-[1.4]">{title}</h2>
      {body ? (
        <p className="max-w-[360px] text-[14px] leading-[1.571] text-text-secondary">{body}</p>
      ) : null}
      {action ? <div className="mt-1">{action}</div> : null}
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
    <div className="grid place-items-center gap-3 rounded-[var(--radius-md)] border border-border-subtle bg-bg-surface p-12 text-center">
      <Icons.XCircle className="h-6 w-6 text-status-error" />
      <h2 className="text-[20px] font-semibold leading-[1.4]">{title}</h2>
      <p className="max-w-[420px] text-[14px] leading-[1.571] text-text-secondary">{body}</p>
      {retry ? (
        <Button className="mt-1" onClick={retry}>
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

export interface ConfirmationDialogProps {
  open: boolean;
  variant?: "destructive" | "warn";
  headline: string;
  body?: ReactNode;
  resourceMatch?: string;
  confirmLabel?: string;
  loading?: boolean;
  errorMessage?: string;
  onConfirm: () => void | Promise<void>;
  onCancel: () => void;
}

export function ConfirmationDialog({
  open,
  variant = "destructive",
  headline,
  body,
  resourceMatch,
  confirmLabel,
  loading = false,
  errorMessage,
  onConfirm,
  onCancel,
}: ConfirmationDialogProps) {
  const [value, setValue] = useState("");
  useEffect(() => {
    if (!open) setValue("");
  }, [open]);
  if (!open) return null;
  const matchOk = !resourceMatch || value.trim() === resourceMatch;
  const label = confirmLabel ?? (variant === "destructive" ? "Delete" : "Confirm");
  const headerIcon =
    variant === "destructive" ? (
      <Icons.XCircle className="h-6 w-6 text-status-error" aria-hidden="true" />
    ) : (
      <Icons.AlertTriangle className="h-6 w-6 text-status-warn" aria-hidden="true" />
    );
  return (
    <Modal
      title={headline}
      headerIcon={headerIcon}
      open={open}
      onClose={onCancel}
      size="sm"
      footer={
        <>
          <Button variant="ghost" onClick={onCancel} disabled={loading}>
            Cancel
          </Button>
          <Button
            variant={variant === "destructive" ? "destructive" : "primary"}
            disabled={!matchOk || loading}
            loading={loading}
            onClick={() => void onConfirm()}
          >
            {label}
          </Button>
        </>
      }
    >
      {body ? <div className="text-[14px] text-text-secondary">{body}</div> : null}
      {resourceMatch ? (
        <div className="mt-4 grid gap-1.5">
          <TextInput
            label={`Type ${resourceMatch} to confirm`}
            value={value}
            onChange={(event) => setValue(event.target.value)}
            state={matchOk && value.length > 0 ? "success" : "default"}
          />
          {matchOk && value.length > 0 ? (
            <p className="text-[13px] text-status-success">
              Match · case-sensitive · whitespace trimmed.
            </p>
          ) : null}
        </div>
      ) : null}
      {errorMessage ? (
        <div
          role="alert"
          className="mt-4 flex items-start gap-2 rounded-[var(--radius-md)] border border-status-error bg-bg-canvas p-3"
        >
          <Icons.AlertTriangle
            className="h-4 w-4 flex-shrink-0 text-status-error"
            aria-hidden="true"
          />
          <p className="text-[13px] leading-[1.538] text-text-secondary">
            <span className="font-medium text-text-primary">{errorMessage}</span>
          </p>
        </div>
      ) : null}
    </Modal>
  );
}

export function SettingsRow({
  label,
  helper,
  control,
  flush = false,
}: {
  label: string;
  helper: string;
  control: ReactNode;
  flush?: boolean;
}) {
  return (
    <div
      className={cn(
        "flex items-center justify-between gap-6 border-b border-border-subtle py-5",
        flush ? "px-0" : "px-6",
      )}
    >
      <div className="grid gap-1">
        <div className="text-[13px] font-medium text-text-primary">{label}</div>
        <p className="text-[13px] leading-[1.538] text-text-secondary">{helper}</p>
      </div>
      {control}
    </div>
  );
}

export function ListRow({
  title,
  meta,
  right,
  onRemove,
  removeLabel = "Remove",
  removeDisabled = false,
  removeTooltip,
}: {
  title: string;
  meta: string;
  right?: ReactNode;
  onRemove?: () => void;
  removeLabel?: string;
  removeDisabled?: boolean;
  removeTooltip?: string;
}) {
  const removeButton = onRemove ? (
    <IconButton
      label={removeLabel}
      icon={<Icons.Trash2 className="h-4 w-4" />}
      variant="destructive"
      disabled={removeDisabled}
      onClick={onRemove}
    />
  ) : null;
  return (
    <div className="flex items-center gap-3 border-b border-border-subtle px-4 py-3">
      <Avatar name={title} sub={meta} size="md" />
      <span className="flex-1 text-[13px] font-medium text-text-primary">{title}</span>
      <span className="text-[13px] text-text-secondary">{meta}</span>
      {right}
      {removeButton && removeTooltip ? (
        <Tooltip label={removeTooltip}>{removeButton}</Tooltip>
      ) : (
        removeButton
      )}
    </div>
  );
}

export function SaveBar({
  dirtyCount,
  onSave,
  onDiscard,
  saving = false,
  state,
  errorTitle = "Couldn't save changes",
  errorBody,
}: {
  dirtyCount: number;
  onSave: () => void | Promise<void>;
  onDiscard: () => void;
  saving?: boolean;
  state?: "dirty" | "saved" | "error";
  errorTitle?: string;
  errorBody?: string;
}) {
  const resolvedState: "dirty" | "saved" | "error" = state ?? (saving ? "dirty" : "dirty");
  const borderClass =
    resolvedState === "saved"
      ? "border-status-success"
      : resolvedState === "error"
        ? "border-status-error"
        : "border-border-emphasis";
  return (
    <div
      className={cn(
        "sticky bottom-4 mt-4 flex items-center justify-between gap-6 rounded-[var(--radius-lg)] border bg-bg-elevated px-6 py-4 shadow-[var(--shadow-lg)]",
        borderClass,
      )}
      role={resolvedState === "error" ? "alert" : undefined}
    >
      {resolvedState === "saved" ? (
        <>
          <span className="flex items-center gap-2 text-[14px] font-medium text-status-success">
            <Icons.CheckCircle2 className="h-4 w-4" aria-hidden="true" />
            Saved
          </span>
          <span className="text-[12px] text-text-tertiary">auto-hides shortly</span>
        </>
      ) : resolvedState === "error" ? (
        <>
          <span className="flex items-center gap-3">
            <Icons.AlertTriangle
              className="h-4 w-4 flex-shrink-0 text-status-error"
              aria-hidden="true"
            />
            <span className="grid gap-0.5">
              <span className="text-[14px] font-medium text-text-primary">{errorTitle}</span>
              {errorBody ? (
                <span className="text-[13px] text-text-secondary">{errorBody}</span>
              ) : null}
            </span>
          </span>
          <span className="flex gap-2">
            <Button variant="ghost" onClick={onDiscard} disabled={saving}>
              Discard
            </Button>
            <Button variant="primary" loading={saving} onClick={() => void onSave()}>
              Try again
            </Button>
          </span>
        </>
      ) : (
        <>
          <span className="flex items-center gap-3 text-[14px] font-medium text-text-primary">
            <span
              className="h-3.5 w-3.5 rounded-full border-2 border-status-warn"
              aria-hidden="true"
            />
            {dirtyCount} unsaved changes
          </span>
          <span className="flex gap-2">
            <Button variant="ghost" onClick={onDiscard} disabled={saving}>
              Discard
            </Button>
            <Button variant="primary" loading={saving} onClick={() => void onSave()}>
              {saving ? "Saving" : "Save"}
            </Button>
          </span>
        </>
      )}
    </div>
  );
}

type SidebarIcon = ComponentType<SVGProps<SVGSVGElement>>;

interface SidebarItem {
  label: string;
  href: string;
  icon: SidebarIcon;
  kind?: "tenant" | "instance";
}

function instanceItems(): SidebarItem[] {
  return [
    { label: "Overview", href: "/dashboard", icon: Icons.LayoutDashboard, kind: "instance" },
    { label: "Tenants", href: "/dashboard/tenants", icon: Icons.Building2, kind: "instance" },
    {
      label: "Instance admins",
      href: "/dashboard/instance/admins",
      icon: Icons.UserCog,
      kind: "instance",
    },
    {
      label: "Instance audit",
      href: "/dashboard/instance/audit",
      icon: Icons.ScrollText,
      kind: "instance",
    },
    {
      label: "Diagnostics",
      href: "/dashboard/instance/diagnostics",
      icon: Icons.Activity,
      kind: "instance",
    },
  ];
}

function tenantItems(slug: string): SidebarItem[] {
  const base = `/dashboard/tenants/${slug}`;
  return [
    { label: "Overview", href: base, icon: Icons.LayoutDashboard, kind: "tenant" },
    { label: "Projects", href: `${base}/projects`, icon: Icons.Folder, kind: "tenant" },
    { label: "Users", href: `${base}/users`, icon: Icons.Users, kind: "tenant" },
    {
      label: "Auth providers",
      href: `${base}/auth-providers`,
      icon: Icons.KeyRound,
      kind: "tenant",
    },
    {
      label: "Signing keys",
      href: `${base}/signing-keys`,
      icon: Icons.KeyRound,
      kind: "tenant",
    },
    { label: "Audit", href: `${base}/audit`, icon: Icons.ScrollText, kind: "tenant" },
    {
      label: "Settings",
      href: `${base}/settings/branding`,
      icon: Icons.Settings,
      kind: "tenant",
    },
  ];
}

function findActiveHref(items: SidebarItem[], pathname: string): string | null {
  let best: string | null = null;
  for (const { href } of items) {
    const matches = pathname === href || pathname.startsWith(`${href}/`);
    if (!matches) continue;
    if (best === null || href.length > best.length) best = href;
  }
  return best;
}

function SidebarNavLink({
  item,
  collapsed,
  active,
}: {
  item: SidebarItem;
  collapsed: boolean;
  active: boolean;
}) {
  const Icon = item.icon;
  const iconColor = active
    ? "text-accent-primary"
    : item.kind === "instance"
      ? "text-context-instance"
      : "text-text-secondary";
  const textColor = active ? "text-accent-primary" : "text-text-secondary";
  if (collapsed) {
    return (
      <a
        href={item.href}
        aria-label={item.label}
        aria-current={active ? "page" : undefined}
        className={cn(
          "flex h-10 w-10 items-center justify-center rounded-[var(--radius-md)] transition duration-[var(--dur-instant)]",
          "hover:bg-bg-elevated hover:text-text-primary",
          "focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-border-focus",
          active && "bg-accent-primary-mu",
        )}
      >
        <Icon className={cn("h-4 w-4", iconColor)} aria-hidden="true" />
      </a>
    );
  }
  return (
    <a
      href={item.href}
      aria-current={active ? "page" : undefined}
      className={cn(
        "flex h-8 items-center gap-2.5 rounded-[var(--radius-md)] px-2.5 text-[13px] font-medium transition duration-[var(--dur-instant)]",
        "hover:bg-bg-elevated hover:text-text-primary",
        "focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-border-focus",
        active ? "bg-accent-primary-mu" : "",
        textColor,
      )}
    >
      <Icon className={cn("h-4 w-4 flex-shrink-0", iconColor)} aria-hidden="true" />
      <span className="truncate">{item.label}</span>
    </a>
  );
}

export function SidebarNav({
  collapsed = false,
  className,
  label = "Dashboard navigation",
  scope = "instance",
  tenantSlug,
  onCreateTenant,
}: {
  collapsed?: boolean;
  className?: string;
  label?: string;
  scope?: "instance" | "tenant";
  tenantSlug?: string;
  onCreateTenant?: () => void;
}) {
  const pathname = typeof window !== "undefined" ? window.location.pathname : "";
  const activeTenantSlug = scope === "tenant" && tenantSlug ? tenantSlug : null;
  const primaryItems = activeTenantSlug ? tenantItems(activeTenantSlug) : instanceItems();
  const showInstanceSection = activeTenantSlug !== null;
  const secondaryItems = showInstanceSection ? instanceItems() : [];
  const activeHref = findActiveHref([...primaryItems, ...secondaryItems], pathname);
  return (
    <aside
      className={cn(
        "flex h-screen flex-col border-r border-border-subtle bg-bg-surface",
        collapsed ? "w-14 px-2 py-4" : "w-60 px-3 py-4",
        className,
      )}
      aria-label={label}
    >
      {collapsed ? (
        <>
          <div className="mb-6 flex h-7 items-center justify-center px-1 font-mono text-[18px] font-semibold text-text-primary">
            c
          </div>
          <nav
            className="flex min-h-0 flex-1 flex-col items-center gap-1 overflow-y-auto"
            aria-label={label}
          >
            {primaryItems.map((item) => (
              <SidebarNavLink
                key={item.href}
                item={item}
                collapsed
                active={item.href === activeHref}
              />
            ))}
            {showInstanceSection ? (
              <div className="my-2 h-px w-6 bg-border-subtle" aria-hidden="true" />
            ) : null}
            {secondaryItems.map((item) => (
              <SidebarNavLink
                key={item.href}
                item={item}
                collapsed
                active={item.href === activeHref}
              />
            ))}
          </nav>
        </>
      ) : (
        <>
          <div className="mb-5 flex flex-shrink-0 items-center px-1">
            <span className="font-mono text-[18px] font-semibold text-text-primary">cypra</span>
          </div>
          <div className="mb-4 flex-shrink-0">
            <TenantSwitcher
              label={tenantSlug ?? "Choose tenant"}
              slug={tenantSlug ?? ""}
              onCreateTenant={onCreateTenant}
            />
          </div>
          <nav className="flex min-h-0 flex-1 flex-col gap-0.5 overflow-y-auto" aria-label={label}>
            {primaryItems.map((item) => (
              <SidebarNavLink
                key={item.href}
                item={item}
                collapsed={false}
                active={item.href === activeHref}
              />
            ))}
            {showInstanceSection ? (
              <>
                <div className="mt-4 px-1 pb-2">
                  <div className="mb-2 h-px bg-border-subtle" />
                  <p className="text-[11px] font-semibold uppercase tracking-[0.7px] text-text-tertiary">
                    Instance admin
                  </p>
                </div>
                {secondaryItems.map((item) => (
                  <SidebarNavLink
                    key={item.href}
                    item={item}
                    collapsed={false}
                    active={item.href === activeHref}
                  />
                ))}
              </>
            ) : null}
          </nav>
          <div className="flex-shrink-0 pt-3">
            <div className="mb-2 h-px bg-border-subtle" />
            <div className="flex items-center justify-between gap-2 px-1">
              <a
                href="/dashboard/account"
                className={cn(
                  "flex min-w-0 flex-1 items-center gap-2 rounded-[var(--radius-md)] px-1 py-1 transition duration-[var(--dur-instant)]",
                  "hover:bg-bg-elevated",
                  "focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-border-focus",
                )}
              >
                <span
                  className="flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-full border border-border-default bg-bg-code"
                  aria-hidden="true"
                />
                <span className="truncate text-[13px] font-medium text-text-primary">Account</span>
              </a>
              <SidebarThemeCycleButton />
              <IconButton
                label="Sign out"
                icon={<Icons.LogOut className="h-3.5 w-3.5" aria-hidden="true" />}
                size="sm"
                variant="ghost"
                onClick={() => {
                  void signOut();
                }}
              />
            </div>
          </div>
        </>
      )}
    </aside>
  );
}

function SidebarThemeCycleButton() {
  const { preference, setPreference } = useTheme();
  const order: ThemePreference[] = ["system", "light", "dark"];
  const next = (current: ThemePreference): ThemePreference => {
    const idx = order.indexOf(current);
    return order[(idx + 1) % order.length];
  };
  const Icon =
    preference === "system" ? Icons.Monitor : preference === "light" ? Icons.Sun : Icons.Moon;
  return (
    <IconButton
      label={`Theme: ${preference} (click to cycle)`}
      icon={<Icon className="h-3.5 w-3.5" aria-hidden="true" />}
      size="sm"
      variant="ghost"
      onClick={() => setPreference(next(preference))}
    />
  );
}

export function ThemeToggle() {
  const { preference, setPreference } = useTheme();
  const options: ThemePreference[] = ["system", "dark", "light"];
  return (
    <SegmentedControl
      label="Theme"
      options={options}
      value={preference}
      onChange={(value) => setPreference(value as ThemePreference)}
    />
  );
}

export function PrimitiveGallery() {
  const [seg, setSeg] = useState("One");
  const [comboboxValue, setComboboxValue] = useState("owner");
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
        <Card title="Combobox states">
          <div className="grid gap-20 md:grid-cols-2">
            <Combobox
              label="Role combobox"
              value={comboboxValue}
              onChange={setComboboxValue}
              options={[
                { value: "owner", label: "Owner", group: "Tenant roles" },
                { value: "admin", label: "Admin", group: "Tenant roles" },
                { value: "member", label: "Member", group: "Tenant roles" },
              ]}
              helper="Arrow keys, Home/End, Enter, and Escape are wired."
              defaultOpen
            />
            <Combobox
              label="Loading options"
              options={[]}
              state="loading"
              helper="Skeleton rows occupy the listbox."
              defaultOpen
            />
            <Combobox label="Filtered empty" options={[]} state="empty" defaultOpen />
            <Combobox
              label="Error with retry"
              options={[]}
              state="error"
              error="Role service unavailable."
              defaultOpen
            />
            <Combobox
              label="Readonly combobox"
              value="owner"
              options={[{ value: "owner", label: "Owner" }]}
              state="readonly"
              helper="Readonly trigger uses code background."
            />
          </div>
        </Card>
        <Card title="Popover and menu states">
          <div className="flex flex-wrap items-start gap-4 pb-24">
            <Popover label="Open popover" defaultOpen>
              <div className="grid gap-2 text-[13px] text-text-secondary">
                <strong className="text-text-primary">Session controls</strong>
                <span>Escape closes. Tab leaves the panel naturally.</span>
                <Button size="sm">Primary action</Button>
              </div>
            </Popover>
            <DropdownMenu
              label="Actions menu"
              defaultOpen
              items={[
                { label: "Copy identifier" },
                { label: "Rotate secret" },
                { label: "Disabled action", disabled: true },
                { label: "Delete project", destructive: true },
              ]}
            />
          </div>
        </Card>
        <Card title="Toolbar">
          <div className="grid gap-3">
            <Toolbar label="Audit log filters">
              <Button size="sm" variant="secondary">
                Last 24h
              </Button>
              <Button size="sm" variant="secondary">
                Failures
              </Button>
              <IconButton label="Refresh" icon={<Icons.RotateCw className="h-4 w-4" />} size="sm" />
              <Button size="sm" variant="ghost" disabled>
                Export locked
              </Button>
            </Toolbar>
            <SegmentedControl
              label="Density"
              options={["Compact", "Comfortable", "Expanded"]}
              value="Comfortable"
              onChange={() => undefined}
              disabledOptions={["Expanded"]}
            />
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
            <div className="grid gap-3">
              <h3 className="text-[13px] font-semibold uppercase tracking-[0.06em] text-text-tertiary">
                Permission matrix
              </h3>
              <PermissionMatrix />
            </div>
            <ConfirmationDialog
              open={false}
              headline="Confirm destructive action"
              body="Type acme to continue."
              resourceMatch="acme"
              onConfirm={() => undefined}
              onCancel={() => undefined}
            />
            <SettingsRow
              label="Branding preview"
              helper="Open a hosted-login preview before saving."
              control={<Button>Preview</Button>}
            />
            <ListRow title="Ada Lovelace" meta="owner" />
            <SaveBar dirtyCount={2} onSave={() => undefined} onDiscard={() => undefined} />
          </div>
        </Card>
        <Card title="Segmented">
          <SegmentedControl options={["One", "Two", "Three"]} value={seg} onChange={setSeg} />
        </Card>
        <GalleryStateMatrix />
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
        <GalleryStateMatrix compact />
      </div>
      <div
        data-mode="dark"
        className="rounded-[var(--radius-lg)] bg-bg-canvas p-4 text-text-primary"
      >
        <PageHeader
          title="Dark mode parity"
          subtitle="Token-bound rendering inside a dark-mode island."
        />
        <GalleryStateMatrix compact />
      </div>
    </div>
  );
}

function GalleryStateMatrix({ compact = false }: { compact?: boolean }) {
  return (
    <Card
      title={compact ? "State matrix" : "Documented state matrix"}
      subtitle="Loading, empty, error, disabled, permission-denied, open-overlay, and toast-stack states."
    >
      <div className={cn("grid gap-4", !compact && "xl:grid-cols-2")}>
        <div className="grid gap-3">
          <Toolbar label="Disabled and loading controls">
            <Button size="sm" loading>
              Loading
            </Button>
            <Button size="sm" disabled>
              Disabled
            </Button>
            <IconButton
              label="Disabled refresh"
              icon={<Icons.RotateCw className="h-4 w-4" />}
              size="sm"
              disabled
            />
          </Toolbar>
          <TextInput label="Validating field" value="acme" state="validating" readOnly />
          <TextInput
            label="Errored field"
            value="bad slug"
            error="Use lowercase letters, numbers, and hyphens."
          />
          <Select label="Disabled select" disabled options={[{ value: "owner", label: "Owner" }]} />
          <SegmentedControl
            label="Disabled segmented option"
            options={["Read", "Write", "Admin"]}
            value="Read"
            onChange={() => undefined}
            disabledOptions={["Admin"]}
          />
        </div>
        <div className="grid gap-3">
          <LoadingState />
          <EmptyState
            title="Empty state"
            body="No records match this filter."
            action={<Button size="sm">Create record</Button>}
          />
          <ErrorState
            title="Error state"
            body="The backing API failed. Retry without demo fallbacks."
            retry={() => undefined}
          />
          <ErrorState title="Permission denied" body="Ask an owner for the missing permission." />
        </div>
        <div className="grid gap-2">
          <Toast variant="success" message="Saved successfully." />
          <Toast variant="info" message="Live updates connected." />
          <Toast variant="warn" message="Deletion is scheduled." />
          <Toast variant="error" message="Provider diagnostic failed." />
        </div>
        <div className="flex flex-wrap items-start gap-4 pb-16">
          <Popover label="Matrix popover" defaultOpen>
            <span className="text-[13px] text-text-secondary">Open overlay state.</span>
          </Popover>
          <DropdownMenu
            label="Matrix menu"
            defaultOpen
            items={[
              { label: "Enabled action" },
              { label: "Disabled action", disabled: true },
              { label: "Destructive action", destructive: true },
            ]}
          />
        </div>
      </div>
    </Card>
  );
}
