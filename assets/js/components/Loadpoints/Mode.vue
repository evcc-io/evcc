<template>
	<div class="mode-group border d-inline-flex position-relative" role="group" data-testid="mode">
		<template v-for="m in modes" :key="m">
			<div
				v-if="m === SMART && alwaysChargePossible"
				class="smart-pill d-flex"
				:class="{ active: isActive(m) }"
			>
				<span class="pill-balance" aria-hidden="true"></span>
				<button
					type="button"
					class="btn smart-btn d-flex align-items-center justify-content-center"
					:class="{ active: isActive(m) }"
					tabindex="0"
					@click="setTargetMode(m)"
				>
					{{ label(m) }}
				</button>
				<button
					ref="chevron"
					type="button"
					class="btn chevron-btn d-flex align-items-center"
					:class="{ active: isActive(m) }"
					data-bs-toggle="dropdown"
					data-bs-reference="parent"
					data-bs-auto-close="outside"
					data-bs-offset="0,10"
					aria-expanded="false"
					:aria-label="$t('main.alwaysCharge.label')"
					@click="ensureSmart"
				>
					<AlwaysChargeIcon
						v-if="alwaysChargeActive"
						class="always-charge-icon"
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
				class="btn flex-grow-1 flex-shrink-1"
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
import type { Options as PopperOptions } from "@popperjs/core";

type PopperConfig = Partial<PopperOptions> & { modifiers: NonNullable<PopperOptions["modifiers"]> };

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
		return { SMART, bsDropdown: null as Dropdown | null };
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
		this.syncDropdown();
	},
	updated() {
		this.syncDropdown();
	},
	beforeUnmount() {
		this.bsDropdown?.dispose();
	},
	methods: {
		label(mode: CHARGE_MODE) {
			return this.$t(chargeModeLabelKey(mode, this.continuous, this.switchDevice));
		},
		isActive(mode: CHARGE_MODE) {
			return this.mode === mode;
		},
		setTargetMode(mode: CHARGE_MODE) {
			this.bsDropdown?.hide();
			this.$emit("updated", mode);
		},
		updateAlwaysCharge(value: ALWAYS_CHARGE) {
			this.$emit("always-charge-updated", value);
		},
		syncDropdown() {
			// ref lives inside v-for, so vue collects it into an array
			const chevron = (this.$refs["chevron"] as HTMLElement[] | undefined)?.[0];
			if (!chevron) {
				this.bsDropdown?.dispose();
				this.bsDropdown = null;
			} else if (!this.bsDropdown) {
				// bootstrap's delegated toggle runs in the capture phase, so the instance
				// has to exist before the first click for these options to apply
				this.bsDropdown = new Dropdown(chevron, {
					popperConfig: (config: PopperConfig) => {
						// keep some distance to the screen edges
						config.modifiers.push({
							name: "preventOverflow",
							options: { padding: 16 },
						});
						return config;
					},
				});
			}
		},
		ensureSmart() {
			if (this.mode !== SMART) {
				this.$emit("updated", SMART);
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
	/* equal share when there is room, content width when there is not.
	   the basis mirrors .btn's padding floor (0.8em at its 1rem font size),
	   so the pill and the plain modes end up the same width */
	flex-grow: 1;
	flex-basis: 1.6rem;
	border-radius: 18px;
}
.smart-pill.active {
	background: var(--evcc-default-text);
}
.smart-pill .btn.active {
	background: none;
}
/* mirrors the chevron side, so the label sits in the middle of the whole pill.
   both are flexible, on tight space they collapse to the chevron content */
.smart-pill .pill-balance {
	flex: 1 1 0;
}
.smart-pill .smart-btn {
	flex: 0 1 auto;
	border-radius: 18px 0 0 18px;
	/* fixed inset, the label must not touch the pill edge on small screens */
	padding-left: 0.8em;
	padding-right: 0;
}
.smart-pill .chevron-btn {
	flex: 1 1 0;
	justify-content: flex-end;
	gap: 3px;
	border-radius: 0 18px 18px 0;
	padding-left: 0.3em;
	/* the chevron glyph carries whitespace, so less than the label side */
	padding-right: 0.3em;
}
.smart-pill .chevron-btn svg {
	flex-shrink: 0;
}
/* the chevron glyph carries whitespace, the infinity glyph does not */
.always-charge-icon {
	margin-left: 0.4em;
}
/* full ring around the whole pill when the mode button is focused */
.smart-pill:has(.smart-btn:focus-visible) {
	outline: var(--bs-focus-ring-width) solid var(--bs-focus-ring-color);
}
.smart-pill .smart-btn:focus-visible {
	outline: none;
}
@media (max-width: 576px) {
	.smart-pill {
		flex-basis: 0.4rem;
	}
}
.chevron-btn.show .chevron {
	transform: rotate(180deg);
}
</style>
