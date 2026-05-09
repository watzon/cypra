import { Auth } from "@auth/core";

export const auth = (request: Request) =>
  Auth(request, {
    providers: [
      {
        id: "cypra",
        name: "Cypra",
        type: "oidc",
        issuer: process.env.CYPRA_ISSUER,
        clientId: process.env.CYPRA_CLIENT_ID,
        clientSecret: process.env.CYPRA_CLIENT_SECRET,
        idToken: false,
      },
    ],
    basePath: "/api/auth",
    secret: process.env.AUTH_SECRET,
    trustHost: true,
    callbacks: {
      jwt({ token, profile }) {
        if (profile?.sub) token.cypraSub = profile.sub;
        if (profile?.email) token.cypraEmail = profile.email;
        return token;
      },
      session({ session, token }) {
        return {
          ...session,
          cypra: {
            sub: token.cypraSub ?? token.sub,
            email: token.cypraEmail ?? token.email,
          },
        };
      },
    },
  });
