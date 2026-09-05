<template>
	<div>
		<h6 class="mb-3">{{ $t(`authProviders.challenge.${challenge.kind}.title`) }}</h6>
		<img
			v-if="challenge.image"
			:src="challenge.image"
			alt=""
			class="challenge-image d-block mb-3 rounded"
		/>
		<p v-if="challenge.link" class="mb-3">
			<a :href="challenge.link" target="_blank" rel="noopener">
				{{ $t("authProviders.challenge.openLogin") }}
			</a>
		</p>
		<FormRow :id="id" :label="$t(`authProviders.challenge.${challenge.kind}.label`)">
			<input
				:id="id"
				:value="modelValue"
				type="text"
				class="form-control"
				autocomplete="off"
				autocapitalize="off"
				spellcheck="false"
				@input="updateAnswer"
				@keyup.enter="$emit('submit')"
			/>
		</FormRow>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import FormRow from "./FormRow.vue";
import type { AuthChallenge } from "./utils/authProvider";

export default defineComponent({
	name: "AuthChallenge",
	components: { FormRow },
	props: {
		id: { type: String, required: true },
		challenge: { type: Object as PropType<AuthChallenge>, required: true },
		modelValue: { type: String, default: "" },
	},
	emits: ["update:modelValue", "submit"],
	methods: {
		updateAnswer(event: Event) {
			this.$emit("update:modelValue", (event.target as HTMLInputElement).value);
		},
	},
});
</script>

<style scoped>
.challenge-image {
	width: 100%;
	max-width: 300px;
}
</style>
