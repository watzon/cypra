import Activity from "lucide-react/dist/esm/icons/activity.mjs";
import AlertTriangle from "lucide-react/dist/esm/icons/alert-triangle.mjs";
import Building2 from "lucide-react/dist/esm/icons/building-2.mjs";
import CheckCircle2 from "lucide-react/dist/esm/icons/circle-check-big.mjs";
import ChevronDown from "lucide-react/dist/esm/icons/chevron-down.mjs";
import ChevronLeft from "lucide-react/dist/esm/icons/chevron-left.mjs";
import ChevronRight from "lucide-react/dist/esm/icons/chevron-right.mjs";
import ChevronsUpDown from "lucide-react/dist/esm/icons/chevrons-up-down.mjs";
import Clock from "lucide-react/dist/esm/icons/clock.mjs";
import Copy from "lucide-react/dist/esm/icons/copy.mjs";
import Eye from "lucide-react/dist/esm/icons/eye.mjs";
import EyeOff from "lucide-react/dist/esm/icons/eye-off.mjs";
import Folder from "lucide-react/dist/esm/icons/folder.mjs";
import HardDrive from "lucide-react/dist/esm/icons/hard-drive.mjs";
import Hash from "lucide-react/dist/esm/icons/hash.mjs";
import ImageIcon from "lucide-react/dist/esm/icons/image.mjs";
import ImageUp from "lucide-react/dist/esm/icons/image-up.mjs";
import Info from "lucide-react/dist/esm/icons/info.mjs";
import Key from "lucide-react/dist/esm/icons/key.mjs";
import KeyRound from "lucide-react/dist/esm/icons/key-round.mjs";
import Laptop from "lucide-react/dist/esm/icons/laptop.mjs";
import LayoutDashboard from "lucide-react/dist/esm/icons/layout-dashboard.mjs";
import Lock from "lucide-react/dist/esm/icons/lock.mjs";
import LogOut from "lucide-react/dist/esm/icons/log-out.mjs";
import Monitor from "lucide-react/dist/esm/icons/monitor.mjs";
import Moon from "lucide-react/dist/esm/icons/moon.mjs";
import Plus from "lucide-react/dist/esm/icons/plus.mjs";
import RotateCw from "lucide-react/dist/esm/icons/rotate-cw.mjs";
import ScrollText from "lucide-react/dist/esm/icons/scroll-text.mjs";
import Search from "lucide-react/dist/esm/icons/search.mjs";
import SearchX from "lucide-react/dist/esm/icons/search-x.mjs";
import Settings from "lucide-react/dist/esm/icons/settings.mjs";
import Shield from "lucide-react/dist/esm/icons/shield.mjs";
import ShieldCheck from "lucide-react/dist/esm/icons/shield-check.mjs";
import Smartphone from "lucide-react/dist/esm/icons/smartphone.mjs";
import SquareDot from "lucide-react/dist/esm/icons/square-dot.mjs";
import Sun from "lucide-react/dist/esm/icons/sun.mjs";
import Terminal from "lucide-react/dist/esm/icons/terminal.mjs";
import Trash2 from "lucide-react/dist/esm/icons/trash-2.mjs";
import TriangleAlert from "lucide-react/dist/esm/icons/triangle-alert.mjs";
import X from "lucide-react/dist/esm/icons/x.mjs";
import UserCog from "lucide-react/dist/esm/icons/user-cog.mjs";
import Users from "lucide-react/dist/esm/icons/users.mjs";
import XCircle from "lucide-react/dist/esm/icons/circle-x.mjs";
import type { SVGProps } from "react";

export const Icons = {
  Activity,
  AlertTriangle,
  Building2,
  CheckCircle2,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  ChevronsUpDown,
  Clock,
  Copy,
  Eye,
  EyeOff,
  Folder,
  HardDrive,
  Hash,
  Image: ImageIcon,
  ImageUp,
  Info,
  Key,
  KeyRound,
  Laptop,
  LayoutDashboard,
  Lock,
  LogOut,
  Monitor,
  Moon,
  Plus,
  RotateCw,
  ScrollText,
  Search,
  SearchX,
  Settings,
  Shield,
  ShieldCheck,
  Smartphone,
  SquareDot,
  Sun,
  Terminal,
  Trash2,
  TriangleAlert,
  UserCog,
  X,
  Users,
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
