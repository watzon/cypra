import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function middleEllipsis(value: string, edge = 8) {
  if (value.length <= edge * 2 + 1) {
    return value;
  }
  return `${value.slice(0, edge)}...${value.slice(-edge)}`;
}

export function actorThemeEndpoint(actorKind: "tenant" | "instance") {
  return actorKind === "instance" ? "/api/v1/instance/admins/me" : "/api/v1/users/me";
}

const ERROR_MESSAGES: Record<string, string> = {
  "provider.email_invalid":
    "That email provider config is incomplete. Make sure From address, From name, and the provider credentials are all filled in.",
  "provider.email_save_failed": "Couldn't save the email provider. Try again in a moment.",
  "provider.email_test_failed":
    "Test email failed. Double-check the SMTP host or Resend API key and try again.",
  "provider.email_failed": "Couldn't load the email provider configuration.",
  "provider.upstream_invalid": "That upstream provider config is incomplete or malformed.",
  "provider.upstream_save_failed": "Couldn't save the upstream provider. Try again in a moment.",
  "provider.upstream_test_failed":
    "Upstream OAuth round-trip failed. Verify the client ID and secret.",
  "auth.signup_closed":
    "Self-serve sign-up is closed for this tenant. Ask an administrator for an invite.",
  "auth.signup_restricted":
    "Sign-up is restricted to specific email addresses or domains for this tenant.",
  "auth.signup_disabled_for_method":
    "Sign-up is disabled for this auth method.",
  "auth.invites_disabled":
    "This tenant isn't accepting invites right now.",
  "auth_providers.registration_load_failed": "Couldn't load registration settings.",
  "auth_providers.registration_save_failed": "Couldn't save registration settings. Try again.",
  "auth_providers.registration_mode_invalid": "That sign-up mode isn't valid.",
  "auth_providers.registration_allowlist_invalid":
    "Allowlist contains too many entries (max 256).",
  "auth_providers.social_unknown_kind": "That social provider isn't supported.",
  "auth_providers.social_credentials_required":
    "Provide both a Client ID and Client secret to save this connection.",
  "auth_providers.social_save_failed": "Couldn't save the social connection. Try again.",
  "auth_providers.social_delete_failed": "Couldn't remove the social connection. Try again.",
  "auth_providers.social_not_configured":
    "This social provider hasn't been configured for this tenant yet.",
  "auth_providers.social_invalid_state":
    "The sign-in state was invalid or expired. Start the flow again.",
  "auth_providers.social_exchange_failed":
    "Couldn't complete the OAuth exchange with the upstream provider.",
  "auth_providers.social_email_not_allowed":
    "The provider returned an email outside this tenant's allowed-domain list.",
  "auth_providers.social_user_failed":
    "Couldn't create or sign in the user from this provider.",
  "auth_providers.social_start_failed": "Couldn't start the upstream OAuth flow.",
  "auth_providers.oidc_invalid_slug":
    "Connection slug must be 1–63 characters using lowercase letters, digits, hyphens, or underscores.",
  "auth_providers.oidc_invalid_payload":
    "Provide a display name, issuer URL, client ID, and client secret to add a connection.",
  "auth_providers.oidc_save_failed": "Couldn't save the OIDC connection. Try again.",
  "auth_providers.oidc_delete_failed": "Couldn't remove the OIDC connection. Try again.",
  "auth_providers.oidc_slug_taken":
    "A connection with that slug already exists for this tenant.",
  "auth_providers.oidc_not_found": "No OIDC connection found for that slug.",
  "auth_providers.oidc_not_configured":
    "This enterprise OIDC connection isn't enabled for this tenant.",
  "auth_providers.oidc_decrypt_failed":
    "Couldn't decrypt the stored OIDC client credentials.",
  "auth_providers.oidc_invalid_state":
    "The sign-in state was invalid or expired. Start the flow again.",
  "auth_providers.oidc_exchange_failed":
    "Couldn't complete the OIDC exchange with the upstream provider.",
  "auth_providers.oidc_email_not_allowed":
    "The provider returned an email outside this connection's allowed-domain list.",
  "auth_providers.oidc_user_failed": "Couldn't create or sign in the user from this provider.",
  "auth_providers.oidc_start_failed": "Couldn't start the OIDC flow.",
};

export function formatErrorCode(code: string | undefined | null): string {
  if (!code) return "Something went wrong. Try again.";
  return ERROR_MESSAGES[code] ?? code;
}
