export const metadata = {
  title: "Cypra Next.js Example",
  description: "Auth.js OIDC consumer for Cypra",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
