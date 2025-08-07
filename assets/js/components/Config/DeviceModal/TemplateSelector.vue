<template>
	<FormRow :id="`${deviceType}Template`" :label="$t(`config.${deviceType}.template`)">
		<select
			v-if="isNew"
			:id="`${deviceType}Template`"
			ref="select"
			v-model="modelProxy"
			class="form-select w-100"
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
						:value="option.template"
					>
						{{ option.name }}
					</option>
				</optgroup>
			</template>
		</select>
		<input
			v-else
			:id="`${deviceType}Template`"
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
	},
	emits: ["update:modelValue", "change"],
	computed: {
		modelProxy: {
			get() {
				return this.modelValue;
			},
			set(value: string) {
				this.$emit("update:modelValue", value);
			},
		},
	},
	methods: {
		changed(e: Event) {
			this.$emit("change", e);
		},
		getProductName() {
			const select = this.$refs["select"] as HTMLSelectElement;
			return select.options[select.selectedIndex].text;
		},
	},
});

export function customTemplateOption(name: string, template = "custom") {
	return { name, template };
}
</script>
