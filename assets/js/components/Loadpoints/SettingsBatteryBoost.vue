<template>
	<div>
		<h6>
			{{ $t("main.loadpointSettings.batteryUsage") }}
		</h6>

		<div class="mb-3 row" data-testid="battery-boost">
			<label :for="formId('batteryBoostLimit')" class="col-sm-4 col-form-label pt-0 pt-sm-2">
				{{ $t("main.loadpointSettings.batteryBoost.label") }}
			</label>
			<div class="col-sm-8 col-lg-4 pe-0 d-flex align-items-center">
				<select
					:id="formId('batteryBoostLimit')"
					v-model.number="selectedLimit"
					class="form-select form-select-sm"
					data-testid="battery-boost-limit"
					@change="handleLimitChange"
				>
					<option v-for="{ value, name } in limitOptions" :key="value" :value="value">
						{{ name }}
					</option>
				</select>
			</div>
			<div class="col-sm-8 offset-sm-4 mt-1">
				<small class="text-muted">{{ description }}</small>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import formatter from "@/mixins/formatter";

const insertSorted = (arr: number[], num: number) => {
	const uniqueSet = new Set(arr);
	uniqueSet.add(num);
	return [...uniqueSet].sort((a, b) => b - a);
};

export default defineComponent({
	mixins: [formatter],
	props: {
		formId: {
			type: Function as PropType<(s: string) => string>,
			default: (s: string) => s,
		},
		batteryBoostLimit: { type: Number, default: 100 },
	},
	emits: ["batteryboostlimit-updated"],
	data() {
		return {
			selectedLimit: this.batteryBoostLimit,
		};
	},
	computed: {
		description(): string {
			if (this.selectedLimit < 100) {
				return this.$t("main.loadpointSettings.batteryBoost.description", {
					limit: this.fmtPercentage(this.selectedLimit),
				});
			}
			return this.$t("main.loadpointSettings.batteryBoost.descriptionDisabled");
		},
		limitOptions() {
			// generate 5-step values: 100 (disabled), 95, 90, ..., 5, 0
			const values = [100];
			for (let i = 95; i >= 0; i -= 5) {
				values.push(i);
			}
			// insert current value if non-standard
			const opts = insertSorted(values, this.batteryBoostLimit);
			return opts.map((value) => ({
				value,
				name:
					value === 100
						? this.$t("main.loadpointSettings.batteryBoost.disabled")
						: `${value} %`,
			}));
		},
	},
	watch: {
		batteryBoostLimit(newVal: number) {
			this.selectedLimit = newVal;
		},
	},
	methods: {
		handleLimitChange() {
			this.$emit("batteryboostlimit-updated", this.selectedLimit);
		},
	},
});
</script>
