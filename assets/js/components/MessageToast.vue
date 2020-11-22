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
import $ from "jquery";

export default {
	name: "MessageToast",
	props: ["item"],
	mounted: function () {
		const id = "#message-id-" + this.item.id;
		$(id).toast("show");
		$(id).on(
			"hidden.bs.toast",
			function () {
				window.toasts.remove(this.item);
			}.bind(this)
		);
	},
};
</script>
