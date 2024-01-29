<template>
	<div class="root">
		<div class="container px-4">
			<TopHeader title="Configuration Editor 🧪" />
			<div class="wrapper">
				<div class="d-flex justify-content-between my-3 align-items-baseline">
					<strong class="d-block">
						{{ path }}
						<span class="badge text-bg-secondary" v-if="!writable">read-only</span>
					</strong>
					<button
						class="btn btn-primary"
						@click="handleSave"
						:disabled="!writable || saving"
					>
						<span
							v-if="saving"
							class="spinner-border spinner-border-sm"
							role="status"
							aria-hidden="true"
						></span>
						Save
					</button>
				</div>
				<Editor v-model="content" height="calc(100vh - 200px)" />
			</div>
		</div>
	</div>
</template>

<script>
import TopHeader from "../components/TopHeader.vue";
import Editor from "../components/Config/Editor.vue";
import "@h2d2/shopicons/es/bold/arrowback";
import store from "../store";
import collector from "../mixins/collector";
import api from "../api";

export default {
	name: "ConfigEditor",
	components: { TopHeader, Editor },
	mixins: [collector],
	data() {
		return {
			content: "",
			path: "evcc.yaml",
			writable: true,
			saving: false,
		};
	},
	computed: {
		TopHeader: function () {
			const vehicleLogins = store.state.auth ? store.state.auth.vehicles : {};
			return { vehicleLogins, ...this.collectProps(TopHeader, store.state) };
		},
	},
	async mounted() {
		const res = await api.get("/config/yaml");
		const { content, path, writable } = res.data?.result || {};
		this.content = content;
		this.path = path;
		this.writable = writable;
	},
	methods: {
		async handleSave() {
			this.saving = true;
			try {
				await api.put("/config/yaml", this.content);
			} catch (e) {
				console.error(e);
			}
			this.saving = false;
		},
	},
};
</script>
<style scoped>
.container {
	max-width: 900px;
}
</style>
