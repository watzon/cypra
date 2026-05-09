import { useEffect, useState, type ReactNode } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import {
  createPAT,
  createProject,
  createTenant,
  inviteUser,
  regenerateBackupCodes,
  registerPasskey,
  type CreatePATResult,
  type ProjectRecord,
  type TenantRecord,
} from "@/api";
import {
  BackupCodeGrid,
  Button,
  Checkbox,
  ConfirmationDialog,
  IdentifierPill,
  MaskedSecret,
  Modal,
  Select,
  TextInput,
} from "@/components";

interface ModalShellProps {
  open: boolean;
  onClose: () => void;
}

function FormError({ message }: { message?: string }) {
  if (!message) return null;
  return (
    <p className="text-[13px] text-status-error" role="alert">
      {message}
    </p>
  );
}

// ─── CreateTenantModal ──────────────────────────────────────────────────────

export function CreateTenantModal({
  open,
  onClose,
  onSuccess,
}: ModalShellProps & { onSuccess?: (tenant: TenantRecord) => void }) {
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const queryClient = useQueryClient();
  const mutation = useMutation({
    mutationFn: (input: { name: string; slug: string }) => createTenant(input),
    onSuccess: (tenant) => {
      void queryClient.invalidateQueries({ queryKey: ["tenants"] });
      onSuccess?.(tenant);
      onClose();
      setName("");
      setSlug("");
    },
  });
  const ready = name.trim().length > 0 && /^[a-z0-9-]+$/.test(slug);
  const errorMessage = mutation.error instanceof Error ? mutation.error.message : undefined;
  return (
    <Modal
      title="Create tenant"
      open={open}
      onClose={onClose}
      size="sm"
      footer={
        <>
          <Button variant="secondary" kbd="Esc" onClick={onClose} disabled={mutation.isPending}>
            Cancel
          </Button>
          <Button
            variant="primary"
            disabled={!ready}
            loading={mutation.isPending}
            onClick={() => mutation.mutate({ name: name.trim(), slug: slug.trim() })}
          >
            Create tenant
          </Button>
        </>
      }
    >
      <div className="grid gap-4">
        <TextInput
          label="Display name"
          value={name}
          placeholder="Acme Operations"
          onChange={(event) => setName(event.target.value)}
        />
        <TextInput
          label="Slug"
          value={slug}
          placeholder="acme"
          helper="Lowercase letters, digits, and hyphens. Becomes the subdomain."
          onChange={(event) => setSlug(event.target.value.toLowerCase())}
        />
        <FormError message={errorMessage} />
      </div>
    </Modal>
  );
}

// ─── CreateProjectModal ─────────────────────────────────────────────────────

export function CreateProjectModal({
  open,
  onClose,
  tenantSlug,
  onSuccess,
}: ModalShellProps & { tenantSlug: string; onSuccess?: (project: ProjectRecord) => void }) {
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const queryClient = useQueryClient();
  const mutation = useMutation({
    mutationFn: (input: { name: string; slug: string }) => createProject(input),
    onSuccess: (project) => {
      void queryClient.invalidateQueries({ queryKey: ["projects", tenantSlug] });
      onSuccess?.(project);
      onClose();
      setName("");
      setSlug("");
    },
  });
  const ready = name.trim().length > 0 && /^[a-z0-9-]+$/.test(slug);
  const errorMessage = mutation.error instanceof Error ? mutation.error.message : undefined;
  return (
    <Modal
      title="Create project"
      open={open}
      onClose={onClose}
      size="sm"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={mutation.isPending}>
            Cancel
          </Button>
          <Button
            variant="primary"
            disabled={!ready}
            loading={mutation.isPending}
            onClick={() => mutation.mutate({ name: name.trim(), slug: slug.trim() })}
          >
            Create project
          </Button>
        </>
      }
    >
      <div className="grid gap-4">
        <TextInput
          label="Project name"
          value={name}
          placeholder="Console App"
          onChange={(event) => setName(event.target.value)}
        />
        <TextInput
          label="Slug"
          value={slug}
          placeholder="console"
          helper="Used in issuer URL paths and client IDs."
          onChange={(event) => setSlug(event.target.value.toLowerCase())}
        />
        <FormError message={errorMessage} />
      </div>
    </Modal>
  );
}

// ─── InviteUserModal ────────────────────────────────────────────────────────

type InviteRole = "owner" | "admin" | "member";

