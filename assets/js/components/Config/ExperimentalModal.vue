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
				v-model="hiddenFeatures"
				class="form-check-input"
				type="checkbox"
				role="switch"
			/>
			<div class="form-check-label">
				<label for="experimentalEnabled">
					{{ $t("settings.hiddenFeatures.value") }} 🧪
				</label>
			</div>
		</div>
		<div class="small text-muted">
			{{ $t("settings.deviceInfo") }}
		</div>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import GenericModal from "../Helper/GenericModal.vue";
import { getHiddenFeatures, setHiddenFeatures } from "@/featureflags.ts";

export default defineComponent({
	name: "ExperimentalModal",
	components: { GenericModal },
	data() {
		return {
			hiddenFeatures: getHiddenFeatures(),
		};
	},
	watch: {
		hiddenFeatures(value) {
			setHiddenFeatures(value);
		},
	},
});
</script>
