<template>
	<a
		v-if="commit"
		:href="githubHashUrl"
		target="_blank"
		rel="noopener noreferrer"
		class="btn btn-link px-0 text-decoration-none evcc-default-text text-nowrap d-flex align-items-end text-truncate"
	>
		<Logo class="logo me-2 flex-shrink-0" />
		<span class="text-decoration-underline text-truncate">v{{ installed }}</span>
		<shopicon-regular-moonstars
			class="ms-2 text-gray-light flex-shrink-0"
		></shopicon-regular-moonstars>
	</a>
	<button
		v-else-if="newVersionAvailable"
		class="btn btn-link px-0 text-decoration-none evcc-default-text text-nowrap d-flex align-items-end text-truncate"
		@click="openModal"
	>
		<shopicon-regular-gift class="me-2"></shopicon-regular-gift>
		<span class="text-decoration-underline text-truncate">v{{ installed }}</span>
		<span class="ms-2 d-none d-sm-block text-gray-medium text-decoration-underline">
			{{ $t("footer.version.availableLong") }}
		</span>
	</button>
	<a
		v-else
		:href="releaseNotesUrl(installed)"
		target="_blank"
		rel="noopener noreferrer"
		class="btn btn-link evcc-default-text px-0 text-decoration-none text-nowrap d-flex align-items-end text-truncate"
	>
		<Logo class="logo me-2 flex-shrink-0" />
		<span class="text-decoration-underline text-truncate">v{{ installed }}</span>
	</a>
</template>

<script lang="ts">
import Modal from "bootstrap/js/dist/modal";
import "@h2d2/shopicons/es/regular/gift";
import "@h2d2/shopicons/es/regular/moonstars";
import Logo from "./Logo.vue";
import { isNewVersionAvailable } from "@/utils/version";
import { defineComponent } from "vue";

export default defineComponent({
	name: "Version",
	components: { Logo },
	props: {
		installed: String,
		available: String,
		commit: String,
	},
	computed: {
		githubHashUrl() {
			return `https://github.com/evcc-io/evcc/commit/${this.commit}`;
		},
		newVersionAvailable() {
			return isNewVersionAvailable(this.installed, this.available);
		},
	},
	methods: {
		releaseNotesUrl(version?: string) {
			return version == "0.0.0"
				? `https://github.com/evcc-io/evcc/releases`
				: `https://github.com/evcc-io/evcc/releases/tag/${version}`;
		},
		openModal() {
			const modal = Modal.getOrCreateInstance(
				document.getElementById("aboutModal") as HTMLElement
			);
			modal.show();
		},
	},
});
</script>

<style scoped>
.icon {
	color: var(--evcc-dark-green);
}
.logo {
	height: 1.1rem;
	margin: 0.3rem 0 0.2rem;
}
</style>
