// eslint-disable-next-line @typescript-eslint/no-unused-vars
import type { ComponentCustomProperties } from "vue";

declare module "vue" {
  interface ComponentCustomProperties {
    $refs: { [key: string]: HTMLElement | undefined };
  }
}
