<template>
	<GenericModal ref="modal" :size="size" :title="title" @open="open">
		<p v-if="description || docsLink">
			<span v-if="description">{{ description + " " }}</span>
			<a v-if="docsLink" :href="docsLink" target="_blank">
				{{ $t("config.general.docsLink") }}
			</a>
		</p>
		<p class="text-danger" v-if="error" data-testid="error">{{ error }}</p>
		<form ref="form" class="container mx-0 px-0">
			<div class="editor-container" :style="{ height }">
				<YamlEditor v-model="yaml" class="editor" :errorLine="errorLine" />
			</div>

			<div class="mt-4 d-flex justify-content-between">
				<button
					type="button"
					class="btn btn-link text-muted btn-cancel"
					data-bs-dismiss="modal"
				>
					{{ $t("config.general.cancel") }}
				</button>
				<button
					type="submit"
					class="btn btn-primary"
					:disabled="saving || nothingChanged"
					@click.prevent="save"
				>
					<span
						v-if="saving"
						class="spinner-border spinner-border-sm"
						role="status"
						aria-hidden="true"
					></span>
					{{ $t("config.general.save") }}
				</button>
			</div>
		</form>
	</GenericModal>
</template>

<script>
import GenericModal from "../GenericModal.vue";
import api from "../../api";
import { docsPrefix } from "../../i18n";
import YamlEditor from "./YamlEditor.vue";

export default {
	name: "YamlModal",
	components: { GenericModal, YamlEditor },
	emits: ["changed"],
	data() {
		return {
			saving: false,
			error: "",
			errorLine: undefined,
			yaml: "",
			serverYaml: "",
		};
	},
	props: {
		title: String,
		description: String,
		docs: String,
		endpoint: String,
		defaultYaml: String,
		size: { type: String, default: "xl" },
	},
	computed: {
		docsLink() {
			return `${docsPrefix()}${this.docs}`;
		},
		height() {
			return Math.max(150, this.yaml.split("\n").length * 18) + 22 + "px";
		},
		nothingChanged() {
			return this.yaml === this.serverYaml && this.yaml !== "";
		},
	},
	methods: {
		reset() {
			this.yaml = "";
			this.serverYaml = "";
			this.error = "";
			this.saving = false;
			this.errorLine = undefined;
		},
		async open() {
			this.reset();
			await this.load();
		},
		async load() {
			try {
				const { data } = await api.get(this.endpoint);
				this.serverYaml = data.result;
				this.yaml = data.result || this.defaultYaml;
			} catch (e) {
				console.error(e);
			}
		},
		async save() {
			this.saving = true;
			this.error = "";
			this.errorLine = undefined;
			try {
				const data = this.yaml === this.defaultYaml ? "" : this.yaml;
				const res = await api.post(this.endpoint, data, {
					validateStatus: (code) => [200, 400].includes(code),
				});
				if (res.status === 200) {
					this.$emit("changed");
					this.$refs.modal.close();
				}
				if (res.status === 400) {
					this.error = res.data.error;
					this.errorLine = res.data.line;
				}
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
	margin-left: calc(var(--bs-gutter-x) * -0.5);
	margin-right: calc(var(--bs-gutter-x) * -0.5);
	padding-right: 0;
}
.editor-container {
	margin: 0 -1rem 0 -1.25rem;
}
/* reset margins on lg */
@media (min-width: 992px) {
	.editor-container {
		margin: 0;
	}
}

.btn-cancel {
	margin-left: -0.75rem;
}
</style>
