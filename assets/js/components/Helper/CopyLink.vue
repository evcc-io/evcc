<template>
	<div class="text-end mt-1">
		<a v-if="!copied" href="#" class="text-primary small" @click.prevent="handleCopy">
			{{ $t("config.general.copy") }}
		</a>
		<span v-else class="text-primary small">
			{{ $t("config.general.copied") }}
		</span>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import { copyWithFeedback } from "@/utils/clipboard";

export default defineComponent({
	name: "CopyLink",
	props: {
		text: {
			type: String,
			required: true,
		},
	},
	data() {
		return {
			copied: false,
		};
	},
	methods: {
		async handleCopy() {
			await copyWithFeedback(
				this.text,
				(value) => {
					this.copied = value;
				}
			);
		},
	},
});
</script>
