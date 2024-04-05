<template>
	<div class="group p-4">
		<div class="config-items">
			<dl class="row" data-testid="generalconfig-title">
				<dt class="col-sm-6 text-lg-end">{{ $t("config.general.title") }}</dt>
				<dd class="col-sm-6">
					{{ title || "---" }}
					<a
						href="#"
						class="ms-2 d-inline-block text-muted"
						@click.prevent="openModal('titleModal')"
					>
						{{ $t("config.general.edit") }}
					</a>
					<TitleModal ref="titleModal" @changed="load" />
				</dd>
			</dl>
			<dl class="row">
				<dt class="col-sm-6 text-lg-end">{{ $t("config.general.telemetry") }}</dt>
				<dd class="col-sm-6">
					{{ $t(`config.general.${telemetryEnabled ? "on" : "off"}`) }}
					<a
						href="#"
						class="ms-2 d-inline-block text-muted"
						@click.prevent="openModal('globalSettingsModal')"
					>
						{{ $t("config.general.change") }}
					</a>
				</dd>
			</dl>
			<dl class="row">
				<dt class="col-sm-6 text-lg-end">{{ $t("config.general.experimental") }} 🧪</dt>
				<dd class="col-sm-6">
					{{ $t(`config.general.${experimentalEnabled ? "on" : "off"}`) }}
					<a
						href="#"
						class="ms-2 d-inline-block text-muted"
						@click.prevent="openModal('globalSettingsModal')"
					>
						{{ $t("config.general.change") }}
					</a>
				</dd>
			</dl>
			<dl class="row" data-testid="generalconfig-password">
				<dt class="col-sm-6 text-lg-end">{{ $t("config.general.password") }}</dt>
				<dd class="col-sm-6">
					*******
					<a
						href="#"
						class="ms-2 d-inline-block text-muted"
						@click.prevent="openModal('passwordModal')"
					>
						{{ $t("config.general.edit") }}
					</a>
				</dd>
			</dl>
			<dl class="row wip">
				<dt class="col-sm-6 text-lg-end">API-Key</dt>
				<dd class="col-sm-6">
					*******
					<a href="#" class="ms-2 d-inline-block text-muted" @click.prevent="todo"
						>show</a
					>
				</dd>
			</dl>
			<dl class="row wip">
				<dt class="col-sm-6 text-lg-end">Sponsoring</dt>
				<dd class="col-sm-6">
					<span class="text-primary"> valid </span>
					<a href="#" class="ms-2 d-inline-block text-muted" @click.prevent="todo"
						>change</a
					>
				</dd>
			</dl>
			<dl class="row wip">
				<dt class="col-sm-6 text-lg-end">Server</dt>
				<dd class="col-sm-6">
					http://evcc.local:7070
					<a href="#" class="ms-2 d-inline-block text-muted" @click.prevent="todo"
						>edit</a
					>
				</dd>
			</dl>
			<dl class="row wip">
				<dt class="col-sm-6 text-lg-end">Update Interval</dt>
				<dd class="col-sm-6">
					30s
					<a href="#" class="ms-2 d-inline-block text-muted" @click.prevent="todo"
						>edit</a
					>
				</dd>
			</dl>
		</div>
		<hr class="mt-0 mb-4" />
		<div class="d-flex justify-content-end">
			<router-link to="/config/editor" class="btn btn-outline-primary btn-md">
				{{ $t("config.general.editEvccYaml") }}
			</router-link>
		</div>
	</div>
</template>

<script>
import Modal from "bootstrap/js/dist/modal";
import TitleModal from "./TitleModal.vue";
import api from "../../api";
import settings from "../../settings";

export default {
	name: "GeneralConfig",
	data() {
		return {
			title: "",
		};
	},
	components: { TitleModal },
	emits: ["site-changed"],
	async mounted() {
		await this.load();
	},
	computed: {
		telemetryEnabled() {
			return settings.telemetry === true;
		},
		experimentalEnabled() {
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
	border-radius: 1rem;
	box-shadow: 0 0 0 0 var(--evcc-gray-50);
	color: var(--evcc-default-text);
	background: var(--evcc-box);
	margin-bottom: 5rem;
	border: 1px solid var(--evcc-gray-50);
}

.config-items {
	padding: 1rem;
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
}

.wip {
	opacity: 0.2;
	display: none !important;
}
dt {
	margin-bottom: 0.5rem;
	hyphens: auto;
}
dd {
	margin-bottom: 1rem;
	hyphens: auto;
}
</style>
