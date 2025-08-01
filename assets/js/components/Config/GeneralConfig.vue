<template>
	<div class="group round-box p-4">
		<GeneralConfigEntry
			test-id="generalconfig-title"
			:label="$t('config.general.title')"
			:text="title || '---'"
			modal-id="titleModal"
		>
		</GeneralConfigEntry>

		<GeneralConfigEntry
			test-id="generalconfig-password"
			:label="$t('config.general.password')"
			text="*******"
			modal-id="passwordUpdateModal"
		/>

		<GeneralConfigEntry
			test-id="generalconfig-telemetry"
			:label="$t('config.general.telemetry')"
			:text="$t(`config.general.${telemetryEnabled ? 'on' : 'off'}`)"
			modal-id="globalSettingsModal"
		/>

		<GeneralConfigEntry
			test-id="generalconfig-experimental"
			:label="$t('config.general.experimental')"
			:text="$t(`config.general.${hiddenFeatures ? 'on' : 'off'}`)"
			modal-id="globalSettingsModal"
		/>

		<GeneralConfigEntry
			v-if="$hiddenFeatures()"
			test-id="generalconfig-sponsoring"
			:label="$t('config.sponsor.title')"
			:text="sponsorStatus.name || '---'"
			:text-class="sponsorStatus.cssClass"
			modal-id="sponsorModal"
			experimental
		>
			<template #text-prefix>
				<span
					v-if="sponsorStatus.expiresSoon && sponsorStatus.name"
					class="d-inline-block me-1 p-1 rounded-circle bg-warning rounded-circle"
				></span>
			</template>
		</GeneralConfigEntry>

		<GeneralConfigEntry
			v-if="$hiddenFeatures()"
			test-id="generalconfig-network"
			:label="$t('config.network.title')"
			:text="networkStatus"
			modal-id="networkModal"
			experimental
		/>

		<GeneralConfigEntry
			v-if="$hiddenFeatures()"
			test-id="generalconfig-control"
			:label="$t('config.control.title')"
			:text="controlStatus"
			modal-id="controlModal"
			experimental
		/>
		<TitleModal ref="titleModal" @changed="load" />
	</div>
</template>

<script>
import TitleModal from "./TitleModal.vue";
import GeneralConfigEntry from "./GeneralConfigEntry.vue";
import api from "@/api";
import settings from "@/settings";
import store from "@/store";
import formatter from "@/mixins/formatter";

export default {
	name: "GeneralConfig",
	components: { TitleModal, GeneralConfigEntry },
	mixins: [formatter],
	emits: ["site-changed"],
	data() {
		return {
			title: "",
		};
	},
	computed: {
		telemetryEnabled() {
			return store.state?.telemetry === true;
		},
		hiddenFeatures() {
			return settings.hiddenFeatures === true;
		},
		networkStatus() {
			const { host, port } = store.state?.network || {};
			return host ? `${host}:${port}` : `${port || ""}`;
		},
		controlStatus() {
			const sec = store.state?.interval;
			return sec ? this.fmtDuration(sec) : "";
		},
		sponsorStatus() {
			const sponsor = store.state?.sponsor || {};
			const { name, expiresSoon } = sponsor;
			let cssClass = "";
			if (name) {
				cssClass = expiresSoon ? "text-warning" : "text-primary";
			}

			return { name, expiresSoon, cssClass };
		},
	},
	async mounted() {
		await this.load();
	},
	methods: {
		async changed() {
			this.$emit("site-changed");
			this.load();
		},
		async load() {
			try {
				const res = await api.get("/config/site", {
					validateStatus: (code) => [200, 404].includes(code),
				});
				if (res.status === 200) {
					this.title = res.data.title;
				} else {
					console.log("TODO: implement site endpoint in config error mode");
				}
			} catch (e) {
				console.error(e);
			}
		},
	},
};
</script>

<style scoped>
.group {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(225px, 1fr));
	grid-gap: 2rem 5rem;
	margin-bottom: 5rem;
	align-items: start;
}
.wip {
	opacity: 0.2;
}
</style>
