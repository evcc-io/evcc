<template>
	<div>
		<ol class="text-muted mb-2">
			<li>
				<i18n-t keypath="config.remote.qrInstall" scope="global">
					<template #ios>
						<a href="https://apps.apple.com/app/evcc-io/id6478510176" target="_blank">
							iOS
						</a>
					</template>
					<template #android>
						<a
							href="https://play.google.com/store/apps/details?id=io.evcc.android"
							target="_blank"
						>
							Android
						</a>
					</template>
				</i18n-t>
			</li>
			<li>{{ $t("config.remote.qrScan") }}</li>
		</ol>
		<div class="text-center my-3">
			<a v-if="qrDataUrl" :href="appUrl" class="d-inline-block">
				<img :src="qrDataUrl" alt="QR Code" class="qr-code" />
			</a>
		</div>

		<hr class="my-4" />

		<p>
			<i18n-t keypath="config.remote.manualLogin" scope="global">
				<template #url>
					<a :href="serverUrl" target="_blank" rel="noopener">{{ serverUrl }}</a>
				</template>
			</i18n-t>
		</p>
		<FormRow id="revealUsername" :label="$t('config.remote.username')">
			<input
				id="revealUsername"
				type="text"
				class="form-control border"
				:value="client.username"
				readonly
			/>
		</FormRow>
		<FormRow id="revealPassword" :label="$t('config.remote.password')">
			<input
				id="revealPassword"
				type="text"
				class="form-control border font-monospace"
				:value="client.password"
				readonly
			/>
			<CopyLink :text="client.password" />
		</FormRow>

		<div class="mt-4 small text-muted">
			<strong class="text-evcc">{{ $t("general.note") }}</strong>
			{{ $t("config.remote.passwordOnce") }}
		</div>

		<div class="d-flex justify-content-end mt-3">
			<button type="button" class="btn btn-primary" @click="$emit('done')">
				{{ $t("config.remote.done") }}
			</button>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import QRCode from "qrcode";
import FormRow from "../FormRow.vue";
import CopyLink from "../../Helper/CopyLink.vue";
import type { RemoteClientCreated } from "@/types/evcc";

export default defineComponent({
	name: "RemoteClientReveal",
	components: { FormRow, CopyLink },
	props: {
		client: { type: Object as PropType<RemoteClientCreated>, required: true },
		serverUrl: { type: String, required: true },
	},
	emits: ["done"],
	data() {
		return {
			qrDataUrl: null as string | null,
		};
	},
	computed: {
		appUrl(): string {
			if (!this.client || !this.serverUrl) return "";
			const params = new URLSearchParams({
				url: this.serverUrl,
				username: this.client.username,
				password: this.client.password,
			});
			return `evcc://server?${params.toString()}`;
		},
	},
	watch: {
		appUrl: {
			immediate: true,
			async handler(url: string) {
				if (!url) return;
				try {
					this.qrDataUrl = await QRCode.toDataURL(url, {
						width: 200,
						margin: 1,
					});
				} catch {
					this.qrDataUrl = null;
				}
			},
		},
	},
});
</script>

<style scoped>
.qr-code {
	image-rendering: pixelated;
}
</style>
