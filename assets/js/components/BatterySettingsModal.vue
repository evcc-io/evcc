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
					<div
						class="modal-body d-flex justify-content-between align-items-center align-items-sm-start pb-2 flex-column flex-sm-row-reverse"
					>
						<div class="battery mb-5 mb-sm-3 me-5 me-sm-0">
							<div class="batteryLimits">
								<label
									class="bufferSoc p-2 end-0"
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
									<span class="text-decoration-underline text-nowrap">
										{{ selectedBufferSoc }} %
									</span>
								</label>
								<label
									class="prioritySoc p-2 end-0"
									for="batteryPrioritySoc"
									:style="{ top: `${100 - bottomHeight}%` }"
								>
									<select
										id="batteryPrioritySoc"
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
									<span class="text-decoration-underline text-nowrap">
										{{ selectedPrioritySoc }} %
									</span>
								</label>
								<label
									class="bufferStart p-2 end-0 pe-0"
									:class="{
										'bufferStart--hidden':
											!selectedBufferStart || selectedBufferSoc === 100,
									}"
									role="button"
									:style="{ top: `${bufferStartTop}%` }"
									@click="toggleBufferStart"
								>
									<shopicon-regular-lightning
										size="s"
										class="me-2 text-primary"
									></shopicon-regular-lightning>
								</label>
							</div>
							<div class="progress">
								<div
									class="bg-dark-green progress-bar text-light align-items-center"
									role="progressbar"
									:style="{ height: `${topHeight}%` }"
								>
									<shopicon-regular-car3
										size="m"
										class="icon"
										:style="iconStyle(topHeight)"
									></shopicon-regular-car3>
								</div>
								<div
									class="bg-darker-green progress-bar text-light align-items-center"
									role="progressbar"
									:style="{ height: `${middleHeight}%` }"
								>
									<!--
									<shopicon-regular-car3
										size="m"
										class="icon"
										:style="iconStyle(middleHeight)"
									></shopicon-regular-car3>
									-->
								</div>
								<div
									class="bg-darkest-green progress-bar text-light align-items-center"
									role="progressbar"
									:style="{ height: `${bottomHeight}%` }"
								>
									<shopicon-regular-home
										size="m"
										class="icon"
										:style="iconStyle(bottomHeight)"
									></shopicon-regular-home>
								</div>
								<div
									class="batterySoc ps-0 bg-white"
									:style="{ top: `${100 - batterySoc}%` }"
								></div>
								<div
									class="bufferStartIndicator ps-0 start-0 bg-white"
									:class="{
										'bufferStartIndicator--hidden':
											!selectedBufferStart || selectedBufferSoc === 100,
									}"
									:style="{ top: `${bufferStartTop}%` }"
								></div>
							</div>
						</div>
						<div class="me-sm-4">
							<p>
								Battery level: <strong>{{ batterySoc }} %</strong>
							</p>
							<p>How to use the battery:</p>
							<p class="d-flex">
								<shopicon-regular-car3
									size="s"
									class="flex-shrink-0 me-2"
								></shopicon-regular-car3>
								<span class="d-block">
									for charging
									<small class="d-block"> without interruptions </small>
								</span>
							</p>
							<p
								class="d-flex ms-4 pb-2 bufferStartLegend"
								:class="{ 'bufferStartLegend--hidden': selectedBufferSoc == 100 }"
							>
								<shopicon-regular-lightning
									size="s"
									class="flex-shrink-0 me-2"
								></shopicon-regular-lightning>
								<span class="d-block">
									start automatically when
									<small class="d-block">
										<a
											href=""
											:class="{ small: bufferStartOption !== 'low' }"
											@click.prevent="setBufferStart('low')"
											>low</a
										>,
										<a
											href=""
											:class="{ small: bufferStartOption !== 'medium' }"
											@click.prevent="setBufferStart('medium')"
											>medium</a
										>,
										<a
											href=""
											:class="{ small: bufferStartOption !== 'high' }"
											@click.prevent="setBufferStart('high')"
											>full</a
										>
										or
										<a
											href=""
											:class="{ small: bufferStartOption !== 'never' }"
											@click.prevent="removeBufferStart"
											>never</a
										>
									</small>
								</span>
							</p>
							<p class="d-flex">
								<shopicon-regular-home
									size="s"
									class="flex-shrink-0 me-2"
								></shopicon-regular-home>
								<span class="d-block">
									for home use
									<small class="d-block"> for nightly consumption </small>
								</span>
							</p>
							<p>
								<small> Note: These settings only affect the solar mode. </small>
							</p>
						</div>
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

