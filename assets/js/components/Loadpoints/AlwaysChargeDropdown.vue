<template>
	<div class="panel" data-testid="always-charge-dropdown">
		<div class="d-flex align-items-center gap-3">
			<AlwaysChargeIcon class="icon flex-shrink-0" :class="{ active: isActive }" />
			<div class="flex-grow-1 overflow-hidden">
				<div class="fw-bold text-truncate">{{ $t("main.alwaysCharge.label") }}</div>
				<div class="subline" :class="{ hint: !!hint }">{{ subline }}</div>
			</div>
			<div class="d-flex align-items-center gap-2 flex-shrink-0">
				<button
					type="button"
					class="once-badge"
					:class="{ active: isOnce }"
					:title="$t('main.alwaysCharge.onceTip')"
					:aria-label="$t('main.alwaysCharge.onceTip')"
					:aria-pressed="isOnce"
					@click="toggleOnce"
				>
					<OnceIcon :filled="isOnce" />
				</button>
				<div class="form-check form-switch m-0">
					<input
						class="form-check-input"
						type="checkbox"
						role="switch"
						:checked="isActive"
						:aria-label="$t('main.alwaysCharge.label')"
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
		alwaysCharge: { type: String, default: OFF },
		minCurrent: { type: Number, default: 0 },
	},
	emits: ["updated"],
	data() {
		return {
			hint: null as "once" | "permanent" | null,
			timeout: null as Timeout | null,
		};
	},
	computed: {
		isActive() {
			return this.alwaysCharge === ON || this.alwaysCharge === ONCE;
		},
		isOnce() {
			return this.alwaysCharge === ONCE;
		},
		subline() {
			if (this.hint) {
				return this.$t(`main.alwaysCharge.${this.hint}`);
			}
			return this.$t("main.alwaysCharge.description", {
				current: this.fmtNumber(this.minCurrent, this.minCurrent % 1 ? 1 : 0),
			});
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
.panel {
	--highlight: var(--evcc-darker-green);
	position: absolute;
	top: calc(100% + 10px);
	right: 0;
	z-index: 5;
	width: 320px;
	max-width: calc(100vw - 2rem);
	padding: 0.75rem 0.875rem;
	background: var(--evcc-box);
	border-radius: 1rem;
	box-shadow: 0 0 8px var(--bs-gray-light);
	text-align: start;
}
html.dark .panel {
	/* brighter green for contrast on dark panel */
	--highlight: var(--evcc-dark-green);
}
.icon {
	color: var(--evcc-gray);
}
.icon.active {
	color: var(--highlight);
}
.subline {
	font-size: var(--bs-body-font-size);
	color: var(--evcc-gray);
}
.subline.hint {
	color: var(--highlight);
}
.once-badge {
	padding: 0;
	color: var(--evcc-gray);
	background: transparent;
	border: none;
}
.once-badge.active {
	color: var(--highlight);
}
.form-check-input:checked {
	background-color: var(--highlight);
	border-color: var(--highlight);
}
</style>
