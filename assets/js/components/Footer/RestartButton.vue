<template>
	<button
		class="btn me-2 btn-sm d-flex align-items-center"
		:class="error ? 'btn-outline-danger' : 'btn-secondary'"
		type="button"
		:disabled="restarting"
		tabindex="0"
		@click="handleRestart"
	>
		<span
			v-if="restarting"
			class="spinner-border spinner-border-sm m-1 me-2"
			role="status"
			aria-hidden="true"
		></span>
		<Sync v-else :size="iconSize" class="restart me-2" />
		{{ $t("offline.restart") }}
	</button>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import Sync from "../MaterialIcon/Sync.vue";
import { ICON_SIZE } from "@/types/evcc";

export default defineComponent({
	name: "RestartButton",
	components: {
		Sync,
	},
	props: {
		restarting: {
			type: Boolean,
			default: false,
		},
		error: {
			type: Boolean,
			default: false,
		},
	},
	emits: ["restart"],
	computed: {
		iconSize() {
			return ICON_SIZE.S;
		},
	},
	methods: {
		handleRestart() {
			if (!this.restarting) {
				this.$emit("restart");
			}
		},
	},
});
</script>

<style scoped>
.restart {
	transform: scaleX(-1);
}
</style>
