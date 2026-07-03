<template>
	<FormRow :id="`${deviceType}Template`" :label="$t(`config.${deviceType}.template`)">
		<select
			v-if="isNew"
			:id="`${deviceType}Template`"
			ref="select"
			v-model="modelProxy"
			class="form-select w-100"
			:disabled="disabled"
			@change="changed"
		>
			<template v-for="group in groups" :key="group.label">
				<optgroup
					v-if="group.options?.length"
					:label="$t(`config.${deviceType}.${group.label}`)"
				>
					<option
						v-for="option in group.options"
						:key="option.name"
						:value="optionKey(option)"
					>
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
			// product-level selection (`template\tname`); the template alone is ambiguous
			// since multiple products may share the same template
			selection: null as string | null,
		};
	},
	computed: {
		modelProxy: {
			get(): string | null {
				if (this.selection?.split("\t")[0] === this.modelValue) {
					return this.selection;
				}
				// template set externally: select its first product
				const match = this.allOptions.find((o) => o.template === this.modelValue);
				return match ? this.optionKey(match) : null;
			},
			set(value: string) {
				this.selection = value;
				this.$emit("update:modelValue", value.split("\t")[0]);
			},
		},
		allOptions(): TemplateOption[] {
			return (this.groups ?? []).flatMap((g) => g.options ?? []);
		},
	},
	methods: {
		optionKey(option: TemplateOption): string {
			return `${option.template}\t${option.name}`;
		},
		changed(e: Event) {
			this.$emit("change", e);
		},
		getProductName() {
			return this.modelProxy?.split("\t")[1] || "";
		},
	},
});

export function customTemplateOption(name: string, template = "custom") {
	return { name, template, group: "" };
}
</script>
