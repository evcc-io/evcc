<template>
	<div
		v-if="visible"
		class="fixed-bottom alert alert-secondary d-flex justify-content-center align-items-center mb-0 rounded-0"
		role="alert"
	>
		<div v-if="needsRestart" class="d-flex align-items-center">
			<Sync v-if="restarting" class="me-2 spin" />
			<Restart v-else class="me-2" />
			{{ $t("offline.restarting") }}
			<button
				class="btn btn-outline-secondary ms-2 btn-sm"
				:disabled="restarting"
				@click="restarting = true"
			>
				{{ $t("offline.restart") }}
			</button>
		</div>
		<div v-else class="d-flex align-items-center">
			<CloudOffline class="me-2" />
			{{ $t("offline.message") }}
		</div>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/cloud";
import CloudOffline from "./MaterialIcon/CloudOffline.vue";
import Restart from "./MaterialIcon/Restart.vue";
import Sync from "./MaterialIcon/Sync.vue";

export default {
	name: "OfflineIndicator",
	components: {
		CloudOffline,
		Restart,
		Sync,
	},
	props: {
		offline: Boolean,
		needsRestart: Boolean,
	},
	data() {
		return {
			restarting: false,
		};
	},
	watch: {
		offline: function () {
			if (!this.offline) {
				this.restarting = false;
			}
		},
	},
	computed: {
		visible() {
			return this.offline || this.needsRestart || this.restarting;
		},
	},
};
</script>
<style>
.fixed-bottom {
	/* $zindex-toast https://getbootstrap.com/docs/5.3/layout/z-index/ */
	z-index: 1090 !important;
}
.spin {
	animation: rotation 1s infinite cubic-bezier(0.37, 0, 0.63, 1);
}
@keyframes rotation {
	from {
		transform: rotate(0deg);
	}
	to {
		transform: rotate(-360deg);
	}
}
</style>
