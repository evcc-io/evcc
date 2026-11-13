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
				<img :src="qrDataUrl" alt="QR Code" width="200" />
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
			<button v-if="canPrint" type="button" class="btn btn-outline-secondary" @click="print">
				{{ $t("config.remote.print") }}
			</button>
			<button type="button" class="btn btn-primary ms-auto" @click="$emit('done')">
				{{ $t("config.remote.done") }}
			</button>
		</div>

		<Teleport to="body">
			<div class="print-sheet d-none d-print-block">
				<p v-if="siteTitle" class="fs-4 mb-1">{{ siteTitle }}</p>
				<h1 class="fs-2 mb-5">{{ brand }} {{ $t("config.remote.title") }}</h1>
				<ol>
					<li>{{ $t("config.remote.qrInstall", { ios: "iOS", android: "Android" }) }}</li>
					<li>{{ $t("config.remote.qrScan") }}</li>
				</ol>
				<img
					v-if="qrDataUrl"
					:src="qrDataUrl"
					alt="QR Code"
					class="d-block mx-auto my-5 print-sheet-qr"
				/>
				<p>{{ $t("config.remote.manualLogin", { url: serverUrl }) }}</p>
				<dl>
					<dt>{{ $t("config.remote.username") }}</dt>
					<dd>{{ client.username }}</dd>
					<dt>{{ $t("config.remote.password") }}</dt>
					<dd class="font-monospace">{{ client.password }}</dd>
				</dl>
			</div>
		</Teleport>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import QRCode from "qrcode";
import FormRow from "../FormRow.vue";
import CopyLink from "../../Helper/CopyLink.vue";
import type { RemoteClientCreated } from "@/types/evcc";
import { isApp } from "@/utils/native";

export default defineComponent({
	name: "RemoteClientReveal",
	components: { FormRow, CopyLink },
	props: {
		client: { type: Object as PropType<RemoteClientCreated>, required: true },
		serverUrl: { type: String, required: true },
		siteTitle: String,
		brand: { type: String, default: () => window.evcc?.customBrand || "evcc" },
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
				title: this.siteTitle || "",
			});
			return `evcc://server?${params.toString()}`;
		},
		canPrint(): boolean {
			// window.print is not available in the native app webviews
			return !isApp();
		},
	},
	watch: {
		appUrl: {
			immediate: true,
			async handler(url: string) {
				if (!url) return;
				try {
					const svg = await QRCode.toString(url, { type: "svg", margin: 1 });
					this.qrDataUrl = `data:image/svg+xml,${encodeURIComponent(svg)}`;
				} catch {
					this.qrDataUrl = null;
				}
			},
		},
	},
	methods: {
		print() {
			window.print();
		},
	},
});
</script>

<style>
@media print {
	body:has(> .print-sheet) > :not(.print-sheet) {
		display: none !important;
	}
	.print-sheet {
		color: #000;
	}
	.print-sheet-qr {
		width: 60mm;
	}
}
</style>
