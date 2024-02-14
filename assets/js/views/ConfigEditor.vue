<template>
	<div class="root">
		<div class="container d-flex flex-column px-4">
			<TopHeader
				:entries="[{ title: this.$t('config.main.title'), to: '/config' }]"
				:title="$t('config.editor.title')"
			/>
			<div class="d-flex flex-column flex-grow-1">
				<Restart ref="restart" v-bind="restartProps" />
				<code class="fs-6 mb-3">
					<div v-for="(error, index) in errors" :key="index">{{ error }}</div>
				</code>
				<div class="d-flex justify-content-between my-3 align-items-baseline">
					<router-link to="/config" class="btn btn-outline-secondary">
						{{ $t("config.editor.back") }}
					</router-link>
					<button
						class="btn btn-primary"
						@click="handleSave"
						:disabled="!writable || saving || !dirty"
					>
						<span
							v-if="saving"
							class="spinner-border spinner-border-sm"
							role="status"
							aria-hidden="true"
						></span>
						{{ $t(`config.editor.${dirty ? "save" : "unchanged"}`) }}
					</button>
				</div>
				<p>
					<strong class="d-block">
						{{ path }}
						<span class="badge text-bg-secondary" v-if="!writable">
							{{ $t("config.editor.readOnly") }}
						</span>
					</strong>
				</p>
				<Editor
					class="editor flex-grow-1 mb-4"
					v-model="content"
					:disabled="!path"
					:error-line="errorLine"
				/>
			</div>
		</div>
	</div>
</template>

<script>
import TopHeader from "../components/TopHeader.vue";
import Editor from "../components/Config/Editor.vue";
import Restart from "../components/Config/Restart.vue";
import "@h2d2/shopicons/es/bold/arrowback";
import store from "../store";
import collector from "../mixins/collector";
import api from "../api";

export default {
	name: "ConfigEditor",
	components: { TopHeader, Editor, Restart },
	mixins: [collector],
	props: {
		offline: Boolean,
	},
	data() {
		return {
			content: "",
			originalContent: "",
			path: "evcc.yaml",
			writable: true,
			saving: false,
			loading: false,
		};
	},
	computed: {
		TopHeader: function () {
			const vehicleLogins = store.state.auth ? store.state.auth.vehicles : {};
			return { vehicleLogins, ...this.collectProps(TopHeader, store.state) };
		},
		dirty: function () {
			return this.content !== this.originalContent;
		},
		restartProps: function () {
			return this.collectProps(Restart);
		},
		errors: function () {
			return store.state.fatal;
		},
		errorLine: function () {
			return store.state.line;
		},
	},
	watch: {
		offline() {
			if (!this.offline) {
				this.load();
			}
		},
	},
	async mounted() {
		await this.load();
	},
	methods: {
		async load() {
			this.loading = true;
			const res = await api.get("/config/yaml");
			const { content, path, writable } = res.data?.result || {};
			this.content = content;
			this.originalContent = content;
			this.path = path;
			this.writable = writable;
			this.loading = false;
		},
		async handleSave() {
			this.saving = true;
			try {
				await api.put("/config/yaml", this.content);
				await this.load();
				await this.$refs.restart.loadDirty();
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
	min-height: 100vh;
}
</style>
