<template>
	<div class="collapse" ref="bar">
		<div class="row p-3 bg-warning">
			<div class="col-12">
				Neue Version verfügbar! Installiert: {{ installed }}. Verfügbar:
				{{ state.availableVersion }}.
				<b
					class="px-3"
					data-toggle="collapse"
					data-target="#release-notes"
					v-if="state.releaseNotes"
				>
					<a href="#" class="text-body">
						Release notes
						<font-awesome-icon icon="chevron-up" v-if="notesShown" />
						<font-awesome-icon icon="chevron-down" v-if="!notesShown" />
					</a>
				</b>
				<b class="px-3">
					<a
						v-bind:href="
							'https://github.com/andig/evcc/releases/tag/' + state.availableVersion
						"
						class="text-body"
					>
						Download <font-awesome-icon icon="chevron-down" />
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
			<div class="col-12" v-html="state.releaseNotes"></div>
		</div>
	</div>
</template>

<script>
import $ from "jquery";

export default {
	name: "Version",
	props: ["installed"],
	data: function () {
		return {
			state: this.$root.$data.store.state,
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
		"state.availableVersion": function () {
			if (
				this.installed != window.evcc.version && // go template parsed?
				this.installed != "0.0.1-alpha" && // make used?
				this.state.availableVersion != this.installed
			) {
				$(this.$refs.bar).collapse("show");
			}
		},
	},
};
</script>
