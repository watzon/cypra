async function cypraPasskey(action, rpId) {
  if (!window.PublicKeyCredential || !navigator.credentials) return;
  const challenge = new Uint8Array(32);
  window.crypto.getRandomValues(challenge);
  if (action === "create") {
    await navigator.credentials.create({
      publicKey: {
        challenge,
        rp: { name: "Cypra", id: rpId },
        user: { id: challenge, name: "user", displayName: "User" },
        pubKeyCredParams: [{ type: "public-key", alg: -7 }],
      },
    });
    return;
  }
  await navigator.credentials.get({ publicKey: { challenge, rpId, allowCredentials: [] } });
}

document.addEventListener("click", (event) => {
  const target = event.target.closest("[data-passkey-action]");
  if (!target) return;
  event.preventDefault();
  cypraPasskey(target.dataset.passkeyAction, target.dataset.rpId).catch(() => {
    const result = document.getElementById("login-result");
    if (result) result.textContent = "Sign in didn't work. Check your details and try again.";
  });
});
