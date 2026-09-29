<template>
	<Card :title="$t('batterySettings.usageTab')" :subtitle="chargeSubtitle">
		<div class="d-flex gap-3 mb-4" data-testid="battery-priority">
			<shopicon-regular-sun
				size="s"
				class="text-primary flex-shrink-0 mt-1"
			></shopicon-regular-sun>
			<div>
				<div class="fw-bold mb-1">{{ $t("battery.config.priorityTitle") }}</div>
				<i18n-t
					:keypath="
						selectedPrioritySoc > 0
							? 'battery.config.priority'
							: 'battery.config.priorityNone'
					"
					tag="p"
					class="mb-0"
					scope="global"
				>
					<template #soc>
						<InlineSocSelect
							id="batteryPriority"
							:options="priorityOptions"
							:selected="selectedPrioritySoc"
							:label="fmtSoc(selectedPrioritySoc)"
							nowrap
							@change="changePrioritySoc"
						/>
					</template>
				</i18n-t>
			</div>
		</div>

		<div class="d-flex gap-3 mb-4" data-testid="battery-buffer">
			<shopicon-regular-lightning
				size="s"
				class="text-primary flex-shrink-0 mt-1"
			></shopicon-regular-lightning>
			<div>
				<div class="fw-bold mb-1">{{ $t("battery.config.bufferTitle") }}</div>
				<i18n-t
					:keypath="
						selectedBufferSoc < 100
							? 'battery.config.buffer'
							: 'battery.config.bufferNone'
					"
					tag="p"
					class="mb-0"
					scope="global"
				>
					<template #soc>
						<InlineSocSelect
							id="batteryBuffer"
							:options="bufferOptions"
							:selected="selectedBufferSoc"
							:label="fmtSoc(selectedBufferSoc)"
							nowrap
							@change="changeBufferSoc"
						/>
					</template>
					<template #start>
						<InlineSocSelect
							id="batteryBufferStart"
							:options="bufferStartOptions"
							:selected="selectedBufferStartSoc"
							:label="selectedBufferStartName"
							@change="changeBufferStart"
						/>
					</template>
				</i18n-t>
			</div>
		</div>

		<template v-if="controllable">
			<hr class="my-3" />
			<div class="form-check form-switch">
				<input
					id="batteryDischarge"
					:checked="batteryDischargeControl"
					class="form-check-input"
					type="checkbox"
					role="switch"
					@change="changeDischargeControl"
				/>
				<label class="form-check-label" for="batteryDischarge">
					{{ $t("battery.config.discharge") }}
				</label>
			</div>
			<div v-if="experimental" class="form-check form-switch mt-2">
				<input
					id="batteryGridDischarge"
					:checked="batteryGridDischarge"
					class="form-check-input"
					type="checkbox"
					role="switch"
					@change="changeGridDischarge"
				/>
				<label class="form-check-label" for="batteryGridDischarge">
					{{ $t("battery.config.gridDischarge") }} 🧪
				</label>
			</div>
			<ConfirmModal
				id="gridDischargeConfirmModal"
				ref="gridDischargeConfirm"
				:title="$t('battery.config.gridDischargeConfirm.title')"
				:description="$t('battery.config.gridDischargeConfirm.description')"
				:confirm-label="$t('config.general.forceEnable')"
				danger
				data-testid="grid-discharge-confirm-modal"
			>
				<p v-if="country">
					{{
						$t("battery.config.gridDischargeConfirm.country", {
							country: fmtCountryName(country),
						})
					}}
				</p>
				<i18n-t
					v-else
					keypath="battery.config.gridDischargeConfirm.noCountry"
					tag="p"
					scope="global"
				>
					<template #link>
						<router-link :to="siteConfigRoute" @click="closeGridDischargeConfirm">
							{{ $t("config.main.title") }} › {{ $t("config.general.site") }}
						</router-link>
					</template>
				</i18n-t>
			</ConfirmModal>
		</template>
	</Card>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/sun";
import "@h2d2/shopicons/es/regular/lightning";
import { defineComponent, type PropType } from "vue";
import formatter from "@/mixins/formatter";
import api from "@/api";
import type { Battery } from "@/types/evcc";
import Card from "../Helper/Card.vue";
import ConfirmModal from "../Helper/ConfirmModal.vue";
import InlineSocSelect from "./InlineSocSelect.vue";

