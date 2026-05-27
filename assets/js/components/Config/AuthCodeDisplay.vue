<template>
	<FormRow
		:id="id"
		:label="$t('authProviders.authCode')"
		:help="
			computedValidityDuration
				? $t('authProviders.authCodeHelp', {
						duration: computedValidityDuration,
					})
				: undefined
		"
	>
		<input
			:id="id"
			type="text"
			class="form-control fs-2 border font-monospace"
			:value="code"
			readonly
		/>
		<CopyLink :text="code" />
	</FormRow>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import FormRow from "./FormRow.vue";
import CopyLink from "../Helper/CopyLink.vue";
import formatter from "@/mixins/formatter";

export default defineComponent({
	name: "AuthCodeDisplay",
	components: {
		FormRow,
		CopyLink,
	},
	mixins: [formatter],
	props: {
		id: { type: String, required: true },
		code: { type: String, required: true },
		expiry: { type: Date as PropType<Date | null>, default: null },
	},
	computed: {
		computedValidityDuration(): string | null {
			if (!this.expiry) return null;
			const seconds = Math.max(
				0,
				Math.floor((this.expiry.getTime() - new Date().getTime()) / 1000)
			);
			return this.fmtDurationLong(seconds);
		},
	},
});
</script>
