<template>
	<div v-if="isSupported" class="text-end mt-1">
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
import { copyToClipboard, isClipboardSupported } from "@/utils/clipboard";

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
	computed: {
		isSupported() {
			return isClipboardSupported();
		},
	},
	methods: {
		async handleCopy() {
			const success = await copyToClipboard(this.text);
			if (success) {
				this.copied = true;
				setTimeout(() => {
					this.copied = false;
				}, 2000);
			}
		},
	},
});
</script>
