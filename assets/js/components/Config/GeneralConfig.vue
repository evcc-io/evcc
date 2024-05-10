<template>
	<div class="group round-box p-4">
		<div class="config-entry" data-testid="generalconfig-title">
			<strong class="config-label">{{ $t("config.general.title") }}</strong>
			<div class="config-text">{{ title || "---" }}</div>
			<button
				class="config-button btn btn-link text-secondary config-button"
				type="button"
				@click.prevent="openModal('titleModal')"
			>
				<EditIcon size="xs" />
				<TitleModal ref="titleModal" @changed="load" />
			</button>
		</div>
		<div class="config-entry" data-testid="generalconfig-password">
			<strong class="config-label">Password</strong>
			<div class="config-text">*******</div>
			<button
				class="config-button btn btn-link text-secondary"
				type="button"
				@click.prevent="openModal('passwordModal')"
			>
				<EditIcon size="xs" />
			</button>
		</div>
		<div class="config-entry" data-testid="generalconfig-telemetry">
			<strong class="config-label">Telemetry</strong>
			<div class="config-text">
				{{ telemetryEnabled ? "on" : "off" }}
			</div>
			<button
				class="config-button btn btn-link text-secondary"
				type="button"
				@click.prevent="openModal('globalSettingsModal')"
			>
				<EditIcon size="xs" />
			</button>
		</div>
		<div class="config-entry" data-testid="generalconfig-experimental">
			<strong class="config-label">Experimental</strong>
			<div class="config-text">
				{{ hiddenFeatures ? "on" : "off" }}
			</div>
			<button
				class="config-button btn btn-link text-secondary"
				type="button"
				@click.prevent="openModal('globalSettingsModal')"
			>
				<EditIcon size="xs" />
			</button>
		</div>
		<div class="config-entry wip">
			<strong class="config-label">Sponsoring</strong>
			<div class="config-text text-primary">valid</div>
			<button
				class="config-button btn btn-link text-secondary"
				type="button"
				@click.prevent="todo"
			>
				<EditIcon size="xs" />
			</button>
		</div>
		<div class="config-entry wip">
			<strong class="config-label">Server</strong>
			<div class="config-text">http://evcc.local:7070</div>
			<button
				class="config-button btn btn-link text-secondary"
				type="button"
				@click.prevent="todo"
			>
				<EditIcon size="xs" />
			</button>
		</div>
		<div class="config-entry wip">
			<strong class="config-label">Update Interval</strong>
			<div class="config-text">30s</div>
			<button
				class="config-button btn btn-link text-secondary"
				type="button"
				@click.prevent="todo"
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

export default {
	name: "GeneralConfig",
	data() {
		return {
			title: "",
		};
	},
	components: { TitleModal, EditIcon },
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
	},
	methods: {
		async changed() {
			this.$emit("site-changed");
			this.load();
		},
		async load() {
			try {
				let res = await api.get("/config/site");
				this.title = res.data.result.title;
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
}
</style>
