<template>
	<div
		v-if="dirty"
		class="alert alert-secondary d-flex justify-content-between align-items-center my-4"
		role="alert"
	>
		<div v-if="restarting"><strong>Restarting evcc.</strong> Please wait ...</div>
		<div v-else><strong>Configuration changed.</strong> Please restart to see the effect.</div>
		<button
			type="button"
			class="btn btn-outline-dark btn-sm"
			:disabled="restarting || offline"
			@click="restart"
		>
			<span
				v-if="restarting || offline"
				class="spinner-border spinner-border-sm"
				role="status"
				aria-hidden="true"
			></span>
			<span v-else> Restart </span>
		</button>
	</div>
</template>

<script>
import api from "../../api";

export default {
	name: "Restart",
	props: {
		offline: Boolean,
	},
	data() {
		return {
			dirty: false,
			restarting: false,
		};
	},
	watch: {
		offline() {
			if (!this.offline) {
				this.restarting = false;
				this.loadAll();
			}
		},
	},
	methods: {
		async loadDirty() {
			try {
				const response = await api.get("/config/dirty");
				this.dirty = response.data?.result;
			} catch (e) {
				console.error(e);
			}
		},
		async restart() {
			try {
				await api.post("shutdown");
				this.restarting = true;
			} catch (e) {
				alert("Unabled to restart server.");
			}
		},
	},
};
</script>
<style scoped></style>
