<template>
	<div :data-testid="`device-ref-${property}`">
		<DeviceRefBox v-if="modelValue" compact :title="title" :error="hasError" @edit="edit" />
		<div v-else class="position-relative d-inline-block">
			<button
				:id="id"
				type="button"
				class="d-flex btn btn-sm btn-outline-secondary border-0 align-items-center gap-2 evcc-gray"
				@click="edit"
			>
				<shopicon-regular-plus size="s" class="flex-shrink-0"></shopicon-regular-plus>
				{{ addLabel }}
			</button>
			<!-- invisible input over the button carries the native required validation -->
			<input
				v-if="required"
				class="position-absolute top-0 start-0 w-100 h-100 opacity-0 pe-none"
				type="text"
				tabindex="-1"
				aria-hidden="true"
				required
				:value="modelValue"
			/>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import DeviceRefBox from "./DeviceRefBox.vue";
import { createDeviceUtils } from "./DeviceModal";
import { getModal, openModal, topModal } from "@/configModal";
import type { DeviceType } from "@/types/evcc";
import "@h2d2/shopicons/es/regular/plus";

export default defineComponent({
	name: "PropertyDeviceField",
	components: { DeviceRefBox },
	props: {
		id: String,
		property: String,
		modelValue: { type: String, default: "" },
		deviceClass: { type: String as PropType<DeviceType>, required: true },
		usage: String,
		hasDeviceError: {
			type: Function as PropType<(type: DeviceType, name: string) => boolean>,
			default: () => false,
		},
		required: Boolean,
	},
	emits: ["update:modelValue"],
	data() {
		return { title: "", error: false };
	},
	computed: {
		hasError(): boolean {
			return this.error || this.hasDeviceError(this.deviceClass, this.modelValue);
		},
		deviceId(): number | undefined {
			return this.modelValue.startsWith("db:")
				? parseInt(this.modelValue.slice(3))
				: undefined;
		},
		addLabel(): string {
			const key = `config.deviceRef.add.${this.usage}`;
			return this.$te(key)
				? this.$t(key)
				: this.$t(`config.deviceRef.add.${this.deviceClass}`);
		},
	},
	watch: {
		modelValue: {
			immediate: true,
			handler() {
				this.loadTitle();
			},
		},
	},
	methods: {
		async loadTitle() {
			this.error = false;
			this.title = this.modelValue;
			if (this.deviceId === undefined) return;
			try {
				const device = await createDeviceUtils(this.deviceClass).load(this.deviceId);
				this.title =
					device.deviceTitle ||
					device.deviceProduct ||
					this.$t("config.general.customOption");
			} catch {
				this.error = true;
			}
		},
		async edit() {
			const cls = this.deviceClass;
			// a modal can't open itself, same-class nesting uses the second mounted instance
			const name = topModal()?.name === cls ? `sub${cls}` : cls;
			const type = this.usage || getModal(cls)?.type;
			const result = await openModal(name, { id: this.deviceId, type });
			if (result.action === "added" && result.name) {
				this.$emit("update:modelValue", result.name);
			} else if (result.action === "removed") {
				this.$emit("update:modelValue", "");
			}
		},
	},
});
</script>
