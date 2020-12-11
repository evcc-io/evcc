<template>
	<div class="collapse" ref="bar">
		<div class="row p-3 bg-warning">
			<div class="col-12">
				Neue Version verfügbar! Installiert: {{ installed }}. Verfügbar:
				{{ availableVersion }}.
				<b
					class="px-3"
					data-toggle="collapse"
					data-target="#release-notes"
					v-if="releaseNotes"
				>
					<a href="#" class="text-body">
						Release notes
						<fa-icon icon="chevron-up" v-if="notesShown"></fa-icon>
						<fa-icon icon="chevron-down" v-else></fa-icon>
					</a>
				</b>
				<b class="px-3">
					<a
						:href="'https://github.com/andig/evcc/releases/tag/' + availableVersion"
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
					data-toggle="collapse"
					data-target="#version-bar"
				>
					<span aria-hidden="true">&times;</span>
				</button>
			</div>
		</div>
		<div class="row p-3 bg-light collapse" id="release-notes" ref="notes">
			<div class="col-12" v-html="releaseNotes"></div>
		</div>
	</div>
</template>

<script>
import $ from "jquery";

export default {
	name: "Version",
	props: {
		availableVersion: String,
		releaseNotes: String,
	},
	data: function () {
		return {
			installed: window.evcc.version,
			notesShown: false,
		};
	},
	mounted: function () {
		$(this.$refs.notes)
			.on(
				"show.bs.collapse",
				function () {
					this.notesShown = true;
				}.bind(this)
			)
			.on(
				"hide.bs.collapse",
				function () {
					this.notesShown = false;
				}.bind(this)
			);
	},
	watch: {
		availableVersion: function () {
			if (
				this.installed != window.evcc.version && // go template parsed?
				this.installed != "0.0.1-alpha" && // make used?
				this.availableVersion != this.installed
			) {
				$(this.$refs.bar).collapse("show");
			}
		},
	},
};
</script>
