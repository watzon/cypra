import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function middleEllipsis(value: string, edge = 8) {
  if (value.length <= edge * 2 + 1) {
    return value;
  }
  return `${value.slice(0, edge)}...${value.slice(-edge)}`;
}

export function actorThemeEndpoint(actorKind: "tenant" | "instance") {
  return actorKind === "instance" ? "/api/v1/instance/admins/me" : "/api/v1/users/me";
}
