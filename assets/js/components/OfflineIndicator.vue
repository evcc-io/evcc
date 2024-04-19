<template>
	<div
		v-if="visible"
		class="fixed-bottom alert alert-secondary d-flex justify-content-center align-items-center mb-0 rounded-0"
		role="alert"
	>
		<div v-if="needsRestart" class="d-flex align-items-center">
			<Sync class="me-2" :class="{ spin: restarting }" />
			{{ restarting ? $t("offline.waitForRestart") : $t("offline.needsRestart") }}
			<button
				class="btn btn-outline-primary ms-2 btn-sm"
				:disabled="restarting"
				@click="restarting = true"
			>
				{{ restarting ? $t("offline.restarting") : $t("offline.restart") }}
			</button>
		</div>
		<div v-else class="d-flex align-items-center">
			<CloudOffline class="me-2" />
			{{ $t("offline.message") }}
		</div>
	</div>
</template>

<script>
import CloudOffline from "./MaterialIcon/CloudOffline.vue";
import Sync from "./MaterialIcon/Sync.vue";

export default {
	name: "OfflineIndicator",
	components: {
		CloudOffline,
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
