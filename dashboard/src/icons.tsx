import AlertTriangle from "lucide-react/dist/esm/icons/alert-triangle.mjs";
import CheckCircle2 from "lucide-react/dist/esm/icons/circle-check-big.mjs";
import Clock from "lucide-react/dist/esm/icons/clock.mjs";
import Copy from "lucide-react/dist/esm/icons/copy.mjs";
import Eye from "lucide-react/dist/esm/icons/eye.mjs";
import EyeOff from "lucide-react/dist/esm/icons/eye-off.mjs";
import Lock from "lucide-react/dist/esm/icons/lock.mjs";
import Monitor from "lucide-react/dist/esm/icons/monitor.mjs";
import RotateCw from "lucide-react/dist/esm/icons/rotate-cw.mjs";
import Trash2 from "lucide-react/dist/esm/icons/trash-2.mjs";
import XCircle from "lucide-react/dist/esm/icons/circle-x.mjs";
import type { SVGProps } from "react";

export const Icons = {
  AlertTriangle,
  CheckCircle2,
  Clock,
  Copy,
  Eye,
  EyeOff,
  Lock,
  Monitor,
  RotateCw,
  Trash2,
  XCircle,
};

type GlyphProps = SVGProps<SVGSVGElement> & { title?: string };

export function TenantGlyph({ title, ...props }: GlyphProps) {
  return (
    <svg
      viewBox="0 0 24 24"
      aria-hidden={title ? undefined : true}
      role={title ? "img" : undefined}
      {...props}
    >
      {title ? <title>{title}</title> : null}
      <path
        d="M8 5H5v14h3M16 5h3v14h-3"
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="1.5"
      />
      <circle cx="12" cy="12" r="2.25" fill="currentColor" />
    </svg>
  );
}

export function InstanceGlyph({ title, ...props }: GlyphProps) {
  return (
    <svg
      viewBox="0 0 24 24"
      aria-hidden={title ? undefined : true}
      role={title ? "img" : undefined}
      {...props}
    >
      {title ? <title>{title}</title> : null}
      <circle cx="8.5" cy="8.5" r="2.2" fill="currentColor" />
      <circle cx="15.5" cy="8.5" r="2.2" fill="currentColor" />
      <circle cx="8.5" cy="15.5" r="2.2" fill="currentColor" />
      <circle cx="15.5" cy="15.5" r="2.2" fill="currentColor" />
    </svg>
  );
}

export function PasskeyGlyph({ title, ...props }: GlyphProps) {
  return (
    <svg
      viewBox="0 0 24 24"
      aria-hidden={title ? undefined : true}
      role={title ? "img" : undefined}
      {...props}
    >
      {title ? <title>{title}</title> : null}
      <path
        d="M7.5 11.2a4.5 4.5 0 1 1 8.6 1.9M9 11.5c0-1.7 1.3-3 3-3s3 1.3 3 3M10.6 14.3c.8.9 2 1.2 3.1.8M12 11.4v2.2"
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeWidth="1.5"
      />
      <path
        d="M14.5 17.5h5m-2.2 0v2m-2.8-2 1.5-1.5"
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="1.5"
      />
    </svg>
  );
}

export function OidcGlyph({ title, ...props }: GlyphProps) {
  return (
    <svg
      viewBox="0 0 24 24"
      aria-hidden={title ? undefined : true}
      role={title ? "img" : undefined}
      {...props}
    >
      {title ? <title>{title}</title> : null}
      <path
        d="M6.2 14.5a6.5 6.5 0 0 1 11.6 0M8.7 12.4a3.7 3.7 0 0 1 6.6 0M4 17.2a9.1 9.1 0 0 1 16 0"
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeWidth="1.5"
      />
      <circle cx="12" cy="16.2" r="1.5" fill="currentColor" />
    </svg>
  );
}
