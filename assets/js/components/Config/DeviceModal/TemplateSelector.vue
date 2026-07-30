<template>
	<FormRow :id="`${deviceType}Template`" :label="$t(`config.${deviceType}.template`)">
		<select
			v-if="isNew"
			:id="`${deviceType}Template`"
			v-model="modelProxy"
			class="form-select w-100"
			:disabled="disabled"
		>
			<template v-for="group in groups" :key="group.label">
				<optgroup
					v-if="group.options?.length"
					:label="$t(`config.${deviceType}.${group.label}`)"
				>
					<option v-for="option in group.options" :key="option.name" :value="option">
						{{ option.name }}
					</option>
				</optgroup>
			</template>
		</select>
		<div v-else class="d-flex gap-2 align-items-stretch">
			<input
				:id="`${deviceType}Template`"
				type="text"
				:value="productName || $t('config.general.customOption')"
				disabled
				class="form-control"
			/>
			<slot name="action" />
		</div>
	</FormRow>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import FormRow from "../FormRow.vue";
import { type DeviceType } from "@/types/evcc";

export interface TemplateOption {
	name: string;
	template: string;
}

export interface PrimaryOption {
	name: string;
	template: string;
}

export interface TemplateGroup {
	label: string;
	options: TemplateOption[];
}

export default defineComponent({
	name: "TemplateSelector",
	components: { FormRow },
	props: {
		deviceType: String as PropType<DeviceType>,
		isNew: Boolean,
		modelValue: String as PropType<string | null>,
		productName: String,
		groups: Array as PropType<TemplateGroup[]>,
		disabled: Boolean,
	},
	emits: ["update:modelValue", "change"],
	data() {
		return {
			// last user pick; the effective selection is derived in modelProxy
			selected: null as TemplateOption | null,
		};
	},
	computed: {
		flatOptions(): TemplateOption[] {
			return (this.groups ?? []).flatMap((g) => g.options ?? []);
		},
		modelProxy: {
			get(): TemplateOption | null {
				const options = this.flatOptions.filter((o) => o.template === this.modelValue);
				const sameNameOption = options.find((o) => o.name === this.selected?.name);
				return sameNameOption ?? options[0] ?? null;
			},
			set(option: TemplateOption | null) {
				this.selected = option;
				this.$emit("update:modelValue", option?.template ?? null);
				this.$emit("change", option?.template ?? null);
			},
		},
	},
	methods: {
		getProductName(): string {
			return this.modelProxy?.name || "";
		},
	},
});

export function customTemplateOption(name: string, template = "custom") {
	return { name, template, group: "" };
}
</script>
