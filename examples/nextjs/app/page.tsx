export default function Home() {
  return (
    <main style={{ fontFamily: "ui-sans-serif, system-ui", padding: 48 }}>
      <h1>Cypra + Next.js</h1>
      <p>
        Configure Auth.js with the tenant issuer in <code>CYPRA_ISSUER</code>, then start a sign-in
        flow against Cypra.
      </p>
      <a href="/api/auth/signin">Sign in with Cypra</a>
    </main>
  );
}
