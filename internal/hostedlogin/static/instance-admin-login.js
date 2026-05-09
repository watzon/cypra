(function () {
  function b64urlToBuffer(value) {
    const padded =
      value.replace(/-/g, "+").replace(/_/g, "/") + "===".slice((value.length + 3) % 4);
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
    for (const descriptor of options.allowCredentials || []) {
      descriptor.id = b64urlToBuffer(descriptor.id);
    }
    return options;
  }

  function credentialToJSON(credential) {
    const response = credential.response;
    return {
      id: credential.id,
      rawId: bufferToB64url(credential.rawId),
      type: credential.type,
      authenticatorAttachment: credential.authenticatorAttachment,
      clientExtensionResults: credential.getClientExtensionResults?.() || {},
      response: {
        clientDataJSON: bufferToB64url(response.clientDataJSON),
        authenticatorData: bufferToB64url(response.authenticatorData),
        signature: bufferToB64url(response.signature),
        userHandle: response.userHandle ? bufferToB64url(response.userHandle) : null,
      },
    };
  }

  async function postJSON(path, body) {
    const response = await window.fetch(path, {
      method: "POST",
      headers: { "Content-Type": "application/json", Accept: "application/json" },
      credentials: "same-origin",
      body: JSON.stringify(body),
    });
    const data = await response.json().catch(() => ({}));
    if (!response.ok) {
      throw new Error(data.error || "auth.failed");
    }
    return data;
  }

  function setError(message) {
    const slot = document.getElementById("instance-admin-error");
    if (!slot) return;
    slot.textContent = message || "";
  }

  async function runPasskey() {
    setError("");
    try {
      const begin = await postJSON("/login/instance-admin/passkey", {});
      const publicKey = publicKeyFromEnvelope(begin);
      const credential = await navigator.credentials.get({ publicKey });
      if (!credential) throw new Error("auth.cancelled");
      await postJSON("/login/instance-admin/passkey", {
        ceremony_id: begin.ceremony_id,
        response: credentialToJSON(credential),
      });
      window.location.assign("/dashboard");
    } catch (err) {
      setError("Sign-in failed. Try again or use a backup code.");
      // eslint-disable-next-line no-console
      console.error(err);
    }
  }

  async function runBackupCode(event) {
    event.preventDefault();
    setError("");
    const email = (document.getElementById("instance-admin-email")?.value || "").trim();
    const code = (event.target.elements.code?.value || "").trim();
    if (!email || !code) {
      setError("Enter your admin email and backup code.");
      return;
    }
    try {
      await postJSON("/login/instance-admin/backup-code", { email, code });
      window.location.assign("/dashboard");
    } catch (err) {
      setError("That backup code didn't work.");
      // eslint-disable-next-line no-console
      console.error(err);
    }
  }

  document.addEventListener("DOMContentLoaded", function () {
    document.getElementById("instance-admin-passkey-button")?.addEventListener("click", runPasskey);
    document
      .getElementById("instance-admin-backup-form")
      ?.addEventListener("submit", runBackupCode);
  });
})();