export function InviteUserModal({
  open,
  onClose,
  roleLock,
  title = "Invite user",
  onSuccess,
}: ModalShellProps & {
  roleLock?: InviteRole;
  title?: string;
  onSuccess?: () => void;
}) {
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<InviteRole>(roleLock ?? "member");
  useEffect(() => {
    if (roleLock) setRole(roleLock);
  }, [roleLock]);
  const queryClient = useQueryClient();
  const mutation = useMutation({
    mutationFn: (input: { email: string; role: InviteRole }) => inviteUser(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["users"] });
      void queryClient.invalidateQueries({ queryKey: ["members"] });
      void queryClient.invalidateQueries({ queryKey: ["tenant-invites"] });
      onSuccess?.();
      onClose();
      setEmail("");
    },
  });
  const ready = /^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(email.trim());
  const errorMessage = mutation.error instanceof Error ? mutation.error.message : undefined;
  return (
    <Modal
      title={title}
      open={open}
      onClose={onClose}
      size="sm"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={mutation.isPending}>
            Cancel
          </Button>
          <Button
            variant="primary"
            disabled={!ready}
            loading={mutation.isPending}
            onClick={() => mutation.mutate({ email: email.trim(), role })}
          >
            Send invite
          </Button>
        </>
      }
    >
      <div className="grid gap-4">
        <TextInput
          label="Email"
          type="email"
          value={email}
          placeholder="user@example.com"
          onChange={(event) => setEmail(event.target.value)}
        />
        {roleLock ? null : (
          <Select
            label="Role"
            value={role}
            onChange={(event) => setRole(event.target.value as InviteRole)}
            options={[
              { value: "owner", label: "Owner" },
              { value: "admin", label: "Admin" },
              { value: "member", label: "Member" },
            ]}
          />
        )}
        <FormError message={errorMessage} />
      </div>
    </Modal>
  );
}

// ─── CreatePATModal ─────────────────────────────────────────────────────────

const PAT_SCOPES = [
  "projects.read",
  "projects.write",
  "users.read",
  "users.write",
  "pats.read",
  "pats.write",
  "audit.export",
] as const;

export function CreatePATModal({ open, onClose }: ModalShellProps) {
  const [name, setName] = useState("");
  const [scopes, setScopes] = useState<string[]>(["projects.read"]);
  const [revealed, setRevealed] = useState<CreatePATResult | null>(null);
  const queryClient = useQueryClient();
  const mutation = useMutation({
    mutationFn: (input: { name: string; scopes: string[] }) => createPAT(input.name, input.scopes),
    onSuccess: (result) => {
      void queryClient.invalidateQueries({ queryKey: ["pats"] });
      setRevealed(result);
    },
  });

  // beforeunload guard while plaintext is shown but not acknowledged
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

  const reset = () => {
    setName("");
    setScopes(["projects.read"]);
    setRevealed(null);
    mutation.reset();
  };
  const close = () => {
    reset();
    onClose();
  };
  const errorMessage = mutation.error instanceof Error ? mutation.error.message : undefined;
  const ready = name.trim().length > 0 && scopes.length > 0;

  return (
    <Modal
      title="Create personal access token"
      open={open}
      onClose={close}
      size="md"
      footer={
        revealed ? (
          <Button variant="primary" onClick={close}>
            I have copied this token
          </Button>
        ) : (
          <>
            <Button variant="ghost" onClick={close} disabled={mutation.isPending}>
              Cancel
            </Button>
            <Button
              variant="primary"
              disabled={!ready}
              loading={mutation.isPending}
              onClick={() => mutation.mutate({ name: name.trim(), scopes })}
            >
              Create token
            </Button>
          </>
        )
      }
    >
      {revealed ? (
        <div className="grid gap-4">
          <p className="text-[14px] text-text-secondary">
            This token is shown once. Copy it before closing — there is no recovery.
          </p>
          <MaskedSecret name="cypra_pat" value={revealed.token} />
        </div>
      ) : (
        <div className="grid gap-4">
          <TextInput
            label="Token name"
            placeholder="Deploy automation"
            value={name}
            onChange={(event) => setName(event.target.value)}
          />
          <fieldset className="grid gap-2 text-[14px]">
            <legend className="text-[13px] font-medium text-text-secondary">Scopes</legend>
            {PAT_SCOPES.map((scope) => (
              <Checkbox
                key={scope}
                label={scope}
                checked={scopes.includes(scope)}
                onChange={(event) => {
                  setScopes((current) =>
                    event.target.checked ? [...current, scope] : current.filter((s) => s !== scope),
                  );
                }}
              />
            ))}
          </fieldset>
          <FormError message={errorMessage} />
        </div>
      )}
    </Modal>
  );
}

// ─── RegenerateBackupCodesModal ────────────────────────────────────────────

