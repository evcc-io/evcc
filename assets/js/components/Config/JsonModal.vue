<template>
	<GenericModal ref="modal" :title="title" @open="open">
		<p v-if="description || docsLink">
			<span v-if="description">{{ description + " " }}</span>
			<a v-if="docsLink" :href="docsLink" target="_blank">
				{{ $t("config.general.docsLink") }}
			</a>
		</p>
		<p class="text-danger" v-if="error">
			<span v-if="errorMessage" class="d-block">{{ errorMessage }}</span>
			{{ error }}
		</p>
		<form ref="form" class="container mx-0 px-0" @submit.prevent="save">
			<slot :values="values"></slot>

			<div
				v-if="!noButtons"
				class="mt-4 d-flex justify-content-between gap-2 flex-column flex-sm-row"
			>
				<div
					class="d-flex justify-content-between order-2 order-sm-1 gap-2 flex-grow-1 flex-sm-grow-0"
				>
					<button
						type="button"
						class="btn btn-link text-muted btn-cancel"
						data-bs-dismiss="modal"
					>
						{{ $t("config.general.cancel") }}
					</button>
					<button
						v-if="!disableRemove"
						type="button"
						class="btn btn-link text-danger"
						:disabled="removing"
						@click="remove"
					>
						<span
							v-if="removing"
							class="spinner-border spinner-border-sm"
							role="status"
							aria-hidden="true"
						></span>
						{{ $t("config.general.remove") }}
					</button>
				</div>

				<button
					type="submit"
					class="btn btn-primary order-1 order-sm-2 flex-grow-1 flex-sm-grow-0 px-4"
					:disabled="saving || nothingChanged"
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
import store from "../../store";

export default {
	name: "JsonModal",
	components: { GenericModal },
	emits: ["changed", "open"],
	data() {
		return {
			saving: false,
			removing: false,
			error: "",
			values: {},
			serverValues: {},
		};
	},
	props: {
		title: String,
		description: String,
		errorMessage: String,
		docs: String,
		endpoint: String,
		disableRemove: Boolean,
		noButtons: Boolean,
		transformReadValues: Function,
		stateKey: String,
		saveMethod: { type: String, default: "post" },
	},
	computed: {
		docsLink() {
			return this.docs ? `${docsPrefix()}${this.docs}` : null;
		},
		nothingChanged() {
			return JSON.stringify(this.values) === JSON.stringify(this.serverValues);
		},
	},
	methods: {
		reset() {
			this.saving = false;
			this.deleting = false;
			this.error = "";
			this.values = "";
			this.serverValues = "";
		},
		async open() {
			this.$emit("open");
			this.reset();
			await this.load();
		},
		async load() {
			this.serverValues = this.stateKey ? store.state[this.stateKey] : store.state;
			if (this.transformReadValues) {
				this.serverValues = this.transformReadValues(this.serverValues);
			}
			this.values = { ...this.serverValues };
		},
		async save() {
			this.saving = true;
			this.error = "";
			try {
				const res = await api[this.saveMethod](this.endpoint, this.values, {
					validateStatus: (code) => [200, 202, 400].includes(code),
				});
				if (res.status === 200 || res.status === 202) {
					this.$emit("changed");
					this.$refs.modal.close();
				}
				if (res.status === 400) {
					this.error = res.data.error;
				}
			} catch (e) {
				console.error(e);
			}
			this.saving = false;
		},
		async remove() {
			this.removing = true;
			this.error = "";
			try {
				const res = await api.delete(this.endpoint, {
					validateStatus: (code) => [200, 400].includes(code),
				});
				if (res.status === 200) {
					this.$emit("changed");
					this.$refs.modal.close();
				}
				if (res.status === 400) {
					this.error = res.data.error;
				}
			} catch (e) {
				console.error(e);
			}
			this.removing = false;
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
.btn-cancel {
	margin-left: -0.75rem;
}
</style>
