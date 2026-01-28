<template>
	<GenericModal
		id="experimentalModal"
		:title="$t('config.experimental.title')"
		data-testid="experimental-modal"
	>
		<p>{{ $t("config.experimental.description") }}</p>
		<div class="form-check form-switch my-3">
			<input
				id="experimentalEnabled"
				:checked="experimental"
				class="form-check-input"
				type="checkbox"
				role="switch"
				@change="change"
			/>
			<div class="form-check-label">
				<label for="experimentalEnabled">
					{{ $t("settings.hiddenFeatures.value") }} 🧪
				</label>
			</div>
		</div>
		<div v-if="error" class="errorMessage my-1 text-danger" v-html="error" />
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import GenericModal from "../Helper/GenericModal.vue";
import api from "@/api";
import type { AxiosError } from "axios";
import formatter from "@/mixins/formatter";

export default defineComponent({
	name: "ExperimentalModal",
	components: { GenericModal },
	mixins: [formatter],
	props: {
		experimental: Boolean,
	},
	data() {
		return {
			error: null as string | null,
		};
	},
	methods: {
		async change(e: Event) {
			try {
				this.error = null;
				await api.post(`settings/experimental/${(e.target as HTMLInputElement).checked}`);
			} catch (err) {
				const e = err as AxiosError<{ error: string }>;
				if (e.response) {
					this.error = this.parseMarkdown("**Error:** " + e.response.data.error);
				}
			}
		},
	},
});
</script>
<style>
.errorMessage :deep(pre) {
	text-overflow: ellipsis;
	font-size: 1em;
}
</style>
