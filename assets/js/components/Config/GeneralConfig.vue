<template>
	<div class="group round-box p-4">
		<div class="config-entry" data-testid="generalconfig-title">
			<strong class="config-label">{{ $t("config.general.title") }}</strong>
			<div class="config-text">{{ title || "---" }}</div>
			<button
				class="config-button btn btn-link"
				type="button"
				:title="$t('config.main.edit')"
				@click.prevent="openModal('titleModal')"
			>
				<EditIcon size="xs" />
				<TitleModal ref="titleModal" @changed="load" />
			</button>
		</div>
		<div class="config-entry" data-testid="generalconfig-password">
			<strong class="config-label">{{ $t("config.general.password") }}</strong>
			<div class="config-text">*******</div>
			<button
				class="config-button btn btn-link"
				type="button"
				:title="$t('config.main.edit')"
				@click.prevent="openModal('passwordModal')"
			>
				<EditIcon size="xs" />
			</button>
		</div>
		<div class="config-entry" data-testid="generalconfig-telemetry">
			<strong class="config-label">{{ $t("config.general.telemetry") }}</strong>
			<div class="config-text">
				{{ $t(`config.general.${telemetryEnabled ? "on" : "off"}`) }}
			</div>
			<button
				class="config-button btn btn-link"
				type="button"
				:title="$t('config.main.edit')"
				@click.prevent="openModal('globalSettingsModal')"
			>
				<EditIcon size="xs" />
			</button>
		</div>
		<div class="config-entry" data-testid="generalconfig-experimental">
			<strong class="config-label">{{ $t("config.general.experimental") }}</strong>
			<div class="config-text">
				{{ $t(`config.general.${hiddenFeatures ? "on" : "off"}`) }}
			</div>
			<button
				class="config-button btn btn-link"
				type="button"
				:title="$t('config.main.edit')"
				@click.prevent="openModal('globalSettingsModal')"
			>
				<EditIcon size="xs" />
			</button>
		</div>
		<div v-if="$hiddenFeatures()" class="config-entry" data-testid="generalconfig-sponsoring">
			<strong class="config-label">{{ $t("config.sponsor.title") }} 🧪</strong>
			<div class="config-text" :class="sponsorStatus.cssClass">
				<span
					v-if="sponsorStatus.expiresSoon"
					class="d-inline-block me-1 p-1 rounded-circle bg-warning rounded-circle"
				></span>
				{{ sponsorStatus.name || "---" }}
			</div>
			<button
				class="config-button btn btn-link"
				type="button"
				:title="$t('config.main.edit')"
				@click.prevent="openModal('sponsorModal')"
			>
				<EditIcon size="xs" />
			</button>
		</div>
		<div v-if="$hiddenFeatures()" class="config-entry" data-testid="generalconfig-network">
			<strong class="config-label">{{ $t("config.network.title") }} 🧪</strong>
			<div class="config-text">{{ networkStatus }}</div>
			<button
				class="config-button btn btn-link"
				type="button"
				:title="$t('config.main.edit')"
				@click.prevent="openModal('networkModal')"
			>
				<EditIcon size="xs" />
			</button>
		</div>
		<div v-if="$hiddenFeatures()" class="config-entry" data-testid="generalconfig-control">
			<strong class="config-label">{{ $t("config.control.title") }} 🧪</strong>
			<div class="config-text">{{ controlStatus }}</div>
			<button
				class="config-button btn btn-link"
				type="button"
				:title="$t('config.main.edit')"
				@click.prevent="openModal('controlModal')"
			>
				<EditIcon size="xs" />
			</button>
		</div>
	</div>
</template>

<script>
import Modal from "bootstrap/js/dist/modal";
import TitleModal from "./TitleModal.vue";
import EditIcon from "../MaterialIcon/Edit.vue";
import api from "../../api";
import settings from "../../settings";
import store from "../../store";
import formatter from "../../mixins/formatter";

export default {
	name: "GeneralConfig",
	data() {
		return {
			title: "",
		};
	},
	components: { TitleModal, EditIcon },
	mixins: [formatter],
	emits: ["site-changed"],
	async mounted() {
		await this.load();
	},
	computed: {
		telemetryEnabled() {
			return settings.telemetry === true;
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
			if (expiresSoon) {
				cssClass = "text-warning";
			} else if (name) {
				cssClass = "text-primary";
			}

			return { name, expiresSoon, cssClass };
		},
	},
	methods: {
		async changed() {
			this.$emit("site-changed");
			this.load();
		},
		async load() {
			try {
				let res = await api.get("/config/site", {
					validateStatus: (code) => [200, 404].includes(code),
				});
				if (res.status === 200) {
					this.title = res.data.result.title;
				} else {
					console.log("TODO: implement site endpoint in config error mode");
				}
			} catch (e) {
				console.error(e);
			}
		},
		todo() {
			alert("not implemented");
		},
		openModal(id) {
			const $el = document.getElementById(id);
			if ($el) {
				Modal.getOrCreateInstance($el).show();
			} else {
				console.error(`modal ${id} not found`);
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
.config-entry {
	display: flex;
	flex-wrap: nowrap;
	justify-content: space-between;
	align-items: center;
	gap: 0.5rem;
}
.config-label {
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
	flex-shrink: 1;
	flex-grow: 0;
}
.config-text {
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
	flex-shrink: 1;
	flex-grow: 1;
	text-align: right;
}
.config-button {
	margin-right: -1rem;
	flex-shrink: 0;
	color: var(--evcc-text-default);
}
</style>
