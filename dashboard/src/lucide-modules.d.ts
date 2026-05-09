declare module "lucide-react/dist/esm/icons/*.mjs" {
  import type { ForwardRefExoticComponent, RefAttributes, SVGProps } from "react";

  const Icon: ForwardRefExoticComponent<
    Omit<SVGProps<SVGSVGElement>, "ref"> & RefAttributes<SVGSVGElement>
  >;
  export default Icon;
}

declare module "*.css?raw" {
  const content: string;
  export default content;
}

declare module "node:fs" {
  export function readFileSync(path: string, encoding: "utf8"): string;
}
