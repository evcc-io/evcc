<template>
	<div>
		<TestResult v-if="testState" v-bind="testState" @test="$emit('test')" />

		<div class="my-4 d-flex justify-content-between">
			<button
				v-if="isDeletable"
				type="button"
				class="btn btn-link text-danger"
				tabindex="0"
				@click.prevent="$emit('remove')"
			>
				{{ $t("config.general.delete") }}
			</button>
			<button
				v-else
				type="button"
				class="btn btn-link text-muted"
				data-bs-dismiss="modal"
				tabindex="0"
			>
				{{ $t("config.general.cancel") }}
			</button>
			<button
				type="submit"
				class="btn btn-primary"
				:disabled="testState.isRunning || isSaving || sponsorTokenRequired"
				tabindex="0"
				@click.prevent="$emit('save')"
			>
				<span
					v-if="isSaving"
					class="spinner-border spinner-border-sm"
					role="status"
					aria-hidden="true"
				></span>
				{{
					testState.isUnknown
						? $t("config.general.validateSave")
						: $t("config.general.save")
				}}
			</button>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import type { PropType } from "vue";
import TestResult from "../TestResult.vue";
import { type TestState } from "../utils/test";

export default defineComponent({
	name: "DeviceModalActions",
	components: {
		TestResult,
	},
	props: {
		isDeletable: Boolean as PropType<boolean>,
		testState: {
			type: Object as PropType<TestState>,
			default: () => {},
		},
		isSaving: Boolean as PropType<boolean>,
		sponsorTokenRequired: Boolean as PropType<boolean>,
	},
	emits: ["save", "remove", "test"],
});
</script>
