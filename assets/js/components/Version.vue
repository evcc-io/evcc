<template>
	<transition name="fade">
		<div v-if="active">
			<div class="row p-3 bg-warning">
				<div class="col-12">
					Neue Version verfügbar! Installiert: {{ installed }}. Verfügbar:
					{{ available }}.
					<b class="px-3" v-if="releaseNotes">
						<a href="#" class="text-body" @click="toggleReleaseNotes">
							Release notes
							<fa-icon
								icon="chevron-down"
								class="expand-icon"
								:class="{ 'expand-icon-rotated': releaseNotesExpanded }"
							>
							</fa-icon>
						</a>
					</b>
					<b class="px-3">
						<a
							:href="'https://github.com/andig/evcc/releases/tag/' + available"
							class="text-body"
						>
							Download <fa-icon icon="chevron-down"></fa-icon>
						</a>
					</b>
					<button
						type="button"
						class="close float-right"
						style="margin-top: -2px"
						aria-label="Close"
						@click="dismiss"
					>
						<span aria-hidden="true">&times;</span>
					</button>
				</div>
			</div>
			<transition name="fade">
				<div class="row p-3 bg-light" v-if="releaseNotesExpanded">
					<div class="col-12" v-html="releaseNotes"></div>
				</div>
			</transition>
		</div>
	</transition>
</template>

<script>
import "../icons";

export default {
	name: "Version",
	props: {
		installed: String,
		available: String,
		releaseNotes: String,
	},
	data: function () {
		return {
			dismissed: false,
			releaseNotesExpanded: false,
		};
	},
	methods: {
		dismiss: function () {
			this.dismissed = true;
		},
		toggleReleaseNotes: function (e) {
			e.preventDefault();
			this.releaseNotesExpanded = !this.releaseNotesExpanded;
		},
	},
	computed: {
		active: function () {
			return (
				this.installed != "[[.Version]]" && // go template parsed?
				this.installed != "0.0.1-alpha" && // make used?
				this.available != this.installed &&
				this.dismissed === false
			);
		},
	},
	watch: {
		available: function () {
			this.dismissed = false;
			this.releaseNotesExpanded = false;
		},
	},
};
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
	transition: opacity 0.25s ease-in;
}

.fade-enter,
.fade-leave-to {
	opacity: 0;
}

.expand-icon {
	transition: transform 0.25s ease-in;
	transform: rotate(0);
}
.expand-icon-rotated {
	transform: rotate(-180deg);
}
</style>
