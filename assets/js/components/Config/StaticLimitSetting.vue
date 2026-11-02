<template>
	<div class="form-check form-switch">
		<input
			:id="`${id}Enabled`"
			class="form-check-input"
			type="checkbox"
			role="switch"
			:checked="enabled"
			:disabled="state === 'saving'"
			@change="toggle"
		/>
		<label :for="`${id}Enabled`" class="form-check-label">{{ label }}</label>
		<div class="ps-2">
			<p class="text-muted small mb-2">{{ hint }}</p>
			<div class="collapsible-wrapper" :class="{ open: enabled }">
				<div class="collapsible-content ring-space">
					<div class="d-flex align-items-center gap-2">
						<div class="input-group input-width">
							<input
								:id="id"
								v-model.number="kw"
								type="number"
								min="0"
								step="0.1"
								:aria-label="label"
								:aria-describedby="`${id}Unit`"
								class="form-control text-end"
								@input="debounce"
								@blur="commit"
								@keydown.enter.prevent="commit"
							/>
							<span :id="`${id}Unit`" class="input-group-text">kW</span>
						</div>
						<SavingIndicator :state="state" />
					</div>
					<slot />
					<div class="form-text evcc-gray mb-2">{{ description }}</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import api from "@/api";
import SavingIndicator, { type SavingState } from "../Helper/SavingIndicator.vue";

export default defineComponent({
	name: "StaticLimitSetting",
	components: { SavingIndicator },
	props: {
		id: { type: String, required: true },
		label: { type: String, required: true },
		hint: { type: String, required: true },
		description: { type: String, required: true },
		/** Saved limit in W, 0 = disabled. */
		value: { type: Number, default: 0 },
		/** API path the limit in W is posted to. */
		endpoint: { type: String, required: true },
	},
	data() {
		return {
			kw: (this.value / 1000 || null) as number | null,
			enabled: this.value > 0,
			state: null as SavingState,
			timer: undefined as ReturnType<typeof setTimeout> | undefined,
		};
	},
	computed: {
		watts(): number {
			return this.enabled ? Math.round((this.kw || 0) * 1000) : 0;
		},
	},
	watch: {
		value(value: number) {
			// keep the last value after disabling, so re-enabling restores it
			if (value > 0) this.kw = value / 1000;
			this.enabled = value > 0;
		},
	},
	beforeUnmount() {
		clearTimeout(this.timer);
	},
	methods: {
		debounce() {
			clearTimeout(this.timer);
			this.timer = setTimeout(this.commit, 2000);
		},
		commit() {
			clearTimeout(this.timer);
			if (this.state === "saving" || this.watts === this.value) return;
			this.save(this.watts);
		},
		toggle($event: Event) {
			this.enabled = ($event.target as HTMLInputElement).checked;
			this.state = null;
			this.commit();
		},
		async save(value: number) {
			this.state = "saving";
			try {
				await api.post(`${this.endpoint}/${value}`);
				this.state = "saved";
			} catch (e) {
				this.state = null;
				console.error(e);
			}
		},
	},
});
</script>

<style scoped>
.input-width {
	max-width: 180px;
}
</style>
