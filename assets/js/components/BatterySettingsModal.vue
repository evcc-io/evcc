<template>
	<Teleport to="body">
		<div
			id="batterySettingsModal"
			ref="modal"
			class="modal fade text-dark"
			data-bs-backdrop="true"
			tabindex="-1"
			role="dialog"
			aria-hidden="true"
		>
			<div class="modal-dialog modal-dialog-centered" role="document">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">
							{{ $t("batterySettings.modalTitle") }}
						</h5>
						<button
							type="button"
							class="btn-close"
							data-bs-dismiss="modal"
							aria-label="Close"
						></button>
					</div>
					<div class="modal-body">
						<div
							class="d-flex justify-content-between align-items-center align-items-sm-start pb-2 flex-column flex-sm-row-reverse"
						>
							<div class="battery mb-5 mb-sm-3 me-5 me-sm-0">
								<div class="batteryLimits">
									<label
										class="bufferSoc p-2 end-0"
										role="button"
										:class="{
											'bufferSoc--hidden':
												selectedBufferSoc === selectedPrioritySoc,
										}"
										:style="{ top: `${topHeight}%` }"
									>
										<select
											:value="selectedBufferSoc"
											class="custom-select"
											@change="changeBufferSoc"
										>
											<option
												v-for="{ value, name, disabled } in bufferOptions"
												:key="value"
												:value="value"
												:disabled="disabled"
											>
												{{ name }}
											</option>
										</select>
										<span class="text-decoration-underline text-nowrap pe-none">
											{{ fmtSoc(selectedBufferSoc) }}
										</span>
									</label>
									<label
										class="prioritySoc p-2 end-0"
										role="button"
										:style="{ top: `${100 - bottomHeight}%` }"
									>
										<select
											:value="selectedPrioritySoc"
											class="custom-select"
											@change="changePrioritySoc"
										>
											<option
												v-for="{ value, name, disabled } in priorityOptions"
												:key="value"
												:value="value"
												:disabled="disabled"
											>
												{{ name }}
											</option>
										</select>
										<span class="text-decoration-underline text-nowrap pe-none">
											{{ fmtSoc(selectedPrioritySoc) }}
										</span>
									</label>
								</div>
								<div class="progress">
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
												!selectedBufferStartSoc ||
												selectedBufferSoc === 100,
										}"
										:style="{ top: `${bufferStartTop}%` }"
									>
										<div class="bufferStartIndicator__left"></div>
										<div class="bufferStartIndicator__right"></div>
									</div>
								</div>
							</div>
							<div class="me-sm-4">
								<p>
									{{ $t("batterySettings.batteryLevel") }}:
									<strong>{{ fmtSoc(batterySoc) }}</strong>
									<small
										v-for="(line, index) in batteryDetails"
										:key="index"
										class="d-block"
									>
										{{ line }}
									</small>
								</p>
								<p>{{ $t("batterySettings.legendTitle") }}</p>
								<p class="d-flex">
									<shopicon-regular-lightning
										size="s"
										class="flex-shrink-0 me-2"
									></shopicon-regular-lightning>
									<span class="d-block">
										{{ $t("batterySettings.legendTopName") }}
										<small v-if="selectedBufferSoc == 100" class="d-block">
											{{ $t("batterySettings.legendTopSubline") }}
										</small>
										<small v-else class="d-block">
											{{ $t("batterySettings.legendTopAutostart") }}
											<label
												for="bufferStartSelect"
												class="position-relative d-block"
											>
												<select
													id="bufferStartSelect"
													class="custom-select"
													@change="changeBufferStart"
												>
													<option
														v-for="option in [
															'never',
															'full',
															'medium',
															'low',
														]"
														:key="option"
														:value="option"
													>
														{{
															$t(
																`batterySettings.bufferStart.${option}`
															)
														}}
													</option>
												</select>
												<span class="text-decoration-underline">
													{{
														$t(
															`batterySettings.bufferStart.${bufferStartOption}`
														)
													}}
												</span>
											</label>
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
										<small class="d-block">
											{{ $t("batterySettings.legendMiddleSubline") }}
										</small>
									</span>
								</p>
								<p class="d-flex">
									<shopicon-regular-home
										size="s"
										class="flex-shrink-0 me-2"
									></shopicon-regular-home>
									<span class="d-block">
										{{ $t("batterySettings.legendBottomName") }}
										<small class="d-block">
											{{ $t("batterySettings.legendBottomSubline") }}
										</small>
									</span>
								</p>
								<p class="small text-muted">
									<strong class="text-evcc">
										{{ $t("batterySettings.disclaimerHint") }}
									</strong>
									{{ $t("batterySettings.disclaimerText") }}
								</p>
							</div>
						</div>
						<FormRow
							v-if="controllable"
							id="batteryDischargeControl"
							:label="`${$t('batterySettings.control')}`"
						>
							<div class="form-check form-switch col-form-label">
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
						</FormRow>
					</div>
				</div>
			</div>
		</div>
	</Teleport>
