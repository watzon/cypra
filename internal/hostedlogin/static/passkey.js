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
  if (!options) throw new Error("missing public key options");
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

async function postJSON(path, body) {
  const response = await window.fetch(path, {
    method: "POST",
    headers: { "Content-Type": "application/json", Accept: "application/json" },
    credentials: "same-origin",
    body: JSON.stringify(body),
  });
  const data = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(data.detail || data.error || "passkey request failed");
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
    throw new Error("passkeys unavailable");
  if (action === "register-account") {
    const email = emailFor(target);
    if (!email) throw new Error("missing email");
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
    if (!userId) throw new Error("missing user id");
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
    .catch(() => {
      if (result) result.textContent = "Sign in didn't work. Check your details and try again.";
    });
});
