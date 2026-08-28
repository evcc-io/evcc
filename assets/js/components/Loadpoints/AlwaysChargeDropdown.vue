<template>
	<div
		class="dropdown-menu dropdown-menu-end rounded-4 p-3 text-wrap"
		role="group"
		:aria-labelledby="titleId"
		data-testid="always-charge-dropdown"
	>
		<div class="d-flex align-items-center gap-3">
			<AlwaysChargeIcon
				class="flex-shrink-0"
				:class="isActive ? 'text-primary' : 'evcc-gray'"
				aria-hidden="true"
			/>
			<div class="flex-grow-1 overflow-hidden">
				<div :id="titleId" class="fw-bold text-truncate">{{ $t(labelKey) }}</div>
				<div v-if="hint" :id="descriptionId" class="text-primary" aria-live="polite">
					{{ $t(`main.alwaysCharge.${hint}`) }}
				</div>
				<div v-else :id="descriptionId" class="evcc-gray d-flex flex-wrap column-gap-1">
					<span class="text-nowrap">{{ $t("main.alwaysCharge.description") }},</span>
					<span class="text-nowrap">{{ minLabel }}</span>
				</div>
			</div>
			<div class="d-flex align-items-center gap-2 flex-shrink-0">
				<button
					type="button"
					class="p-0 border-0 bg-transparent"
					:class="isOnce ? 'text-primary' : 'evcc-gray'"
					:title="$t('main.alwaysCharge.onceTip')"
					:aria-label="$t('main.alwaysCharge.onceTip')"
					:aria-pressed="isOnce"
					@click="toggleOnce"
				>
					<OnceIcon :filled="isOnce" aria-hidden="true" />
				</button>
				<div class="form-check form-switch m-0">
					<input
						class="form-check-input"
						type="checkbox"
						role="switch"
						:checked="isActive"
						:aria-labelledby="titleId"
						:aria-describedby="descriptionId"
						@change="toggle"
					/>
				</div>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import { ALWAYS_CHARGE, type Timeout } from "@/types/evcc";
import AlwaysChargeIcon from "../MaterialIcon/AlwaysCharge.vue";
import OnceIcon from "../MaterialIcon/Once.vue";
import formatter from "@/mixins/formatter";

const { OFF, ON, ONCE } = ALWAYS_CHARGE;

export default defineComponent({
	name: "AlwaysChargeDropdown",
	components: { AlwaysChargeIcon, OnceIcon },
	mixins: [formatter],
	props: {
		id: { type: String, default: "" },
		alwaysCharge: { type: String, default: OFF },
		minCurrent: { type: Number, default: 0 },
		heating: Boolean,
	},
	emits: ["updated"],
	data() {
		return {
			hint: null as "once" | "permanent" | null,
			timeout: null as Timeout | null,
		};
	},
	computed: {
		titleId() {
			return `always-charge-${this.id}-title`;
		},
		descriptionId() {
			return `always-charge-${this.id}-description`;
		},
		labelKey() {
			return this.heating ? "main.alwaysCharge.labelHeating" : "main.alwaysCharge.label";
		},
		isActive() {
			return this.alwaysCharge === ON || this.alwaysCharge === ONCE;
		},
		isOnce() {
			return this.alwaysCharge === ONCE;
		},
		minLabel() {
			const current = `${this.fmtNumber(this.minCurrent, this.minCurrent % 1 ? 1 : 0)} A`;
			return this.$t("main.alwaysCharge.minCurrent", { current });
		},
	},
	unmounted() {
		this.clearHint();
	},
	methods: {
		toggle() {
			this.clearHint();
			this.$emit("updated", this.isActive ? OFF : ON);
		},
		toggleOnce() {
			const newValue = this.isOnce ? ON : ONCE;
			// briefly explain the new scope, then revert to the regular description
			this.showHint(newValue === ONCE ? "once" : "permanent");
			this.$emit("updated", newValue);
		},
		showHint(hint: "once" | "permanent") {
			this.hint = hint;
			if (this.timeout) clearTimeout(this.timeout);
			this.timeout = setTimeout(() => this.clearHint(), 5000);
		},
		clearHint() {
			this.hint = null;
			if (this.timeout) {
				clearTimeout(this.timeout);
				this.timeout = null;
			}
		},
	},
});
</script>

<style scoped>
.dropdown-menu {
	--bs-dropdown-min-width: min(340px, calc(100vw - 2rem));
}
</style>