</template>

<script>
import "@h2d2/shopicons/es/regular/lightning";
import "@h2d2/shopicons/es/regular/car3";
import "@h2d2/shopicons/es/regular/home";
import formatter from "../mixins/formatter";
import FormRow from "./FormRow.vue";

import api from "../api";

export default {
	name: "BatterySettingsModal",
	components: { FormRow },
	mixins: [formatter],
	props: {
		bufferSoc: Number,
		prioritySoc: Number,
		batterySoc: Number,
		bufferStartSoc: Number,
		batteryDischargeControl: Boolean,
		battery: { type: Array, default: () => [] },
	},
	data: function () {
		return {
			isModalVisible: false,
			selectedBufferSoc: 100,
			selectedPrioritySoc: 0,
			selectedBufferStartSoc: 0,
		};
	},
	computed: {
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
		topHeight() {
			return 100 - (this.bufferSoc || 100);
		},
		middleHeight() {
			return 100 - this.topHeight - this.bottomHeight;
		},
		bottomHeight() {
			return this.prioritySoc;
		},
		bufferStartOption() {
			if (!this.selectedBufferStartSoc) return "never";
			if (this.selectedBufferStartSoc >= this.bufferStartOptionToSoc("full")) return "full";
			if (this.selectedBufferStartSoc >= this.bufferStartOptionToSoc("medium"))
				return "medium";
			return "low";
		},
		batteryDetails() {
			if (!Array.isArray(this.battery)) {
				return;
			}
			return this.battery
				.filter(({ capacity }) => capacity > 0)
				.map(({ soc, capacity }) => {
					const multipleBatteries = this.battery.length > 1;
					const energy = this.fmtKWh(
						(capacity / 100) * soc * 1e3,
						true,
						!multipleBatteries,
						1
					);
					const total = this.fmtKWh(capacity * 1e3, true, true, 1);
					const name = multipleBatteries ? "↳ " : "";
					const formattedSoc = multipleBatteries ? ` (${this.fmtSoc(soc)})` : "";
					const formattedEnergy = this.$t("batterySettings.capacity", {
						energy,
						total,
					});
					return `${name}${formattedEnergy}${formattedSoc}`;
				});
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
		this.$refs.modal.addEventListener("show.bs.modal", this.modalVisible);
		this.$refs.modal.addEventListener("hidden.bs.modal", this.modalInvisible);
	},
	unmounted() {
		this.$refs.modal?.removeEventListener("show.bs.modal", this.modalVisible);
		this.$refs.modal?.removeEventListener("hidden.bs.modal", this.modalInvisible);
	},
	methods: {
		modalVisible: function () {
			this.isModalVisible = true;
		},
		modalInvisible: function () {
			this.isModalVisible = false;
		},
		changeBufferStart($event) {
			this.setBufferStart($event.target.value);
		},
		changePrioritySoc($event) {
			const value = parseInt($event.target.value, 10);
			if (value > this.bufferSoc) {
				const startOption = this.bufferStartOption;
				this.saveBufferSoc(value);
				this.setBufferStart(startOption);
			} else {
				this.savePrioritySoc(value);
			}
		},
		bufferStartOptionToSoc(option) {
			const bufferSoc = this.selectedBufferSoc;
			const bufferHeight = 100 - bufferSoc;
			switch (option) {
				case "low":
					return bufferSoc + bufferHeight * 0.2;
				case "medium":
					return bufferSoc + bufferHeight * 0.5;
				case "full":
					return bufferSoc + bufferHeight * 0.8;
				case "never":
					return 0;
			}
		},
		toggleBufferStart() {
			const next = {
				never: "low",
				low: "medium",
				medium: "full",
				full: "never",
			};
			const nextOption = next[this.bufferStartOption];
			if (nextOption) {
				this.setBufferStart(nextOption);
			}
		},
		setBufferStart(option) {
			this.selectedBufferStartSoc = this.bufferStartOptionToSoc(option);
			this.saveBufferStartSoc(this.selectedBufferStartSoc);
		},
		changeBufferSoc($event) {
			const startOption = this.bufferStartOption;
			this.saveBufferSoc(parseInt($event.target.value, 10));
			this.setBufferStart(startOption);
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
			return `${Math.round(soc)}%`;
		},
		async changeDischargeControl(e) {
			try {
				await api.post(`batterydischargecontrol/${e.target.checked ? "true" : "false"}`);
			} catch (err) {
				console.error(err);
			}
		},
	},
};
</script>

<style scoped>
.battery {
	height: 300px;
	display: flex;
}
.batteryLimits {
	width: 50px;
	position: relative;
}

.bufferStart,
.bufferSoc,
.prioritySoc {
	position: absolute;
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
	height: 100%;
	width: 100px;
	flex-direction: column;
	position: relative;
	border-radius: 10px;
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
.custom-select {
	left: 0;
	top: 0;
	bottom: 0;
	right: 0;
	position: absolute;
	opacity: 0;
	-webkit-appearance: menulist-button;
}
</style>
