<template>
	<div>
		<button
			class="form-select text-start text-nowrap"
			type="button"
			:id="id"
			data-bs-toggle="dropdown"
			aria-expanded="false"
			data-bs-auto-close="outside"
		>
			<slot></slot>
		</button>
		<ul class="dropdown-menu dropdown-menu-end" ref="dropdown" :aria-labelledby="id">
			<li v-for="option in options" :key="option.value" class="dropdown-item p-0">
				<label class="form-check px-3 py-2" :for="option.value">
					<input
						class="form-check-input ms-0 me-2"
						type="checkbox"
						:id="option.value"
						:value="option.value"
						v-model="internalValue"
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
	},
	emits: ["open"],
	data() {
		return {
			internalValue: [...this.value],
		};
	},
	mounted() {
		this.$refs.dropdown.addEventListener("show.bs.dropdown", this.open);
	},
	unmounted() {
		this.$refs.dropdown?.removeEventListener("show.bs.dropdown", this.open);
	},
	watch: {
		options() {
			// reposition on items change
			this.$nextTick(() => {
				Dropdown.getOrCreateInstance(this.$refs.dropdown).update();
			});
		},
		value(newValue) {
			this.internalValue = [...newValue];
		},
		internalValue() {
			this.$emit("update:modelValue", [...this.internalValue]);
		},
	},
	methods: {
		open() {
			this.$emit("open");
		},
	},
};
</script>
