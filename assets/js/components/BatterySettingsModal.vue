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
						class="modal-body d-flex justify-content-between align-items-center pb-2 flex-column flex-sm-row-reverse"
					>
						<div class="battery mb-5 mb-sm-3 me-5 me-sm-0">
							<div class="batteryLimits">
								<label
									class="bufferSoc p-2 end-0"
									:style="{ top: `${topHeight}%` }"
								>
									<select
										v-model.number="selectedBufferSoc"
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
										v-model.number="selectedPrioritySoc"
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
							</div>
							<div class="progress">
								<div
									class="bg-dark-green progress-bar text-light align-items-center"
									role="progressbar"
									:style="{ height: `${topHeight}%` }"
								>
									<shopicon-regular-cloudsun
										size="m"
										class="icon"
										:style="iconStyle(topHeight)"
									></shopicon-regular-cloudsun>
								</div>
								<div
									class="bg-darker-green progress-bar text-light align-items-center"
									role="progressbar"
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
									class="batterySoc ps-0 start-0 bg-white"
									:style="{ top: `${100 - batterySoc}%` }"
								></div>
							</div>
						</div>
						<div class="me-sm-4">
							<p>
								Battery level: <strong>{{ batterySoc }} %</strong>
							</p>
							<p>How to use the battery:</p>
							<p class="d-flex">
								<shopicon-regular-cloudsun
									size="s"
									class="flex-shrink-0 me-2"
								></shopicon-regular-cloudsun>
								<span class="d-block">
									Bad weather reserve
									<small class="d-block"> fewer starts and stops </small>
								</span>
							</p>
							<p class="d-flex">
								<shopicon-regular-car3
									size="s"
									class="flex-shrink-0 me-2"
								></shopicon-regular-car3>
								<span class="d-block">
									Allowed for charging
									<small class="d-block"> increases charging speed </small>
								</span>
							</p>
							<p class="d-flex">
								<shopicon-regular-home
									size="s"
									class="flex-shrink-0 me-2"
								></shopicon-regular-home>
								<span class="d-block">
									Home reserve
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
import "@h2d2/shopicons/es/regular/cloudsun";
import "@h2d2/shopicons/es/regular/car3";
import "@h2d2/shopicons/es/regular/home";

import api from "../api";

export default {
	name: "BatterySettingsModal",
	props: {
		bufferSoc: Number,
		prioritySoc: Number,
		batterySoc: Number,
	},
	data: function () {
		return {
			isModalVisible: false,
			selectedBufferSoc: 100,
			selectedPrioritySoc: 0,
		};
	},
	computed: {
		priorityOptions() {
			const options = [];
			for (let i = 100; i >= 0; i -= 5) {
				options.push({ value: i, name: `${i} %`, disabled: i > this.bufferSoc });
			}
			return options;
		},
		bufferOptions() {
			const options = [];
			for (let i = 100; i >= 0; i -= 5) {
				options.push({ value: i, name: `${i} %`, disabled: i < this.prioritySoc });
			}
			return options;
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
	},
	watch: {
		prioritySoc(soc) {
			this.selectedPrioritySoc = soc;
		},
		bufferSoc(soc) {
			this.selectedBufferSoc = soc || 100;
		},
	},
	mounted() {
		this.selectedBufferSoc = this.bufferSoc;
		this.selectedPrioritySoc = this.prioritySoc;
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
			this.savePrioritySoc($event.target.value);
		},
		changeBufferSoc($event) {
			this.saveBufferSoc($event.target.value);
		},
		async savePrioritySoc(soc) {
			try {
				await api.post(`prioritysoc/${encodeURIComponent(soc)}`);
			} catch (err) {
				console.error(err);
			}
		},
		async saveBufferSoc(soc) {
			try {
				await api.post(`buffersoc/${encodeURIComponent(soc)}`);
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
.batteryLimits,
.batteryLevel {
	width: 50px;
	position: relative;
}
.bufferSoc,
.prioritySoc {
	position: absolute;
	transform: translateY(-50%);
	transition: top var(--evcc-transition-fast) linear;
}
.batterySoc {
	position: absolute;
	transform: translateY(-50%);
	transition: top var(--evcc-transition-fast) linear;
	height: 0.5rem;
	width: 0.5rem;
	border-radius: 0 100% 100% 0;
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
