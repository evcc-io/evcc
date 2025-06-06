<template>
	<div data-testid="offline-indicator">
		<div v-if="offline" class="modal-backdrop" />
		<div
			class="fixed-bottom alert d-flex justify-content-center align-items-center mb-0 rounded-0 p-2"
			:class="{ visible: visible, 'alert-danger': showError, 'alert-secondary': !showError }"
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
			<div
				v-else-if="showError"
				class="d-flex align-items-center container px-4 justify-content-center"
				data-testid="fatal-error"
			>
				<shopicon-regular-car1
					size="m"
					class="fatal-icon flex-grow-0 flex-shrink-0"
				></shopicon-regular-car1>
				<div class="mx-4 mt-1">
					<div>
						<strong>
							{{ $t("offline.configurationError") }}
						</strong>
					</div>
					<div v-if="fatalText" class="text-break">{{ fatalText }}</div>
				</div>
				<RestartButton error @restart="restart" />
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
import type { FatalError } from "@/types/evcc";

export default defineComponent({
	name: "OfflineIndicator",
	components: {
		CloudOffline,
		RestartButton,
	},
	props: {
		offline: Boolean,
		fatal: Object as PropType<FatalError>,
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
		visible() {
			return this.offline || this.restartNeeded || this.restarting || this.showError;
		},
		showError() {
			return (
				!this.offline &&
				!this.restartNeeded &&
				!this.restarting &&
				this.fatal?.error &&
				!this.dismissed
			);
		},
		fatalText() {
			const { error, class: errorClass } = this.fatal || {};
			if (!error) return;
			return errorClass ? `${errorClass}: ${error}` : error;
		},
	},
	watch: {
		offline() {
			if (!this.offline) {
				restartComplete();
				this.dismissed = false;
			}
		},
	},
	methods: {
		restart() {
			performRestart();
		},
	},
});
</script>
<style scoped>
.alert {
	opacity: 0;
	transform: translateY(100%);
	transition:
		transform var(--evcc-transition-fast) ease-in,
		opacity var(--evcc-transition-fast) ease-in;
	min-height: 58px;
	/* above backdrop, below modal https://getbootstrap.com/docs/5.3/layout/z-index/ */
	z-index: 1054 !important;
}
.alert.visible {
	opacity: 1;
	transform: translateY(0);
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
.btn-close {
	filter: none;
}
</style>
