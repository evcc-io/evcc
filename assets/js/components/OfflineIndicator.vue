<template>
	<div>
		<div class="modal-backdrop" v-if="offline" />
		<div
			class="fixed-bottom alert alert-secondary d-flex justify-content-center align-items-center mb-0 rounded-0 p-2"
			:class="{ visible: visible }"
			role="alert"
		>
			<div v-if="restarting" class="d-flex align-items-center">
				<button
					class="btn btn-outline-primary me-2 btn-sm d-flex align-items-center"
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
					class="btn btn-outline-primary me-2 btn-sm d-flex align-items-center"
					type="button"
					@click="restart"
				>
					<Sync class="me-2" />
					{{ $t("offline.restart") }}
				</button>
				{{ $t("offline.restartNeeded") }}
			</div>
			<div v-else-if="offline" class="d-flex align-items-center">
				<CloudOffline class="m-2" />
				{{ $t("offline.message") }}
			</div>
		</div>
	</div>
</template>

<script>
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
	},
	watch: {
		offline: function () {
			if (!this.offline) {
				restartComplete();
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
			return this.offline || this.restartNeeded || this.restarting;
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
.fixed-bottom {
	/* $zindex-toast https://getbootstrap.com/docs/5.3/layout/z-index/ */
	z-index: 1090 !important;
}
.spin {
	animation: rotation 1s infinite cubic-bezier(0.37, 0, 0.63, 1);
}
@keyframes rotation {
	from {
		transform: rotate(0deg) scaleX(-1);
	}
	to {
		transform: rotate(360deg) scaleX(-1);
	}
}
.alert {
	transform: translateY(100%);
	transition: transform var(--evcc-transition-fast) ease-in;
	min-height: 58px;
}
.alert.visible {
	transform: translateY(0);
}
</style>
