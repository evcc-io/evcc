<template>
	<div data-testid="offline-indicator" :aria-hidden="!visible">
		<div v-if="offline || starting" class="modal-backdrop" />
		<div
			class="fixed-bottom alert d-flex justify-content-center align-items-center mb-0 rounded-0 p-2"
			:class="{
				visible: visible,
				'alert-danger': showError,
				'alert-secondary': !showError,
				'alert--bottomtabs': !blocking,
			}"
			role="alert"
			data-testid="bottom-banner"
		>
			<div v-if="restarting" class="d-flex align-items-center">
				<RestartButton restarting @restart="restart" />
				{{ $t("offline.restarting") }}
			</div>
			<div
				v-else-if="restartNeeded"
				class="d-flex align-items-center"
				data-testid="restart-needed"
			>
				<RestartButton @restart="restart" />
				{{ $t("offline.restartNeeded") }}
			</div>
			<div v-else-if="offline" class="d-flex align-items-center">
				<CloudOffline class="m-2" />
				{{ $t("offline.message") }}
			</div>
			<div v-else-if="starting" class="d-flex align-items-center">
				<span
					class="spinner-border spinner-border-sm m-1 me-2"
					role="status"
					aria-hidden="true"
				></span>
				{{ $t("offline.starting") }}
			</div>
			<div
				v-else-if="showError"
				class="d-flex align-items-center container px-0 px-sm-4 flex-wrap gap-2"
				data-testid="fatal-error"
			>
				<div class="d-flex align-items-center gap-4">
					<shopicon-regular-car1
						size="m"
						class="fatal-icon flex-shrink-0 d-none d-sm-block"
					></shopicon-regular-car1>
					<div class="mt-1">
						<div>
							<strong>
								{{ $t("offline.configurationError") }}
							</strong>
						</div>
						<div class="d-flex flex-column gap-1">
							<div
								v-for="fatalText in fatalTexts"
								:key="fatalText"
								class="text-break"
							>
								{{ fatalText }}
							</div>
						</div>
					</div>
				</div>
				<div class="ms-auto d-flex align-items-center gap-3">
					<button
						type="button"
						class="btn btn-link btn-sm text-reset p-0"
						@click="dismiss"
					>
						{{ $t("config.general.dismiss") }}
					</button>
					<RestartButton error @restart="restart" />
				</div>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import "@h2d2/shopicons/es/regular/car1";
import CloudOffline from "../MaterialIcon/CloudOffline.vue";
import RestartButton from "./RestartButton.vue";
import restart, { performRestart, restartComplete } from "@/restart";
import deepEqual from "@/utils/deepEqual";
import type { FatalError } from "@/types/evcc";

export default defineComponent({
	name: "OfflineIndicator",
	components: {
		CloudOffline,
		RestartButton,
	},
	props: {
		offline: Boolean,
		fatal: { type: Array as PropType<FatalError[]>, default: () => [] },
		startupCompleted: Boolean,
	},
	data() {
		return { dismissed: false };
	},
	computed: {
		restartNeeded() {
			return restart.restartNeeded;
		},
		restarting() {
			return restart.restarting;
		},
		starting() {
			return this.startupCompleted === false;
		},
		blocking() {
			return this.offline || this.starting || this.restarting;
		},
		visible() {
			return (
				this.starting ||
				this.offline ||
				this.restartNeeded ||
				this.restarting ||
				this.showError
			);
		},
		showError() {
			return (
				!this.offline &&
				!this.restartNeeded &&
				!this.restarting &&
				this.fatal.length > 0 &&
				!this.dismissed
			);
		},
		fatalTexts() {
			return this.fatal.map(({ error, class: errorClass }) =>
				errorClass ? `${errorClass}: ${error}` : error
			);
		},
	},
	watch: {
		offline() {
			if (!this.offline) {
				restartComplete();
				this.dismissed = false;
			}
		},
		fatal(next, prev) {
			if (!deepEqual(next, prev)) {
				this.dismissed = false;
			}
		},
	},
	methods: {
		restart() {
			performRestart();
		},
		dismiss() {
			this.dismissed = true;
		},
	},
});
</script>
<style scoped>
.alert {
	opacity: 0;
	transform: translateY(100%);
	min-height: 58px;
	transition:
		transform var(--evcc-transition-fast) ease-in,
		opacity var(--evcc-transition-fast) ease-in,
		padding-bottom var(--evcc-transition-fast) ease-in;
	padding-bottom: max(0.5rem, var(--safe-area-inset-bottom)) !important;
	border-bottom: none;
	border-left: none;
	border-right: none;
	/* above backdrop, below modal https://getbootstrap.com/docs/5.3/layout/z-index/ */
	z-index: 1054 !important;
}
.alert.visible {
	opacity: 1;
	transform: translateY(0);
	transition:
		transform var(--evcc-transition-medium) ease-in,
		opacity var(--evcc-transition-medium) ease-in,
		padding-bottom var(--evcc-transition-fast) ease-in;
}

.fatal-icon {
	transform-origin: 60% 40%;
	animation: swinging 3.5s ease-in-out infinite;
}

@keyframes swinging {
	0% {
		transform: translateY(6px) rotate(170deg);
	}
	50% {
		transform: translateY(6px) rotate(185deg);
	}
	100% {
		transform: translateY(6px) rotate(170deg);
	}
}
.alert--bottomtabs {
	z-index: 1029 !important;
	padding-bottom: calc(
		var(--tab-bar-height) + max(0.4rem, var(--safe-area-inset-bottom)) + 0.75rem
	) !important;
}
.btn-close {
	filter: none;
}
</style>
