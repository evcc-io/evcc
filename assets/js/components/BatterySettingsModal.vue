<template>
	<GenericModal
		id="batterySettingsModal"
		:title="$t('batterySettings.modalTitle')"
		size="xl"
		data-testid="battery-settings-modal"
		@open="modalVisible"
		@closed="modalInvisible"
	>
		<ul v-if="gridChargePossible" class="nav nav-tabs mb-4">
			<li class="nav-item">
				<a
					class="nav-link"
					:class="{ active: usageTabActive }"
					href="#"
					@click.prevent="showUsageTab"
				>
					{{ $t("batterySettings.usageTab") }}
				</a>
			</li>
			<li class="nav-item">
				<a
					class="nav-link"
					:class="{ active: gridTabActive }"
					href="#"
					@click.prevent="showGridTab"
				>
					{{ $t("batterySettings.gridChargeTab") }} 🧪
				</a>
			</li>
		</ul>

		<div v-show="usageTabActive" class="row">
			<p class="text-center text-md-start col-md-6 order-md-2 col-lg-3 order-lg-3 pt-lg-2">
				{{ $t("batterySettings.batteryLevel") }}:
				<strong>{{ fmtSoc(batterySoc) }}</strong>
				<small v-for="(line, index) in batteryDetails" :key="index" class="d-block">
					{{ line }}
				</small>
			</p>
			<div
				class="col-md-6 order-md-1 col-lg-3 order-lg-2 mb-5 mb-lg-0 battery justify-content-center justify-content-md-start"
			>
				<div class="batteryLimits">
					<CustomSelect
						id="batterySettingsBuffer"
						:options="bufferOptions"
						:selected="selectedBufferSoc"
						class="bufferSoc p-2 end-0"
						:class="{
							'bufferSoc--hidden': selectedBufferSoc === selectedPrioritySoc,
						}"
						:style="{ top: `${topHeight}%` }"
						@change="changeBufferSoc"
					>
						<span class="text-decoration-underline text-nowrap pe-none">
							{{ fmtSoc(selectedBufferSoc) }}
						</span>
					</CustomSelect>

					<CustomSelect
						id="batterySettingsPriority"
						:options="priorityOptions"
						:selected="selectedPrioritySoc"
						class="prioritySoc p-2 end-0"
						:style="{ top: `${100 - bottomHeight}%` }"
						@change="changePrioritySoc"
					>
						<span class="text-decoration-underline text-nowrap pe-none">
							{{ fmtSoc(selectedPrioritySoc) }}
						</span>
					</CustomSelect>
				</div>
				<div class="progress me-md-0">
					<div
						class="bg-dark-green progress-bar text-light align-items-center"
						role="button"
						:style="{ height: `${topHeight}%` }"
						@click="toggleBufferStart"
					>
						<shopicon-regular-lightning
							size="m"
							class="icon"
							:style="iconStyle(topHeight)"
						></shopicon-regular-lightning>
					</div>
					<div
						class="bg-darker-green progress-bar text-light align-items-center"
						:style="{ height: `${middleHeight}%` }"
					>
						<shopicon-regular-car3
							size="m"
							class="icon"
							:style="iconStyle(middleHeight)"
						></shopicon-regular-car3>
					</div>
					<div
						class="bg-darkest-green progress-bar text-light align-items-center"
						:style="{ height: `${bottomHeight}%` }"
					>
						<shopicon-regular-home
							size="m"
							class="icon"
							:style="iconStyle(bottomHeight)"
						></shopicon-regular-home>
					</div>
					<div
						class="batterySoc ps-0 bg-white pe-none"
						:style="{ top: `${100 - batterySoc}%` }"
					></div>
					<div
						class="bufferStartIndicator pe-none"
						:class="{
							'bufferStartIndicator--hidden':
								!selectedBufferStartSoc || selectedBufferSoc === 100,
						}"
						:style="{ top: `${bufferStartTop}%` }"
					>
						<div class="bufferStartIndicator__left"></div>
						<div class="bufferStartIndicator__right"></div>
					</div>
				</div>
			</div>
			<div class="col-md-12 order-md-3 col-lg-6 order-lg-1 legend pt-lg-2">
				<p class="d-flex">
					<shopicon-regular-lightning
						size="s"
						class="flex-shrink-0 me-2"
					></shopicon-regular-lightning>
					<span class="d-block">
						{{ $t("batterySettings.legendTopName") }}
						<i18n-t
							keypath="batterySettings.legendTopSubline"
							tag="small"
							class="d-block"
							scope="global"
						>
							<template #soc>
								<CustomSelect
									id="batterySettingsBufferTop"
									class="custom-select-inline"
									:options="bufferOptions"
									:selected="selectedBufferSoc"
									@change="changeBufferSoc"
								>
									<span class="text-decoration-underline">
										{{ fmtSoc(selectedBufferSoc) }}
									</span>
								</CustomSelect>
							</template>
						</i18n-t>

						<small class="d-block">
							{{ $t("batterySettings.legendTopAutostart") }}
							<CustomSelect
								id="batterySettingsBufferStart"
								class="custom-select-inline"
								:selected="selectedBufferStartSoc"
								:options="bufferStartOptions"
								@change="changeBufferStart"
							>
								<span class="text-decoration-underline">
									{{ selectedBufferStartName }}
								</span>
							</CustomSelect>
						</small>
					</span>
				</p>
				<p class="d-flex">
					<shopicon-regular-car3
						size="s"
						class="flex-shrink-0 me-2"
					></shopicon-regular-car3>
					<span class="d-block">
						{{ $t("batterySettings.legendMiddleName") }}
						<i18n-t
							keypath="batterySettings.legendMiddleSubline"
							tag="small"
							class="d-block"
							scope="global"
						>
							<template #soc>
								<CustomSelect
									id="batterySettingsPriorityMiddle"
									class="custom-select-inline"
									:options="priorityOptions"
									:selected="selectedPrioritySoc"
									@change="changePrioritySoc"
								>
									<span class="text-decoration-underline">
										{{ fmtSoc(selectedPrioritySoc) }}
									</span>
								</CustomSelect>
							</template>
						</i18n-t>
					</span>
				</p>
				<p class="d-flex">
					<shopicon-regular-home
						size="s"
						class="flex-shrink-0 me-2"
					></shopicon-regular-home>
					<span class="d-block">
						{{ $t("batterySettings.legendBottomName") }}
						<i18n-t
							keypath="batterySettings.legendBottomSubline"
							tag="small"
							class="d-block"
							scope="global"
						>
							<template #soc>
								<CustomSelect
									id="batterySettingsPriorityBottom"
									class="custom-select-inline"
									:options="priorityOptions"
									:selected="selectedPrioritySoc"
									@change="changePrioritySoc"
								>
									<span class="text-decoration-underline">
										{{ fmtSoc(selectedPrioritySoc) }}
									</span>
								</CustomSelect>
							</template>
						</i18n-t>
					</span>
				</p>
				<div v-if="controllable" class="form-check form-switch mt-4">
					<input
						id="batteryDischargeControl"
						:checked="batteryDischargeControl"
						class="form-check-input"
						type="checkbox"
						role="switch"
						@change="changeDischargeControl"
					/>
					<div class="form-check-label">
						<label for="batteryDischargeControl">
							{{ $t("batterySettings.discharge") }}
						</label>
					</div>
				</div>
			</div>
		</div>
		<SmartCostLimit
			v-if="gridChargePossible"
			v-show="gridTabActive"
			v-bind="smartCostLimitProps"
		/>
	</GenericModal>
