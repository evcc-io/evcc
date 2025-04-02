import { ComponentCustomProperties } from "vue";

declare module "vue" {
	interface ComponentCustomProperties {
		/**
		 * Whether experimental UI features should be shown.
		 */
		$hiddenFeatures: () => boolean;
		$refs: { [key: string]: HTMLElement | undefined };
	}
}
