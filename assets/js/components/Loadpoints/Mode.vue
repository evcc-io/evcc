<template>
	<div
		ref="root"
		class="mode-group border d-inline-flex position-relative"
		role="group"
		data-testid="mode"
	>
		<template v-for="m in modes" :key="m">
			<div
				v-if="m === SMART && alwaysChargePossible"
				class="smart-pill d-flex"
				:class="{ active: isActive(m) }"
			>
				<button
					type="button"
					class="btn smart-btn flex-grow-1 flex-shrink-1 text-truncate-xs-only d-flex align-items-center justify-content-center"
					:class="{ active: isActive(m) }"
					tabindex="0"
					@click="setTargetMode(m)"
				>
					{{ label(m) }}
				</button>
				<button
					type="button"
					class="btn chevron-btn d-flex align-items-center"
					:class="{ active: isActive(m) }"
					data-testid="always-charge-toggle"
					data-bs-toggle="dropdown"
					aria-expanded="false"
					:aria-label="$t('main.alwaysCharge.label')"
					@click="ensureSmart"
				>
					<AlwaysChargeIcon
						v-if="alwaysChargeActive"
						:class="{ 'text-evcc': charging && isActive(m) }"
						aria-hidden="true"
					/>
					<DropdownIcon class="chevron" aria-hidden="true" />
				</button>
				<AlwaysChargeDropdown
					:id="id"
					:alwaysCharge="alwaysCharge"
					:heating="heating"
					:minCurrent="effectiveMinCurrent"
					@updated="updateAlwaysCharge"
				/>
			</div>
			<button
				v-else
				type="button"
				class="btn flex-grow-1 flex-shrink-1 text-truncate-xs-only"
				:class="{ active: isActive(m) }"
				tabindex="0"
				@click="setTargetMode(m)"
			>
				{{ label(m) }}
			</button>
		</template>
	</div>
</template>

<script lang="ts">
import { CHARGE_MODE, ALWAYS_CHARGE } from "@/types/evcc";
import { defineComponent } from "vue";
import Dropdown from "bootstrap/js/dist/dropdown";
import chargeModeLabelKey from "@/utils/chargeModeLabel";
import AlwaysChargeIcon from "../MaterialIcon/AlwaysCharge.vue";
import DropdownIcon from "../MaterialIcon/Dropdown.vue";
import AlwaysChargeDropdown from "./AlwaysChargeDropdown.vue";

const { OFF, SMART, NOW } = CHARGE_MODE;

export default defineComponent({
	name: "Mode",
	components: { AlwaysChargeIcon, DropdownIcon, AlwaysChargeDropdown },
	props: {
		id: { type: String, default: "" },
		mode: String,
		pvPossible: Boolean,
		smartCostAvailable: Boolean,
		switchDevice: Boolean,
		continuous: Boolean,
		heating: Boolean,
		charging: Boolean,
		alwaysCharge: { type: String, default: ALWAYS_CHARGE.OFF },
		effectiveMinCurrent: { type: Number, default: 0 },
	},
	emits: ["updated", "always-charge-updated"],
	data() {
		return { SMART };
	},
	computed: {
		modes(): CHARGE_MODE[] {
			if (this.pvPossible || this.smartCostAvailable) {
				return [OFF, SMART, NOW];
			}
			return [OFF, NOW];
		},
		alwaysChargePossible() {
			return !this.switchDevice;
		},
		alwaysChargeActive() {
			return this.alwaysCharge !== ALWAYS_CHARGE.OFF;
		},
	},
	mounted() {
		this.initDropdown();
	},
	updated() {
		this.initDropdown();
	},
	beforeUnmount() {
		this.dropdown()?.dispose();
	},
	methods: {
		label(mode: CHARGE_MODE) {
			return this.$t(chargeModeLabelKey(mode, this.continuous, this.switchDevice));
		},
		isActive(mode: CHARGE_MODE) {
			return this.mode === mode;
		},
		setTargetMode(mode: CHARGE_MODE) {
			this.dropdown()?.hide();
			this.$emit("updated", mode);
		},
		updateAlwaysCharge(value: ALWAYS_CHARGE) {
			this.$emit("always-charge-updated", value);
		},
		ensureSmart() {
			if (this.mode !== SMART) {
				this.$emit("updated", SMART);
			}
		},
		toggleEl() {
			return (this.$el as HTMLElement).querySelector<HTMLElement>(
				'[data-bs-toggle="dropdown"]'
			);
		},
		dropdown() {
			const el = this.toggleEl();
			return el ? Dropdown.getInstance(el) : null;
		},
		initDropdown() {
			const el = this.toggleEl();
			if (el && !Dropdown.getInstance(el)) {
				// anchor to the whole group, keep open while toggling inside
				Dropdown.getOrCreateInstance(el, {
					reference: this.$refs["root"] as HTMLElement,
					autoClose: "outside",
					offset: [0, 10],
				});
			}
		},
	},
});
</script>

<style scoped>
.mode-group {
	border: 2px solid var(--evcc-default-text);
	border-radius: 20px;
	padding: 4px;
	min-width: 255px;
}
.btn {
	/* equal width buttons */
	flex-basis: 0;
	white-space: nowrap;
	border-radius: 18px;
	padding: 0.1em 0.8em;
	color: var(--evcc-default-text);
	border: none;
}
@media (max-width: 576px) {
	.btn {
		padding: 0.1em 0.2em;
	}
}

.btn:hover {
	color: var(--evcc-gray);
}
.btn:focus-visible {
	outline: var(--bs-focus-ring-width) solid var(--bs-focus-ring-color);
	outline-width: var(--bs-focus-ring-width);
}
.btn.active {
	color: var(--evcc-background);
	background: var(--evcc-default-text);
}

.smart-pill {
	flex-grow: 1.5;
	flex-basis: 0;
	border-radius: 18px;
}
.smart-pill .smart-btn {
	border-radius: 18px 0 0 18px;
	padding-right: 0;
}
.smart-pill .chevron-btn {
	flex-grow: 0;
	flex-basis: auto;
	gap: 3px;
	border-radius: 0 18px 18px 0;
	padding-left: 0.3em;
	padding-right: 0.5em;
}
/* full ring around the whole pill when the mode button is focused */
.smart-pill:has(.smart-btn:focus-visible) {
	outline: var(--bs-focus-ring-width) solid var(--bs-focus-ring-color);
}
.smart-pill .smart-btn:focus-visible {
	outline: none;
}
@media (max-width: 576px) {
	.smart-pill .smart-btn {
		padding-right: 0.1em;
	}
	.smart-pill .chevron-btn {
		padding-left: 0;
		padding-right: 0.2em;
	}
}
.chevron-btn.show .chevron {
	transform: rotate(180deg);
}
</style>