</template>

<script>
import "@h2d2/shopicons/es/regular/lightning";
import "@h2d2/shopicons/es/regular/car3";
import "@h2d2/shopicons/es/regular/home";
import SmartCostLimit from "./SmartCostLimit.vue";
import CustomSelect from "./CustomSelect.vue";
import GenericModal from "./GenericModal.vue";
import formatter, { POWER_UNIT } from "../mixins/formatter";
import collector from "../mixins/collector";
import api from "../api";
import smartCostAvailable from "../utils/smartCostAvailable";

export default {
	name: "BatterySettingsModal",
	components: { SmartCostLimit, CustomSelect, GenericModal },
	mixins: [formatter, collector],
	props: {
		bufferSoc: Number,
		prioritySoc: Number,
		batterySoc: Number,
		bufferStartSoc: Number,
		batteryDischargeControl: Boolean,
		battery: { type: Array, default: () => [] },
		batteryGridChargeLimit: { type: Number, default: null },
		smartCostType: String,
		tariffGrid: Number,
		currency: String,
	},
	data: function () {
		return {
			isModalVisible: false,
			selectedBufferSoc: 100,
			selectedPrioritySoc: 0,
			selectedBufferStartSoc: 0,
			gridTabActive: false,
		};
	},
	computed: {
		usageTabActive() {
			return !this.gridTabActive;
		},
		priorityOptions() {
			const options = [];
			for (let i = 100; i >= 0; i -= 5) {
				// avoid intersection with buffer soc; allow everything if they touch
				const disabled =
					i > this.selectedBufferSoc &&
					!(this.selectedBufferSoc == this.selectedPrioritySoc);
				options.push({ value: i, name: this.fmtSoc(i), disabled });
			}
			return options;
		},
		controllable() {
			return this.battery.some(({ controllable }) => controllable);
		},
		gridChargePossible() {
			return (
				this.controllable &&
				this.isModalVisible &&
				this.smartCostAvailable &&
				this.$hiddenFeatures()
			);
		},
		bufferOptions() {
			const options = [];
			for (let i = 100; i >= 5; i -= 5) {
				options.push({
					value: i,
					name: this.fmtSoc(i),
					disabled: i < this.selectedPrioritySoc,
				});
			}
			return options;
		},
		bufferStartTop() {
			if (!this.selectedBufferStartSoc) return 0;
			return 100 - this.selectedBufferStartSoc;
		},
		bufferStartOptions() {
			const options = [];
			for (let i = 100; i >= this.bufferSoc; i -= 5) {
				options.push({
					value: i,
					name: this.getBufferStartName(i),
				});
			}
			options.push({
				value: 0,
				name: this.getBufferStartName(0),
			});
			return options;
		},
		bufferStartOption() {
			return this.bufferStartOptions.find((option) => this.bufferStartSoc >= option.value);
		},
		selectedBufferStartName() {
			return this.getBufferStartName(this.selectedBufferStartSoc);
		},
		topHeight() {
			return 100 - (this.bufferSoc || 100);
		},
		middleHeight() {
			return 100 - this.topHeight - this.bottomHeight;
		},
		bottomHeight() {
			return this.prioritySoc;
		},
		batteryDetails() {
			if (!Array.isArray(this.battery)) {
				return;
			}
			return this.battery
				.filter(({ capacity }) => capacity > 0)
				.map(({ soc = 0, capacity }) => {
					const multipleBatteries = this.battery.length > 1;
					const energy = this.fmtWh(
						(capacity / 100) * soc * 1e3,
						POWER_UNIT.KW,
						!multipleBatteries,
						1
					);
					const total = this.fmtWh(capacity * 1e3, POWER_UNIT.KW, true, 1);
					const name = multipleBatteries ? "↳ " : "";
					const formattedSoc = multipleBatteries ? ` (${this.fmtSoc(soc)})` : "";
					const formattedEnergy = this.$t("batterySettings.capacity", {
						energy,
						total,
					});
					return `${name}${formattedEnergy}${formattedSoc}`;
				});
		},
		smartCostLimitProps() {
			return {
				...this.collectProps(SmartCostLimit),
				smartCostLimit: this.batteryGridChargeLimit,
			};
		},
		smartCostAvailable() {
			return smartCostAvailable(this.smartCostType);
		},
	},
	watch: {
		prioritySoc(soc) {
			this.selectedPrioritySoc = soc;
		},
		bufferSoc(soc) {
			this.selectedBufferSoc = soc || 100;
		},
		bufferStartSoc(soc) {
			this.selectedBufferStartSoc = soc;
		},
	},
	mounted() {
		this.selectedBufferSoc = this.bufferSoc || 100;
		this.selectedPrioritySoc = this.prioritySoc;
		this.selectedBufferStartSoc = this.bufferStartSoc;
	},
	methods: {
		showGridTab() {
			this.gridTabActive = true;
		},
		showUsageTab() {
			this.gridTabActive = false;
		},
		modalVisible: function () {
			this.isModalVisible = true;
		},
		modalInvisible: function () {
			this.isModalVisible = false;
		},
		changeBufferStart($event) {
			this.setBufferStartSoc(parseInt($event.target.value, 10));
		},
		changePrioritySoc($event) {
			const soc = parseInt($event.target.value, 10);
			if (soc > (this.bufferSoc || 100)) {
				this.saveBufferSoc(soc);
				if (soc > this.bufferStartSoc && this.bufferStartSoc > 0) {
					this.setBufferStartSoc(soc);
				}
			} else {
				this.savePrioritySoc(soc);
			}
		},
		toggleBufferStart() {
			const options = this.bufferStartOptions.map((option) => option.value);
			const index = options.findIndex((value) => this.bufferStartSoc >= value);
			const nextIndex = index === 0 ? options.length - 1 : index - 1;
			this.setBufferStartSoc(options[nextIndex]);
		},
		async setBufferStartSoc(soc) {
			this.selectedBufferStartSoc = soc;
			await this.saveBufferStartSoc(this.selectedBufferStartSoc);
		},
		async changeBufferSoc($event) {
			const soc = parseInt($event.target.value, 10);
			if (soc > this.bufferStartSoc && this.bufferStartSoc > 0) {
				await this.setBufferStartSoc(soc);
			}
			await this.saveBufferSoc(soc);
		},
		async savePrioritySoc(soc) {
			this.selectedPrioritySoc = soc;
			try {
				await api.post(`prioritysoc/${encodeURIComponent(soc)}`);
			} catch (err) {
				console.error(err);
			}
		},
		async saveBufferSoc(soc) {
			this.selectedBufferSoc = soc;
			try {
				await api.post(`buffersoc/${encodeURIComponent(soc)}`);
			} catch (err) {
				console.error(err);
			}
		},
		async saveBufferStartSoc(soc) {
			try {
				await api.post(`bufferstartsoc/${encodeURIComponent(soc)}`);
			} catch (err) {
				console.error(err);
			}
		},
		iconStyle(height) {
			let scale = 1;
			if (height <= 10) scale = 0.75;
			if (height <= 5) scale = 0;
			return { transform: `scale(${scale})` };
		},
		fmtSoc(soc) {
			return this.fmtPercentage(soc);
		},
		async changeDischargeControl(e) {
			try {
				await api.post(`batterydischargecontrol/${e.target.checked ? "true" : "false"}`);
			} catch (err) {
				console.error(err);
			}
		},
		getBufferStartName(value) {
			const key = value === 0 ? "never" : value === 100 ? "full" : "above";
			const soc = this.fmtSoc(value);
			return this.$t(`batterySettings.bufferStart.${key}`, { soc });
		},
	},
};
</script>

