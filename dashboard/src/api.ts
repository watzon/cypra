export interface VersionResponse {
  version: string;
  commit: string;
}

export async function getVersion(): Promise<VersionResponse> {
  const response = await fetch("/api/v1/version");
  if (!response.ok) {
    throw new Error("version request failed");
  }
  return (await response.json()) as VersionResponse;
}

export async function getReady(): Promise<boolean> {
  const response = await fetch("/readyz");
  return response.ok;
}
