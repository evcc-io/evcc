<template>
	<div class="alert alert-warning my-4" data-testid="remote-access-required">
		{{ $t("config.remote.required") }}
		<a href="#" role="button" class="text-warning" @click.prevent="open">
			{{ $t("config.remote.requiredEnable") }}
		</a>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import { openModal } from "@/configModal";

export default defineComponent({
	name: "RemoteAccessRequired",
	emits: ["navigate"],
	methods: {
		open() {
			this.$emit("navigate");
			openModal("remote");
		},
	},
});
</script>