<style scoped>
.battery {
	height: 285px;
	display: flex;
}

.batteryLimits {
	width: 50px;
	position: relative;
}

.bufferStart,
.bufferSoc,
.prioritySoc {
	position: absolute !important;
	transform: translateY(-50%);
	transition-property: top, opacity;
	transition-timing-function: linear;
	transition-duration: var(--evcc-transition-fast);
	opacity: 1;
}

.bufferStart--hidden,
.bufferSoc--hidden {
	opacity: 0;
	pointer-events: none;
}

.batterySoc,
.bufferStartIndicator {
	position: absolute;
	transition-property: top, opacity;
	transition-timing-function: linear;
	transition-duration: var(--evcc-transition-fast);
	transform: translateY(-50%);
}
.batterySoc {
	border-radius: 0.5rem;
	left: 0.5rem;
	right: 0.5rem;
	height: 0.5rem;
	opacity: 0.5;
}

.bufferStartIndicator {
	display: flex;
	justify-content: space-between;
	left: 0;
	right: 0;
}
.bufferStartIndicator--hidden {
	opacity: 0;
	transform: translateY(-50%);
}
.bufferStartIndicator__left,
.bufferStartIndicator__right {
	height: 0.5rem;
	width: 0.5rem;
	background-color: var(--evcc-box);
}
.bufferStartIndicator__left {
	border-radius: 0 0.5rem 0.5rem 0;
}
.bufferStartIndicator__right {
	border-radius: 0.5rem 0 0 0.5rem;
}
.progress {
	flex: 1;
	height: 100%;
	min-width: 100px;
	max-width: 130px;
	margin-right: 50px;
	flex-direction: column;
	position: relative;
	border-radius: 1rem;
	background-color: var(--evcc-box) !important;
}
.progress-bar {
	transition: height var(--evcc-transition-fast) linear;
}
.icon {
	transition: transform var(--evcc-transition-fast) linear;
	z-index: 1;
	border-radius: 0.5rem;
}
.custom-select-inline {
	display: inline-block !important;
}
</style>
