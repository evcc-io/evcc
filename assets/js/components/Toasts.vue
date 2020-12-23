<template>
	<div aria-atomic="true" style="position: absolute; top: 4rem; right: 0.5rem">
		<!-- Position it -->
		<ToastMessage
			v-for="item in items"
			:item="item"
			:id="'message-id-' + item.id"
			:key="item.id"
		>
		</ToastMessage>
	</div>
</template>

<script>
import "../icons";
import ToastMessage from "./ToastMessage";

export default {
	name: "Toasts",
	components: { ToastMessage },
	props: {
		items: Object,
	},
};
</script>
