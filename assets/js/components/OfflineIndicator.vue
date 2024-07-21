<template>
	<div>
		<div class="modal-backdrop" v-if="offline" />
		<div
			class="fixed-bottom alert d-flex justify-content-center align-items-center mb-0 rounded-0 p-2"
			:class="{ visible: visible, 'alert-danger': showError, 'alert-secondary': !showError }"
			role="alert"
			data-testid="bottom-banner"
		>
			<div v-if="restarting" class="d-flex align-items-center">
				<button
					class="btn btn-secondary me-2 btn-sm d-flex align-items-center"
					type="button"
					disabled
				>
					<span
						class="spinner-border spinner-border-sm m-1 me-2"
						role="status"
						aria-hidden="true"
					></span>
					{{ $t("offline.restart") }}
				</button>
				{{ $t("offline.restarting") }}
			</div>
			<div v-else-if="restartNeeded" class="d-flex align-items-center">
				<button
					class="btn btn-secondary me-2 btn-sm d-flex align-items-center"
					type="button"
					@click="restart"
				>
					<Sync class="restart me-2" />
					{{ $t("offline.restart") }}
				</button>
				{{ $t("offline.restartNeeded") }}
			</div>
			<div v-else-if="offline" class="d-flex align-items-center">
				<CloudOffline class="m-2" />
				{{ $t("offline.message") }}
			</div>
			<div
				v-else-if="showError"
				class="d-flex align-items-start container px-4 justify-content-center"
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
					<div v-if="fatal">{{ fatal.error }}</div>
				</div>
				<button
					type="button"
					class="btn-close mt-1"
					aria-label="Close"
					@click="dismissed = true"
				></button>
			</div>
		</div>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/car1";
import CloudOffline from "./MaterialIcon/CloudOffline.vue";
import Sync from "./MaterialIcon/Sync.vue";
import restart, { performRestart, restartComplete } from "../restart";

export default {
	name: "OfflineIndicator",
	components: {
		CloudOffline,
		Sync,
	},
	props: {
		offline: Boolean,
		fatal: Object,
	},
	data() {
		return { dismissed: false };
	},
	watch: {
		offline: function () {
			if (!this.offline) {
				restartComplete();
				this.dismissed = false;
			}
		},
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
	},
	methods: {
		restart() {
			performRestart();
		},
	},
};
</script>
<style scoped>
.restart {
	transform: scaleX(-1);
}
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