import api from "../api";

export default {
	name: "BatterySettingsModal",
	props: {
		bufferSoc: Number,
		prioritySoc: Number,
		batterySoc: Number,
		bufferStart: Number,
	},
	data: function () {
		return {
			isModalVisible: false,
			selectedBufferSoc: 100,
			selectedPrioritySoc: 0,
			selectedBufferStart: null,
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
				options.push({ value: i, name: `${i} %`, disabled });
			}
			return options;
		},
		bufferOptions() {
			const options = [];
			for (let i = 100; i >= 5; i -= 5) {
				options.push({ value: i, name: `${i} %`, disabled: i < this.selectedPrioritySoc });
			}
			return options;
		},
		bufferStartTop() {
			if (!this.selectedBufferStart) return 0;
			return 100 - this.selectedBufferStart;
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
			if (!this.selectedBufferStart) return "never";
			if (this.selectedBufferStart >= 95) return "high";
			if (this.selectedBufferStart - 5 <= this.selectedBufferSoc) return "low";
			return "medium";
		},
	},
	watch: {
		prioritySoc(soc) {
			this.selectedPrioritySoc = soc;
		},
		bufferSoc(soc) {
			this.selectedBufferSoc = soc || 100;
		},
		selectedBufferStart(soc) {
			this.saveBufferStart(soc);
		},
	},
	mounted() {
		this.selectedBufferSoc = this.bufferSoc;
		this.selectedPrioritySoc = this.prioritySoc;
		this.selectedBufferSoc = this.bufferStart;
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
		changePrioritySoc($event) {
			const value = parseInt($event.target.value, 10);
			if (value > this.bufferSoc) {
				this.saveBufferSoc(value);
			} else {
				this.savePrioritySoc(value);
			}
		},
		toggleBufferStart() {
			const next = {
				low: "medium",
				medium: "high",
				high: "low",
			};
			const nextOption = next[this.bufferStartOption];
			if (nextOption) {
				this.setBufferStart(nextOption);
			}
		},
		setBufferStart(position) {
			switch (position) {
				case "low":
					this.selectedBufferStart = this.selectedBufferSoc + 5;
					break;
				case "medium":
					this.selectedBufferStart =
						this.selectedBufferSoc + (100 - this.selectedBufferSoc) / 2;
					break;
				case "high":
					this.selectedBufferStart = 95;
					break;
			}
		},
		removeBufferStart() {
			this.selectedBufferStart = null;
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
		async saveBufferStart(soc) {
			try {
				await api.post(`bufferstart/${encodeURIComponent(soc)}`);
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
	border-radius: 0 0.5rem 0.5rem 0;
	height: 0.5rem;
	width: 0.5rem;
}
.bufferStartIndicator--hidden {
	opacity: 0;
	transform: translateY(-50%);
}
.bufferStartLegend {
	transition: opacity;
	transition-duration: var(--evcc-transition-medium);
	transition-timing-function: linear;
	opacity: 1;
	overflow: hidden;
	height: auto;
}
.bufferStartLegend--hidden {
	opacity: 0;
	height: 0;
	margin: 0 !important;
	padding: 0 !important;
}
.progress {
	height: 100%;
	width: 100px;
	flex-direction: column;
	position: relative;
	border-radius: 10px;
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
}
</style>
