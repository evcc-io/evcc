<template>
	<div>
		<button
			:id="id"
			class="form-select text-start text-nowrap"
			type="button"
			data-bs-toggle="dropdown"
			aria-expanded="false"
			data-bs-auto-close="outside"
		>
			<slot></slot>
		</button>
		<ul ref="dropdown" class="dropdown-menu dropdown-menu-end" :aria-labelledby="id">
			<template v-if="selectAllLabel">
				<li class="dropdown-item p-0">
					<label class="form-check px-3 py-2">
						<input
							class="form-check-input ms-0 me-2"
							type="checkbox"
							value="all"
							:checked="allOptionsSelected"
							@change="toggleCheckAll()"
						/>
						<div class="form-check-label">{{ selectAllLabel }}</div>
					</label>
				</li>
				<li><hr class="dropdown-divider" /></li>
			</template>
			<li v-for="option in options" :key="option.value" class="dropdown-item p-0">
				<label class="form-check px-3 py-2 d-flex" :for="option.value">
					<input
						:id="option.value"
						v-model="internalValue"
						class="form-check-input ms-0 me-2"
						type="checkbox"
						:value="option.value"
					/>
					<div class="form-check-label">
						{{ option.name }}
					</div>
				</label>
			</li>
		</ul>
	</div>
</template>

<script>
import Dropdown from "bootstrap/js/dist/dropdown";

export default {
	name: "MultiSelect",
	props: {
		id: String,
		value: { type: Array, default: () => [] },
		options: { type: Array, default: () => [] },
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
			handler(newOptions) {
				// If value is empty, set internalValue to include all options
				if (this.value.length === 0) {
					this.internalValue = newOptions.map((option) => option.value);
				} else {
					// Otherwise, keep selected options that still exist in the new options
					this.internalValue = this.internalValue.filter((value) =>
						newOptions.some((option) => option.value === value)
					);
				}
				this.$nextTick(() => {
					Dropdown.getOrCreateInstance(this.$refs.dropdown).update();
				});
			},
		},
		value: {
			immediate: true,
			handler(newValue) {
				this.internalValue =
					newValue.length === 0 && this.options.length > 0
						? this.options.map((o) => o.value)
						: [...newValue];
			},
		},
		internalValue(newValue) {
			if (this.allOptionsSelected || this.noneSelected) {
				this.$emit("update:modelValue", []);
			} else {
				this.$emit("update:modelValue", newValue);
			}
		},
	},
	mounted() {
		this.$refs.dropdown.addEventListener("show.bs.dropdown", this.open);
	},
	unmounted() {
		this.$refs.dropdown?.removeEventListener("show.bs.dropdown", this.open);
	},
	methods: {
		open() {
			this.$emit("open");
		},
		toggleCheckAll() {
			if (this.allOptionsSelected) {
				this.internalValue = [];
			} else {
				this.internalValue = this.options.map((option) => option.value);
			}
		},
	},
};
</script>
