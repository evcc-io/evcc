<template>
	<GenericModal
		:id="`${name}Modal`"
		ref="modal"
		:data-testid="`${name}-modal`"
		:title="title"
		:size="size"
		:config-modal-name="name"
		@open="open"
	>
		<p v-if="description || docsLink">
			<span v-if="description">{{ description + " " }}</span>
			<a v-if="docsLink" :href="docsLink" target="_blank">
				{{ $t("config.general.docsLink") }}
			</a>
		</p>
		<p v-if="error" class="text-danger">
			<span v-if="errorMessage" class="d-block">{{ errorMessage }}</span>
			{{ error }}
		</p>
		<form ref="form" class="container mx-0 px-0" @submit.prevent="save">
			<slot :values="values" :changes="!nothingChanged" :save="save"></slot>

			<div
				v-if="!noButtons"
				class="mt-4 d-flex justify-content-between gap-2 flex-column flex-sm-row"
			>
				<div
					class="d-flex justify-content-between order-2 order-sm-1 gap-2 flex-grow-1 flex-sm-grow-0"
				>
					<button
						v-if="!disableCancel"
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
					v-if="!disableSave"
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
import GenericModal from "../Helper/GenericModal.vue";
import api from "@/api";
import { docsPrefix } from "@/i18n";
import store from "@/store";
import deepClone from "@/utils/deepClone";
import { closeModal } from "@/configModal";

export default {
	name: "JsonModal",
	components: { GenericModal },
	props: {
		title: String,
		description: String,
		errorMessage: String,
		docs: String,
		endpoint: String,
		disableCancel: Boolean,
		disableRemove: Boolean,
		disableSave: Boolean,
		noButtons: Boolean,
		transformReadValues: Function,
		transformWriteValues: Function,
		stateKey: String,
		saveMethod: { type: String, default: "post" },
		storeValuesInArray: Boolean,
		size: { type: String },
		confirmRemove: String,
		name: String,
	},
	emits: ["changed", "open"],
	data() {
		return {
			saving: false,
			removing: false,
			error: "",
			values: this.storeValuesInArray ? [] : {},
			serverValues: this.storeValuesInArray ? [] : {},
		};
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
			if (this.stateKey) {
				// Support nested keys like "eebus.config"
				const keys = this.stateKey.split(".");
				this.serverValues = keys.reduce((obj, key) => obj?.[key], store.state);
			} else {
				this.serverValues = store.state;
			}
			if (this.transformReadValues) {
				this.serverValues = this.transformReadValues(this.serverValues);
			}
			// Handle null/undefined values when expecting an array or object
			if (this.serverValues == null) {
				this.serverValues = this.storeValuesInArray ? [] : {};
			}
			this.values = deepClone(this.serverValues);
		},
		async save(shouldClose = true) {
			this.saving = true;
			this.error = "";
			try {
				const trimmedValues = this.trimValues(deepClone(this.values));
				const payload = this.transformWriteValues
					? this.transformWriteValues(trimmedValues)
					: trimmedValues;
				const res = await api[this.saveMethod](this.endpoint, payload, {
					validateStatus: (code) => [200, 202, 400].includes(code),
				});
				if (res.status === 200 || res.status === 202) {
					this.$emit("changed");
					if (shouldClose) {
						await closeModal();
					} else {
						await this.load();
					}
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
			if (this.confirmRemove && !window.confirm(this.confirmRemove)) {
				return;
			}
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
		trimValues(values) {
			if (Array.isArray(values)) {
				for (let index = 0; index < values.length; index++) {
					values[index] = this.trimValues(values[index]);
				}
				return values;
			} else {
				return Object.fromEntries(
					Object.entries(values).map(([key, value]) => [
						key,
						typeof value === "string" ? value.trim() : value,
					])
				);
			}
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
