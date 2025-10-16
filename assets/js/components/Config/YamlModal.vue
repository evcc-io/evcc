<template>
	<GenericModal ref="modal" :size="size" :title="title" @open="open" @close="close">
		<p v-if="description || docsLink">
			<span v-if="description">{{ description + " " }}</span>
			<a v-if="docsLink" :href="docsLink" target="_blank">
				{{ $t("config.general.docsLink") }}
			</a>
		</p>
		<p v-if="error" class="text-danger" data-testid="error">{{ error }}</p>
		<form ref="form" class="container mx-0 px-0">
			<div class="editor-container">
				<YamlEditorContainer
					v-model="yaml"
					:errorLine="errorLine"
					:removeKey="removeKey"
					:hidden="!modalVisible"
				/>
			</div>
			<slot name="extra" />

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
import GenericModal from "../Helper/GenericModal.vue";
import api from "@/api";
import { docsPrefix } from "@/i18n";
import YamlEditorContainer from "./YamlEditorContainer.vue";

export default {
	name: "YamlModal",
	components: { GenericModal, YamlEditorContainer },
	props: {
		title: String,
		description: String,
		docs: String,
		endpoint: String,
		defaultYaml: String,
		removeKey: String,
		size: { type: String, default: "xl" },
	},
	emits: ["changed"],
	data() {
		return {
			saving: false,
			error: "",
			errorLine: undefined,
			yaml: "",
			serverYaml: "",
			modalVisible: false,
		};
	},
	computed: {
		docsLink() {
			return `${docsPrefix()}${this.docs}`;
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
			this.modalVisible = true;
			await this.load();
		},
		close() {
			this.modalVisible = false;
		},
		async load() {
			try {
				const { data } = await api.get(this.endpoint);
				this.serverYaml = data;
				this.yaml = data || this.defaultYaml;
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
</style>
