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
      },
    ],
    secret: process.env.AUTH_SECRET,
  });
