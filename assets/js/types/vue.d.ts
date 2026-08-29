// oxlint-disable-next-line typescript/no-unused-vars
import type { ComponentCustomProperties } from "vue";

declare module "vue" {
  interface ComponentCustomProperties {
    $refs: { [key: string]: HTMLElement | undefined };
  }
}
