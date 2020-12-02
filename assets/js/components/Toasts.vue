<template>
	<div aria-atomic="true" style="position: absolute; top: 4rem; right: 0.5rem">
		<!-- Position it -->
		<MessageToast
			v-for="item in items"
			v-bind:item="item"
			:id="'message-id-' + item.id"
			:key="item.id"
		>
		</MessageToast>
	</div>
</template>

<script>
import MessageToast from "./MessageToast";

export default {
	name: "Toasts",
	components: { MessageToast },
	props: ["items", "count"],
};
</script>
