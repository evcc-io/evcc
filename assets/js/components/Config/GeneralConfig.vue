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
			modal-id="telemetryModal"
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
			:text="sponsorStatus.title"
			:text-class="sponsorStatus.textClass"
			modal-id="sponsorModal"
			experimental
		>
			<template #text-prefix>
				<span
					v-if="sponsorStatus.badgeClass"
					class="d-inline-block me-1 p-1 rounded-circle"
					:class="sponsorStatus.badgeClass"
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
	props: {
		sponsorError: Boolean,
	},
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
			let textClass = "";
			let badgeClass = "";
			let title = name;
			if (name) {
				if (expiresSoon) {
					textClass = "text-warning";
					badgeClass = "bg-warning";
				} else {
					badgeClass = "bg-primary";
				}
			} else {
				title = "---";
			}

			if (this.sponsorError) {
				textClass = "text-danger";
				badgeClass = "bg-danger";
				title = this.$t("config.sponsor.invalid");
			}

			return { title, expiresSoon, textClass, badgeClass };
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
