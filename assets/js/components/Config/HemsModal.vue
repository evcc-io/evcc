<template>
	<YamlModal
		name="hems"
		:title="$t('config.hems.title')"
		:description="$t('config.hems.description')"
		docs="/docs/features/14a-enwg-steuve"
		:defaultYaml="defaultYaml"
		endpoint="/config/hems"
		removeKey="hems"
		:noYamlEditor="fromYaml"
		:disableSave="fromYaml"
		@changed="$emit('changed')"
		@open="loadSessions"
	>
		<template #afterDescription>
			<div
				v-if="sessionCount"
				class="alert alert-info my-4 d-flex justify-content-between align-items-start flex-wrap gap-2"
				role="alert"
				data-testid="grid-sessions"
			>
				<div>
					<span>{{ $t("config.hems.eventsRecorded", { count: sessionCount }) }}</span>
					<span class="ms-2">{{
						$t("config.hems.lastEvent", { timeAgo: formatLastEvent(lastEvent.created) })
					}}</span>
				</div>
				<a :href="csvLink" download class="alert-link text-nowrap">
					{{ $t("config.hems.downloadCsv") }}
				</a>
			</div>
			<p v-if="fromYaml" class="text-muted">
				{{ $t("config.general.fromYamlHint") }}
			</p>
		</template>
	</YamlModal>
</template>

<script>
import YamlModal from "./YamlModal.vue";
import defaultYaml from "./defaultYaml/hems.yaml?raw";
import api from "../../api";
import formatter from "../../mixins/formatter";

export default {
	name: "HemsModal",
	components: { YamlModal },
	mixins: [formatter],
	props: {
		yamlSource: String,
	},
	emits: ["changed"],
	data() {
		return {
			defaultYaml: defaultYaml.trim(),
			sessions: [],
		};
	},
	computed: {
		fromYaml() {
			return this.yamlSource === "file";
		},
		sessionCount() {
			return this.sessions.length;
		},
		lastEvent() {
			if (!this.sessions.length) {
				return null;
			}
			return this.sessions[0];
		},
		csvLink() {
			const params = new URLSearchParams({
				format: "csv",
				lang: this.$i18n?.locale,
			});
			return `./api/gridsessions?${params.toString()}`;
		},
	},
	methods: {
		async loadSessions() {
			try {
				const response = await api.get("gridsessions");
				this.sessions = response.data || [];
			} catch (e) {
				// Silently fail if no sessions available
				this.sessions = [];
				console.error(e);
			}
		},
		formatLastEvent(created) {
			const now = new Date();
			const eventDate = new Date(created);
			const diffMs = now - eventDate;
			return this.fmtTimeAgo(-diffMs);
		},
	},
};
</script>
