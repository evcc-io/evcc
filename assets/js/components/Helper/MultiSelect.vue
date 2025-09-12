<template>
	<div>
		<button
			:id="id"
			class="form-select text-start text-nowrap"
			type="button"
			data-bs-toggle="dropdown"
			aria-expanded="false"
			data-bs-auto-close="outside"
			tabindex="0"
		>
			<slot></slot>
		</button>
		<ul ref="dropdown" class="dropdown-menu dropdown-menu-end" :aria-labelledby="id">
			<template v-if="selectAllLabel">
				<li class="dropdown-item p-0">
					<label class="form-check px-3 py-2" :for="formId('all')">
						<input
							:id="formId('all')"
							class="form-check-input ms-0 me-2"
							type="checkbox"
							value="all"
							tabindex="0"
							:checked="allOptionsSelected"
							@change="toggleCheckAll()"
						/>
						<div class="form-check-label">{{ selectAllLabel }}</div>
					</label>
				</li>
				<li><hr class="dropdown-divider" /></li>
			</template>
			<li v-for="option in options" :key="option.value" class="dropdown-item p-0">
				<label class="form-check px-3 py-2 d-flex" :for="formId(option.value)">
					<input
						:id="formId(option.value)"
						v-model="internalValue"
						class="form-check-input ms-0 me-2"
						type="checkbox"
						:value="option.value"
						tabindex="0"
					/>
					<div class="form-check-label">
						{{ option.name }}
					</div>
				</label>
			</li>
		</ul>
	</div>
</template>

<script lang="ts">
import Dropdown from "bootstrap/js/dist/dropdown";
import deepEqual from "@/utils/deepEqual";
import { defineComponent, type PropType } from "vue";
import type { SelectOption } from "@/types/evcc";

export default defineComponent({
	name: "MultiSelect",
	props: {
		id: String,
		value: { type: Array as PropType<string[] | number[]>, default: () => [] },
		options: { type: Array as PropType<SelectOption<string | number>[]>, default: () => [] },
		selectAllLabel: String,
	},
	emits: ["open", "update:modelValue"],
	data() {
		return {
			internalValue: [...this.value],
		};
	},
	computed: {
		allOptionsSelected() {
			return this.internalValue.length === this.options.length;
		},
		noneSelected() {
			return this.internalValue.length === 0;
		},
	},
	watch: {
		options: {
			immediate: true,
			handler(newOptions: SelectOption<string>[]) {
				this.internalValue = this.internalValue.filter((value) =>
					newOptions.some((option) => option.value === value)
				);
				this.$nextTick(() => {
					Dropdown.getOrCreateInstance(this.$refs["dropdown"] as HTMLElement).update();
				});
			},
		},
		internalValue(newValue, oldValue) {
			if (deepEqual(newValue, oldValue)) return;
			this.$emit("update:modelValue", newValue);
		},
	},
	mounted() {
		this.$refs["dropdown"]?.addEventListener("show.bs.dropdown", this.open);
	},
	unmounted() {
		this.$refs["dropdown"]?.removeEventListener("show.bs.dropdown", this.open);
	},
	methods: {
		open() {
			this.$emit("open");
		},
		formId(name: string | number) {
			return `${this.id}-${name}`;
		},
		toggleCheckAll() {
			if (this.allOptionsSelected) {
				this.internalValue = [];
			} else {
				this.internalValue = this.options.map((option) => option.value);
			}
		},
	},
});
</script>