// Battery usage controls: surplus priority, charging buffer and discharge switches.
export default defineComponent({
	name: "BatteryConfigCard",
	components: { Card, ConfirmModal, InlineSocSelect },
	mixins: [formatter],
	props: {
		bufferSoc: { type: Number, default: 100 },
		prioritySoc: { type: Number, default: 0 },
		bufferStartSoc: { type: Number, default: 0 },
		batteryDischargeControl: Boolean,
		batteryGridDischarge: Boolean,
		battery: { type: Object as PropType<Battery> },
		experimental: Boolean,
		country: String,
	},
	data() {
		return {
			selectedBufferSoc: 100,
			selectedPrioritySoc: 0,
			selectedBufferStartSoc: 0,
		};
	},
	computed: {
		siteConfigRoute() {
			return { path: "/config", query: { site: "" } };
		},
		gridDischargeConfirmModal() {
			return this.$refs["gridDischargeConfirm"] as
				| InstanceType<typeof ConfirmModal>
				| undefined;
		},
		chargeSubtitle(): string {
			return `${this.$t("battery.card.soc")} ${this.fmtSoc(this.batterySoc)}`;
		},
		batterySoc(): number {
			return this.battery?.soc ?? 0;
		},
		controllable(): boolean {
			return (this.battery?.devices ?? []).some(({ controllable }) => controllable);
		},
		priorityOptions() {
			const options = [];
			for (let i = 100; i >= 0; i -= 5) {
				const disabled =
					i > this.selectedBufferSoc &&
					!(this.selectedBufferSoc == this.selectedPrioritySoc);
				options.push({ value: i, name: this.fmtSoc(i), disabled });
			}
			return options;
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
		bufferStartOptions() {
			const options = [];
			for (let i = 100; i >= this.selectedBufferSoc; i -= 5) {
				options.push({ value: i, name: this.getBufferStartName(i) });
			}
			options.push({ value: 0, name: this.getBufferStartName(0) });
			return options;
		},
		selectedBufferStartName(): string {
			return this.getBufferStartName(this.selectedBufferStartSoc);
		},
	},
	watch: {
		prioritySoc: {
			handler(soc) {
				this.selectedPrioritySoc = soc;
			},
			immediate: true,
		},
		bufferSoc: {
			handler(soc) {
				this.selectedBufferSoc = soc || 100;
			},
			immediate: true,
		},
		bufferStartSoc: {
			handler(soc) {
				this.selectedBufferStartSoc = soc;
			},
			immediate: true,
		},
	},
	methods: {
		changePrioritySoc($event: Event) {
			const soc = parseInt(($event.target as HTMLInputElement).value, 10);
			if (soc > (this.bufferSoc || 100)) {
				this.saveBufferSoc(soc);
				if (soc > this.bufferStartSoc && this.bufferStartSoc > 0) {
					this.setBufferStartSoc(soc);
				}
			} else {
				this.savePrioritySoc(soc);
			}
		},
		changeBufferStart($event: Event) {
			this.setBufferStartSoc(parseInt(($event.target as HTMLInputElement).value, 10));
		},
		async changeBufferSoc($event: Event) {
			const soc = parseInt(($event.target as HTMLInputElement).value, 10);
			if (soc === 100) {
				await this.setBufferStartSoc(0);
			} else if (soc > this.selectedBufferStartSoc && this.selectedBufferStartSoc > 0) {
				await this.setBufferStartSoc(soc);
			}
			await this.saveBufferSoc(soc);
		},
		async setBufferStartSoc(soc: number) {
			this.selectedBufferStartSoc = soc;
			await this.saveBufferStartSoc(soc);
		},
		async savePrioritySoc(soc: number) {
			this.selectedPrioritySoc = soc;
			try {
				await api.post(`prioritysoc/${encodeURIComponent(soc)}`);
			} catch (err) {
				console.error(err);
			}
		},
		async saveBufferSoc(soc: number) {
			this.selectedBufferSoc = soc;
			try {
				await api.post(`buffersoc/${encodeURIComponent(soc)}`);
			} catch (err) {
				console.error(err);
			}
		},
		async saveBufferStartSoc(soc: number) {
			try {
				await api.post(`bufferstartsoc/${encodeURIComponent(soc)}`);
			} catch (err) {
				console.error(err);
			}
		},
		async changeDischargeControl(e: Event) {
			try {
				await api.post(
					`batterydischargecontrol/${(e.target as HTMLInputElement).checked ? "true" : "false"}`
				);
			} catch (err) {
				console.error(err);
			}
		},
		async changeGridDischarge(e: Event) {
			const target = e.target as HTMLInputElement;
			try {
				if (!(await this.postGridDischarge(target.checked))) {
					target.checked = this.batteryGridDischarge; // cancelled, revert to stay in sync with state
				}
			} catch (err) {
				target.checked = this.batteryGridDischarge;
				console.error(err);
			}
		},
		closeGridDischargeConfirm() {
			this.gridDischargeConfirmModal?.close();
		},
		// 428 asks for confirmation, then the request is repeated with force
		async postGridDischarge(enable: boolean, force = false): Promise<boolean> {
			const res = await api.post(`batterygriddischarge/${enable}`, null, {
				params: force ? { force } : undefined,
				validateStatus: (status) => status === 428 || (status >= 200 && status < 300),
			});
			if (res.status !== 428) {
				return true;
			}
			if (!(await this.gridDischargeConfirmModal?.confirm())) {
				return false;
			}
			return this.postGridDischarge(enable, true);
		},
		getBufferStartName(value: number) {
			const key = value === 0 ? "never" : value === 100 ? "full" : "above";
			return this.$t(`battery.config.bufferStart.${key}`, { soc: this.fmtSoc(value) });
		},
		fmtSoc(soc: number) {
			return this.fmtPercentage(soc);
		},
	},
});
</script>
