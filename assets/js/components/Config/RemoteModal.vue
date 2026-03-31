<template>
	<GenericModal
		id="remoteModal"
		:title="$t('config.remote.title')"
		config-modal-name="remote"
		data-testid="remote-modal"
	>
		<SponsorTokenRequired v-if="!isSponsor" feature class="mt-0" />
		<p>{{ $t("config.remote.description") }}</p>
		<ErrorMessage :error="error" />
		<div class="form-check form-switch my-3">
			<input
				id="remoteEnabled"
				:checked="remote?.config?.enabled"
				class="form-check-input"
				type="checkbox"
				role="switch"
				:disabled="!isSponsor"
				@change="change"
			/>
			<div class="form-check-label">
				<label for="remoteEnabled">
					{{ $t("config.remote.enableLabel") }}
				</label>
			</div>
		</div>
		<div v-if="remote?.config?.enabled && remote?.status?.url" class="mt-3">
			<ol class="text-muted mb-2">
				<li>
					<i18n-t keypath="config.remote.qrInstall" scope="global">
						<template #ios>
							<a
								href="https://apps.apple.com/app/evcc-io/id6478510176"
								target="_blank"
								>iOS</a
							>
						</template>
						<template #android>
							<a
								href="https://play.google.com/store/apps/details?id=io.evcc.android"
								target="_blank"
								>Android</a
							>
						</template>
					</i18n-t>
				</li>
				<li>{{ $t("config.remote.qrScan") }}</li>
			</ol>
			<div class="text-center">
				<img v-if="qrDataUrl" :src="qrDataUrl" alt="QR Code" class="qr-code" />
			</div>
			<hr class="my-4" />
			<FormRow id="remoteServerUrl" :label="$t('config.remote.url')">
				<input
					id="remoteServerUrl"
					type="text"
					class="form-control border"
					:value="remote?.status?.url"
					readonly
				/>
			</FormRow>
			<div class="row">
				<div class="col">
					<FormRow id="remoteUsername" :label="$t('config.remote.username')">
						<input
							id="remoteUsername"
							type="text"
							class="form-control border"
							value="admin"
							readonly
						/>
					</FormRow>
				</div>
				<div class="col">
					<FormRow id="remotePassword" :label="$t('config.remote.password')">
						<input
							id="remotePassword"
							type="text"
							class="form-control border"
							value="secret"
							readonly
						/>
					</FormRow>
				</div>
			</div>
		</div>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import GenericModal from "../Helper/GenericModal.vue";
import ErrorMessage from "../Helper/ErrorMessage.vue";
import FormRow from "./FormRow.vue";
import SponsorTokenRequired from "./DeviceModal/SponsorTokenRequired.vue";
import QRCode from "qrcode";
import api from "@/api";
import type { Remote } from "@/types/evcc";
import type { AxiosError } from "axios";

export default defineComponent({
	name: "RemoteModal",
	components: { GenericModal, ErrorMessage, FormRow, SponsorTokenRequired },
	props: {
		remote: { type: Object as () => Remote | undefined, default: undefined },
		isSponsor: Boolean,
	},
	data() {
		return {
			error: null as string | null,
			qrDataUrl: null as string | null,
		};
	},
	computed: {
		appUrl(): string | null {
			const url = this.remote?.status?.url;
			if (!url) return null;
			const params = new URLSearchParams({
				url,
				username: "admin",
				password: "secret",
			});
			return `evcc://server?${params.toString()}`;
		},
	},
	watch: {
		appUrl: {
			immediate: true,
			async handler(val: string | null) {
				if (!val) {
					this.qrDataUrl = null;
					return;
				}
				try {
					this.qrDataUrl = await QRCode.toDataURL(val, {
						width: 200,
						margin: 1,
					});
				} catch {
					this.qrDataUrl = null;
				}
			},
		},
	},
	methods: {
		async change(e: Event) {
			const target = e.target as HTMLInputElement;
			const checked = target.checked;
			try {
				this.error = null;
				await api.post(`config/remote/${checked}`);
			} catch (err) {
				target.checked = !checked;
				const axiosErr = err as AxiosError<{ error: string }>;
				this.error = axiosErr.response?.data?.error || axiosErr.message;
			}
		},
	},
});
</script>

<style scoped>
.qr-code {
	image-rendering: pixelated;
}
</style>
