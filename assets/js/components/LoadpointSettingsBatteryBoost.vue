<template>
	<div>
		<h6>
			{{ $t("main.loadpointSettings.batteryUsage") }}
			{{ mode }}
		</h6>

		<div class="mb-3 row">
			<label :for="formId('batteryBoost')" class="col-sm-4 col-form-label pt-0 pt-sm-2">
				{{ $t("main.loadpointSettings.batteryBoost.label") }}&nbsp;🧪
			</label>
			<div class="col-sm-8 pe-0">
				<div class="form-check form-switch my-1">
					<input
						:id="formId('batteryBoost')"
						v-model="selectedEnabled"
						class="form-check-input"
						type="checkbox"
						role="switch"
						:disabled="disabled"
						@change="handleEnabledChange"
					/>
					<label :for="formId('batteryBoost')" class="form-check-label">
						<i18n-t
							keypath="main.loadpointSettings.batteryBoost.description"
							tag="div"
							scope="global"
						>
							<template #limit>
								<div class="custom-select-inline" @click.prevent="() => {}">
									<CustomSelect
										:id="formId('batteryBoostLimit')"
										:options="limitOptions"
										:selected="selectedLimit"
										@change="handleLimitChange"
									>
										<span class="text-decoration-underline">
											{{ fmtPercentage(selectedLimit) }}
										</span>
									</CustomSelect>
								</div>
							</template>
						</i18n-t>
						<span v-if="selectedEnabled && !disabled" class="d-block text-primary">
							{{ $t("main.loadpointSettings.batteryBoost.once") }}
						</span>
						<small v-if="disabled" class="d-block">
							{{ $t("main.loadpointSettings.batteryBoost.mode") }}
						</small>
					</label>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
import CustomSelect from "./CustomSelect.vue";
import formatter from "../mixins/formatter";

export default {
	components: { CustomSelect },
	mixins: [formatter],
	props: {
		formId: Function,
		batterySoc: Number,
		mode: String,
		batteryBoost: Boolean,
		batteryBoostLimit: Number,
	},
	data() {
		return {
			selectedEnabled: this.batteryBoost,
			selectedLimit: this.batteryBoostLimit,
		};
	},
	watch: {
		batteryBoost(newVal) {
			this.selectedEnabled = newVal;
		},
		batteryBoostLimit(newVal) {
			this.selectedLimit = newVal;
		},
	},
	computed: {
		limitOptions() {
			return this.range(95, 0, -5).map((value) => this.limitOption(value));
		},
		disabled() {
			return ["off", "now"].includes(this.mode);
		},
	},
	methods: {
		handleEnabledChange() {
			this.$emit("batteryboost-updated", this.selectedEnabled);
		},
		handleLimitChange($event) {
			this.selectedLimit = parseInt($event.target.value);
			this.$emit("batteryboostlimit-updated", this.selectedLimit);
		},
		range(start, stop, step = -1) {
			return Array.from({ length: (stop - start) / step + 1 }, (_, i) => start + i * step);
		},
		limitOption(value) {
			return { value, name: this.fmtPercentage(value), disabled: value > this.batterySoc };
		},
	},
	emits: ["batteryboost-updated", "batteryboostlimit-updated"],
};
</script>

<style scoped>
.custom-select-inline {
	display: inline-block !important;
}
</style>
