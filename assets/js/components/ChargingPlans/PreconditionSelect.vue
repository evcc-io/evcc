<template>
	<!-- small screen (select, description), large screen (checkbox only) -->
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
			<select
				:id="`${id}-sm`"
				v-model="localValue"
				class="form-select mx-0 d-lg-none"
				:data-testid="`${testid}-select`"
				@change="onChange"
			>
				<option :value="0">
					{{ $t("main.chargingPlan.preconditionOptionNo") }}
				</option>
				<option v-for="opt in options" :key="opt.value" :value="opt.value">
					{{ opt.name }}
				</option>
			</select>

			<small v-if="enabled" class="d-block d-lg-none mt-1 mb-2">
				{{ $t("main.chargingPlan.preconditionDescription", { duration: valueFmt }) }}
			</small>
		</div>
	</template>

	<!-- large screen (description with select) -->
	<template v-else>
		<p v-if="enabled" class="m-2 d-none d-lg-block">
			<strong>{{ $t("main.chargingPlan.preconditionLong") }}: </strong>
			<i18n-t tag="span" scope="global" keypath="main.chargingPlan.preconditionDescription">
				<template #duration>
					<CustomSelect
						:id="`${id}-lg`"
						class="d-inline-flex"
						:options="options"
						:selected="localValue"
						:data-testid="`${testid}-lg-select`"
						@change="onChange"
					>
						<u>{{ valueFmt }}</u>
					</CustomSelect>
				</template>
			</i18n-t>
		</p>
	</template>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import CustomSelect from "../Helper/CustomSelect.vue";
import formatter from "@/mixins/formatter";

export default defineComponent({
	name: "PreconditionSelect",
	components: { CustomSelect },
	mixins: [formatter],
	props: {
		id: String,
		modelValue: { type: Number, default: 0 },
		testid: String,
		descriptionLgOnly: Boolean,
	},
	emits: ["update:modelValue"],
	data() {
		return {
			localValue: this.modelValue,
		};
	},
	computed: {
		enabled() {
			return this.localValue > 0;
		},
		options() {
			const HOUR = 60 * 60;
			const HALF_HOUR = 0.5 * HOUR;
			const ONE_HOUR = 1 * HOUR;
			const TWO_HOURS = 2 * HOUR;
			const EVERYTHING = 7 * 24 * HOUR;

			const options = [HALF_HOUR, ONE_HOUR, TWO_HOURS, EVERYTHING];

			// support custom values (via API)
			if (this.localValue && !options.includes(this.localValue)) {
				options.push(this.localValue);
			}

			return options.map((s) => ({
				value: s,
				name:
					s === EVERYTHING
						? this.$t("main.chargingPlan.preconditionOptionAll")
						: this.fmtDurationLong(s),
			}));
		},
		valueFmt() {
			return this.options.find((o: { value: number }) => o.value === this.localValue)?.name;
		},
	},
	watch: {
		modelValue(newValue) {
			this.localValue = newValue;
		},
	},
	methods: {
		toggle() {
			const DEFAULT_PRECONDITION = 3600; // 1 hour
			const newValue = this.localValue > 0 ? 0 : DEFAULT_PRECONDITION;
			this.localValue = newValue;
			this.$emit("update:modelValue", newValue);
		},
		onChange(event: Event) {
			const value = parseInt((event.target as HTMLSelectElement).value);
			this.$emit("update:modelValue", value);
		},
	},
});
</script>
<style scoped>
.form-check-input {
	margin-left: 0.1rem;
}
</style>
