<template>
	<div>
		<TestResult
			v-if="testState"
			v-bind="testState"
			:sponsor-token-required="sponsorTokenRequired"
			@test="$emit('test')"
		/>

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
				:class="buttonClass"
				:disabled="testState.isRunning || isSaving || isSucceeded || sponsorTokenRequired"
				tabindex="0"
				@click.prevent="handleSave"
			>
				<span
					v-if="isSaving"
					class="spinner-border spinner-border-sm me-2"
					role="status"
					aria-hidden="true"
				></span>
				<template v-if="isSucceeded">{{ $t("config.general.saved") }}</template>
				<template v-else>{{ saveButtonLabel }}</template>
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
		isSucceeded: Boolean as PropType<boolean>,
		isNew: Boolean as PropType<boolean>,
		sponsorTokenRequired: Boolean as PropType<boolean>,
	},
	emits: ["save", "remove", "test"],
	computed: {
		saveButtonLabel(): string {
			const { isError, isUnknown, isRunning } = this.testState;
			if (isRunning) return this.$t("config.validation.running");
			if (this.isSaving) return this.$t("config.general.saving");
			if (isError) return this.$t("config.general.forceSave");
			if (isUnknown) return this.$t("config.general.validateSave");
			return this.$t("config.general.save");
		},
		buttonClass(): string {
			if (this.isSucceeded) return "btn btn-succeeded";
			if (this.testState.isError) return "btn btn-danger";
			return "btn btn-primary";
		},
	},
	methods: {
		handleSave(): void {
			const force = this.testState.isError;
			this.$emit("save", force);
		},
	},
});
</script>
