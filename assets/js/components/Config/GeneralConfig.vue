<template>
	<div class="group round-box p-4">
		<GeneralConfigEntry
			test-id="generalconfig-title"
			:label="$t('config.general.title')"
			:text="title || '---'"
			@edit="openModal('title')"
		>
		</GeneralConfigEntry>

		<GeneralConfigEntry
			test-id="generalconfig-password"
			:label="$t('config.general.password')"
			text="*******"
			@edit="openModal('passwordupdate')"
		/>

		<GeneralConfigEntry
			test-id="generalconfig-telemetry"
			:label="$t('config.general.telemetry')"
			:text="$t(`config.general.${telemetryEnabled ? 'on' : 'off'}`)"
			@edit="openModal('telemetry')"
		/>

		<GeneralConfigEntry
			test-id="generalconfig-experimental"
			:label="$t('config.general.experimental')"
			:text="$t(`config.general.${experimental ? 'on' : 'off'}`)"
			@edit="openModal('experimental')"
		/>

		<GeneralConfigEntry
			test-id="generalconfig-sponsoring"
			:label="$t('config.sponsor.title')"
			:text="sponsorStatus.title"
			:text-class="sponsorStatus.textClass"
			@edit="openModal('sponsor')"
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
			test-id="generalconfig-network"
			:label="$t('config.network.title')"
			:text="networkStatus"
			@edit="openModal('network')"
		/>

		<GeneralConfigEntry
			test-id="generalconfig-control"
			:label="$t('config.control.title')"
			:text="controlStatus"
			@edit="openModal('control')"
		/>
	</div>
</template>

<script>
import GeneralConfigEntry from "./GeneralConfigEntry.vue";
import { openModal } from "@/configModal";
import store from "@/store";
import formatter from "@/mixins/formatter";

export default {
	name: "GeneralConfig",
	components: { GeneralConfigEntry },
	mixins: [formatter],
	props: {
		sponsorError: Boolean,
		experimental: Boolean,
	},
	emits: ["site-changed"],
	computed: {
		title() {
			return store.state?.siteTitle || "";
		},
		telemetryEnabled() {
			return store.state?.telemetry === true;
		},
		networkStatus() {
			return `${store.state?.network?.port ?? ""}`;
		},
		controlStatus() {
			const sec = store.state?.interval;
			return sec ? this.fmtDuration(sec) : "";
		},
		sponsorStatus() {
			const sponsor = store.state?.sponsor || {};
			const name = sponsor.status?.name;
			const expiresSoon = sponsor.status?.expiresSoon;
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
	methods: {
		openModal,
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
