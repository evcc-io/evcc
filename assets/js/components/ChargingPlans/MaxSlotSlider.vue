<template>
	<!-- Compact inline version -->
	<div v-if="compact" class="d-flex align-items-center gap-2 w-100">
		<input
			:id="id"
			v-model.number="localValue"
			type="range"
			min="0"
			max="5"
			step="1"
			class="form-range flex-grow-1"
			:data-testid="`${testid}-slider`"
			@input="onChange"
		/>
		<span class="text-nowrap" style="min-width: 2rem; text-align: center">{{ valueFmt }}</span>
	</div>

	<!-- Full version for large screen description -->
	<template v-else>
		<template v-if="!descriptionLgOnly">
			<div class="form-check d-none d-lg-block" :style="{ padding: '2px' }">
				<input
					:id="id"
					class="form-check-input d-none d-lg-block"
					type="checkbox"
					tabindex="0"
					:data-testid="`${testid}-lg-toggle`"
					:checked="enabled"
					@change="toggle"
				/>
			</div>

			<div class="d-flex flex-column w-100">
				<div class="d-lg-none mb-2">
					<div class="d-flex align-items-center gap-2">
						<input
							:id="`${id}-sm-slider`"
							v-model.number="localValue"
							type="range"
							class="form-range flex-grow-1"
							min="0"
							max="5"
							step="1"
							:data-testid="`${testid}-slider`"
							:disabled="!enabled"
							@input="onChange"
						/>
						<span class="text-nowrap" style="min-width: 3rem">{{ valueFmt }}</span>
					</div>
				</div>

				<small class="d-block d-lg-none mt-1 mb-2">
					{{ $t("main.chargingPlan.maxSlotsDescription", { windows: valueFmt }) }}
				</small>
			</div>
		</template>

		<template v-else>
			<p class="m-2 d-none d-lg-block">
				<strong>{{ $t("main.chargingPlan.maxSlotsLong") }}: </strong>
				<span>
					{{ $t("main.chargingPlan.maxSlotsDescription", { windows: valueFmt }) }}
				</span>
				<div class="d-flex align-items-center gap-2 mt-2">
					<input
						:id="`${id}-lg-slider`"
						v-model.number="localValue"
						type="range"
						class="form-range flex-grow-1"
						min="0"
						max="5"
						step="1"
						:data-testid="`${testid}-lg-slider`"
						@input="onChange"
					/>
					<span class="text-nowrap" style="min-width: 3rem">{{ valueFmt }}</span>
				</div>
			</p>
		</template>
	</template>
</template>

<script lang="ts">
import { defineComponent } from "vue";

export default defineComponent({
	name: "MaxSlotSlider",
	props: {
	  id: String,
	  modelValue: { type: Number, default: 0 },
	  testid: String,
	  descriptionLgOnly: Boolean,
	  compact: { type: Boolean, default: false },
	  defaultValue: { type: Number, default: 1 }
	},
	emits: ["update:modelValue"],
	data() {
	  return {
	    //localValue: this.modelValue || this.defaultValue,
		localValue: this.modelValue !== undefined ? this.modelValue : this.defaultValue,
	  };
	},
	computed: {
		enabled() {
			return this.localValue > 0;
		},
		valueFmt() {
			return String(this.localValue);
		},
	},
	watch: {
		modelValue(newValue) {
			this.localValue = newValue;
		},
	},
	methods: {
		toggle() {
			const DEFAULT_MAX_WINDOWS = 1;
			const newValue = this.localValue > 0 ? 0 : DEFAULT_MAX_WINDOWS;
			this.localValue = newValue;
			this.$emit("update:modelValue", newValue);
		},
		onChange() {
			this.$emit("update:modelValue", this.localValue);
		},
	},
});
</script>

<style scoped>
.form-check-input {
	margin-left: 0.1rem;
}

.form-range {
	cursor: pointer;
}

.form-range:disabled {
	cursor: not-allowed;
	opacity: 0.5;
}
</style>
