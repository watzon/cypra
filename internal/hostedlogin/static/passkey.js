function b64urlToBuffer(value) {
  const padded = value.replace(/-/g, "+").replace(/_/g, "/") + "===".slice((value.length + 3) % 4);
  const binary = window.atob(padded);
  const bytes = new Uint8Array(binary.length);
  for (let index = 0; index < binary.length; index += 1) bytes[index] = binary.charCodeAt(index);
  return bytes.buffer;
}

function bufferToB64url(value) {
  const bytes = new Uint8Array(value || new ArrayBuffer(0));
  let binary = "";
  for (const byte of bytes) binary += String.fromCharCode(byte);
  return window.btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");
}

function publicKeyFromEnvelope(envelope) {
  const options = envelope.options?.publicKey || envelope.publicKey || envelope.options?.response;
  if (!options) throw new PasskeyError("server", "missing public key options");
  options.challenge = b64urlToBuffer(options.challenge);
  if (options.user?.id) options.user.id = b64urlToBuffer(options.user.id);
  for (const descriptor of options.excludeCredentials || [])
    descriptor.id = b64urlToBuffer(descriptor.id);
  for (const descriptor of options.allowCredentials || [])
    descriptor.id = b64urlToBuffer(descriptor.id);
  return options;
}

function credentialToJSON(credential) {
  const response = credential.response;
  const payload = {
    id: credential.id,
    rawId: bufferToB64url(credential.rawId),
    type: credential.type,
    authenticatorAttachment: credential.authenticatorAttachment,
    clientExtensionResults: credential.getClientExtensionResults?.() || {},
    response: {
      clientDataJSON: bufferToB64url(response.clientDataJSON),
    },
  };
  if (response.attestationObject) {
    payload.response.attestationObject = bufferToB64url(response.attestationObject);
    payload.response.transports = response.getTransports?.() || [];
  }
  if (response.authenticatorData)
    payload.response.authenticatorData = bufferToB64url(response.authenticatorData);
  if (response.signature) payload.response.signature = bufferToB64url(response.signature);
  if (response.userHandle) payload.response.userHandle = bufferToB64url(response.userHandle);
  return payload;
}

class PasskeyError extends Error {
  constructor(kind, message) {
    super(message);
    this.name = "PasskeyError";
    this.kind = kind;
  }
}

// classifyPasskeyError maps any error thrown during a passkey ceremony to a
// user-facing string. The classification is intentionally coarse so we never
// echo RP IDs, origins, or server detail back to an end user.
function classifyPasskeyError(err) {
  if (err instanceof PasskeyError) {
    switch (err.kind) {
      case "unsupported":
        return "Passkeys aren't supported on this browser yet. Try a different browser or sign-in method.";
      case "cancelled":
        return "Sign-in was cancelled. Tap the passkey button to try again.";
      case "origin":
        return "This sign-in link can't be used here. Contact your administrator.";
      case "network":
        return "We couldn't reach the sign-in service. Check your connection and try again.";
      case "server":
        return "Sign-in didn't go through. Try again, or use another method.";
      default:
        return "Sign-in didn't go through. Try again, or use another method.";
    }
  }
  if (err && typeof err === "object" && typeof err.name === "string") {
    if (err.name === "NotAllowedError" || err.name === "AbortError") {
      return "Sign-in was cancelled. Tap the passkey button to try again.";
    }
    if (err.name === "NotSupportedError" || err.name === "InvalidStateError") {
      return "Passkeys aren't available on this device. Try another sign-in method.";
    }
    if (err.name === "SecurityError") {
      return "This sign-in link can't be used here. Contact your administrator.";
    }
  }
  return "Sign-in didn't go through. Try again, or use another method.";
}

async function postJSON(path, body) {
  let response;
  try {
    response = await window.fetch(path, {
      method: "POST",
      headers: { "Content-Type": "application/json", Accept: "application/json" },
      credentials: "same-origin",
      body: JSON.stringify(body),
    });
  } catch (err) {
    throw new PasskeyError("network", "fetch failed");
  }
  const data = await response.json().catch(() => ({}));
  if (!response.ok) {
    // We do not surface server-supplied detail to the user — it can leak
    // tenant/RP information. We use the status to pick a class only.
    if (response.status >= 500) throw new PasskeyError("server", "server rejected");
    if (response.status === 403 || response.status === 404)
      throw new PasskeyError("origin", "rejected by origin/policy");
    throw new PasskeyError("server", "server rejected");
  }
  return data;
}

function userIDFor(target) {
  return target.dataset.userId || document.querySelector("[name=user_id]")?.value || "";
}

function emailFor(target) {
  const id = target.dataset.emailFrom;
  if (!id) return "";
  const input = document.getElementById(id);
  return input ? input.value.trim() : "";
}

async function cypraPasskey(action, target) {
  if (!window.PublicKeyCredential || !navigator.credentials)
    throw new PasskeyError("unsupported", "WebAuthn not available");
  if (action === "register-account") {
    const email = emailFor(target);
    if (!email) throw new PasskeyError("server", "missing email");
    const begin = await postJSON("/signup/passkey", { email });
    const credential = await navigator.credentials.create({
      publicKey: publicKeyFromEnvelope(begin),
    });
    await postJSON("/signup/passkey", {
      ceremony_id: begin.ceremony_id,
      user_id: begin.user_id,
      response: credentialToJSON(credential),
    });
    return "Account created. Continue to your application.";
  }
  if (action === "create") {
    const userId = userIDFor(target);
    if (!userId) throw new PasskeyError("server", "missing user id");
    const begin = await postJSON("/api/v1/auth/passkey/register", { user_id: userId });
    const credential = await navigator.credentials.create({
      publicKey: publicKeyFromEnvelope(begin),
    });
    await postJSON("/api/v1/auth/passkey/register", {
      user_id: userId,
      ceremony_id: begin.ceremony_id,
      response: credentialToJSON(credential),
    });
    return "Passkey added.";
  }
  if (action === "2fa") {
    const userId = userIDFor(target);
    const begin = await postJSON(
      "/api/v1/auth/webauthn2fa/verify",
      userId ? { user_id: userId } : {},
    );
    const credential = await navigator.credentials.get({ publicKey: publicKeyFromEnvelope(begin) });
    await postJSON("/api/v1/auth/webauthn2fa/verify", {
      ceremony_id: begin.ceremony_id,
      response: credentialToJSON(credential),
    });
    return "Factor accepted.";
  }
  const begin = await postJSON("/api/v1/auth/passkey/assert", {});
  const credential = await navigator.credentials.get({ publicKey: publicKeyFromEnvelope(begin) });
  await postJSON("/api/v1/auth/passkey/assert", {
    ceremony_id: begin.ceremony_id,
    response: credentialToJSON(credential),
  });
  return "Signed in. Continue to your application.";
}

document.addEventListener("click", (event) => {
  const target = event.target.closest("[data-passkey-action]");
  if (!target) return;
  event.preventDefault();
  const result = document.getElementById(target.dataset.passkeyResult || "login-result");
  cypraPasskey(target.dataset.passkeyAction, target)
    .then((message) => {
      if (target.dataset.continueUrl) {
        window.location.assign(target.dataset.continueUrl);
        return;
      }
      if (result) result.textContent = message;
    })
    .catch((err) => {
      if (result) result.textContent = classifyPasskeyError(err);
    });
});

// Export for tests in environments that load this as a module-like context. The
// no-op assignment is harmless in plain browser execution.
if (typeof window !== "undefined") {
  window.__cypraPasskeyInternals = { classifyPasskeyError, PasskeyError };
}