export function RegenerateBackupCodesModal({
  open,
  onClose,
  userID,
}: ModalShellProps & { userID: string }) {
  const [codes, setCodes] = useState<string[] | null>(null);
  const mutation = useMutation({
    mutationFn: () => regenerateBackupCodes(userID),
    onSuccess: (result) => setCodes(result.codes),
  });

  const reset = () => {
    setCodes(null);
    mutation.reset();
  };
  const close = () => {
    reset();
    onClose();
  };
  const errorMessage = mutation.error instanceof Error ? mutation.error.message : undefined;

  if (codes) {
    return (
      <Modal
        title="New backup codes"
        open={open}
        onClose={close}
        size="md"
        footer={
          <Button variant="primary" onClick={close}>
            Done
          </Button>
        }
      >
        <BackupCodeGrid codes={codes} />
      </Modal>
    );
  }

  return (
    <ConfirmationDialog
      open={open}
      variant="warn"
      headline="Regenerate backup codes?"
      body={
        <>
          Existing codes are invalidated immediately. Save the new codes before closing — they are
          shown only once.
        </>
      }
      confirmLabel="Regenerate"
      loading={mutation.isPending}
      errorMessage={errorMessage}
      onConfirm={() => mutation.mutate()}
      onCancel={close}
    />
  );
}

// ─── AddPasskeyModal ────────────────────────────────────────────────────────

export function AddPasskeyModal({
  open,
  onClose,
  userID,
  rpID,
}: ModalShellProps & { userID: string; rpID: string }) {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<"idle" | "ceremony" | "registering" | "done">("idle");
  const [errorMessage, setErrorMessage] = useState<string | undefined>();

  const reset = () => {
    setStatus("idle");
    setErrorMessage(undefined);
  };
  const close = () => {
    reset();
    onClose();
  };

  const start = async () => {
    setStatus("ceremony");
    setErrorMessage(undefined);
    try {
      const challenge = crypto.getRandomValues(new Uint8Array(32));
      const subjectBytes = new TextEncoder().encode(userID);
      const credential = (await navigator.credentials.create({
        publicKey: {
          challenge,
          rp: { id: rpID, name: "Cypra" },
          user: {
            id: subjectBytes,
            name: userID,
            displayName: userID,
          },
          pubKeyCredParams: [{ type: "public-key", alg: -7 }],
          authenticatorSelection: { userVerification: "preferred" },
          timeout: 60_000,
        },
      })) as PublicKeyCredential | null;
      if (!credential) {
        throw new Error("passkey.no_credential");
      }
      const credentialID = bufferToBase64Url(credential.rawId);
      const publicKey = bufferToBase64Url(
        (credential.response as AuthenticatorAttestationResponse).getPublicKey() ??
          new ArrayBuffer(0),
      );
      setStatus("registering");
      await registerPasskey({ userID, credentialID, publicKey });
      setStatus("done");
      void queryClient.invalidateQueries({ queryKey: ["passkeys"] });
    } catch (error) {
      setStatus("idle");
      setErrorMessage(error instanceof Error ? error.message : "passkey.register_failed");
    }
  };

  const renderBody = (): ReactNode => {
    if (status === "done") {
      return (
        <p className="text-[14px] text-text-secondary">
          Passkey enrolled. Use it on your next sign-in.
        </p>
      );
    }
    if (status === "ceremony" || status === "registering") {
      return (
        <p className="text-[14px] text-text-secondary">
          Touch your security key or use your platform authenticator…
        </p>
      );
    }
    return (
      <p className="text-[14px] text-text-secondary">
        Use your security key or platform authenticator (Touch ID, Windows Hello) to enroll a
        passkey for this account.
      </p>
    );
  };

  return (
    <Modal
      title="Add passkey"
      open={open}
      onClose={close}
      size="sm"
      footer={
        status === "done" ? (
          <Button variant="primary" onClick={close}>
            Done
          </Button>
        ) : (
          <>
            <Button variant="ghost" onClick={close} disabled={status !== "idle"}>
              Cancel
            </Button>
            <Button
              variant="primary"
              loading={status === "ceremony" || status === "registering"}
              onClick={() => void start()}
            >
              Enroll passkey
            </Button>
          </>
        )
      }
    >
      <div className="grid gap-4">
        {renderBody()}
        {status === "idle" && errorMessage ? <FormError message={errorMessage} /> : null}
      </div>
    </Modal>
  );
}

function bufferToBase64Url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = "";
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }
  return btoa(binary).replaceAll("+", "-").replaceAll("/", "_").replace(/=+$/, "");
}

// ─── re-exports for convenience ────────────────────────────────────────────

export { IdentifierPill };
