<template>
	<FormRow :id="`${deviceType}Template`" :label="$t(`config.${deviceType}.template`)">
		<select
			v-if="isNew"
			:id="`${deviceType}Template`"
			ref="select"
			v-model="localValue"
			class="form-select w-100"
			@change="changed"
		>
			<template v-if="primaryOption">
				<option :value="primaryOption.template">
					{{ primaryOption.name }}
				</option>
				<option disabled>----------</option>
			</template>

			<template v-for="group in groups" :key="group.label">
				<optgroup
					v-if="group.options?.length"
					:label="$t(`config.${deviceType}.${group.label}`)"
				>
					<option
						v-for="option in group.options"
						:key="option.name"
						:value="option.template"
					>
						{{ option.name }}
					</option>
				</optgroup>
			</template>

			<option disabled>----------</option>
			<option value="custom">{{ $t("config.general.customOption") }}</option>
		</select>
		<input
			v-else
			type="text"
			:value="productName || $t('config.general.customOption')"
			disabled
			class="form-control w-100"
		/>
	</FormRow>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import FormRow from "../FormRow.vue";
import type { DeviceType } from "@/types/evcc";

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
		primaryOption: Object as PropType<PrimaryOption>,
	},
	emits: ["update:modelValue", "change"],
	data() {
		return {
			localValue: this.modelValue,
		};
	},
	watch: {
		modelValue() {
			this.localValue = this.modelValue;
		},
		localValue() {
			this.$emit("update:modelValue", this.localValue);
		},
	},
	beforeMount() {
		this.localValue = this.modelValue;
	},
	methods: {
		changed() {
			this.$emit("change");
		},
		getProductName() {
			const select = this.$refs["select"] as HTMLSelectElement;
			return select.options[select.selectedIndex].text;
		},
	},
});
</script>
