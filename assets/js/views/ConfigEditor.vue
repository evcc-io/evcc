<template>
	<div class="root">
		<div class="container d-flex flex-column px-4">
			<TopHeader
				:entries="[{ title: this.$t('config.main.title'), to: '/config' }]"
				:title="$t('config.editor.title')"
			/>
			<div class="d-flex flex-column flex-grow-1">
				<Restart ref="restart" v-bind="restartProps" />
				<div v-if="errors.length" class="alert alert-danger my-4" role="alert">
					<p>
						<strong>{{ $t("config.editor.errorTitle") }}</strong>
						{{ $t("config.editor.errorMessage") }}
					</p>
					<code>
						<div v-for="(error, index) in errors" :key="index">{{ error }}</div>
					</code>
				</div>
				<div
					v-else-if="restarted && !saved"
					class="alert alert-success d-flex justify-content-between align-items-center my-4"
					role="alert"
				>
					<div>
						<strong>{{ $t("config.editor.successTitle") }}</strong>
						{{ $t("config.editor.successMessage") }}
					</div>
					<router-link to="/" class="btn btn-outline-dark btn-sm">
						{{ $t("config.editor.toHome") }}
					</router-link>
				</div>

				<div class="d-flex justify-content-between my-3 align-items-baseline">
					<router-link to="/config" class="btn btn-outline-secondary">
						{{ $t("config.editor.back") }}
					</router-link>
					<button
						class="btn btn-primary"
						@click="handleSave"
						:disabled="!writable || saving || !dirty || offline"
					>
						<span
							v-if="saving"
							class="spinner-border spinner-border-sm"
							role="status"
							aria-hidden="true"
						></span>
						{{ $t(`config.editor.${dirty ? "save" : saved ? "saved" : "unchanged"}`) }}
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
			saved: false,
			restarted: false,
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
			return this.saved ? [] : store.state.fatal || [];
		},
		errorLine: function () {
			return this.saved ? undefined : store.state.line;
		},
	},
	watch: {
		offline() {
			if (!this.offline) {
				this.load();
				this.restarted = true;
			}
		},
	},
	async mounted() {
		await this.load();
	},
	methods: {
		async load() {
			this.saved = false;
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
				this.saved = true;
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
	min-height: 100vh;
}
</style>
