<template>
	<FormRow :id="`${deviceType}Template`" :label="$t(`config.${deviceType}.template`)">
		<select
			v-if="isNew"
			:id="`${deviceType}Template`"
			v-model="selected"
			class="form-select w-100"
			:disabled="disabled"
			@change="changed"
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
			selected: null as TemplateOption | null,
		};
	},
	computed: {
		flatOptions(): TemplateOption[] {
			return (this.groups ?? []).flatMap((g) => g.options ?? []);
		},
	},
	watch: {
		modelValue: "syncFromModel",
		groups: { handler: "syncFromModel", deep: true },
	},
	created() {
		this.syncFromModel();
	},
	methods: {
		syncFromModel() {
			const options = this.flatOptions.filter((o) => o.template === this.modelValue);
			const sameNameOption = options.find((o) => o.name === this.selected?.name);
			this.selected = sameNameOption ?? options[0] ?? null;
		},
		changed(e: Event) {
			this.selected = this.flatOptions[(e.target as HTMLSelectElement).selectedIndex] ?? null;
			this.$emit("update:modelValue", this.selected?.template ?? null);
			this.$emit("change", this.selected?.template ?? null);
		},
		getProductName() {
			return this.selected?.name || "";
		},
	},
});

export function customTemplateOption(name: string, template = "custom") {
	return { name, template, group: "" };
}
</script>
