<template>
	<div class="toast" data-delay="10000" v-bind:data-autohide="true">
		<div class="toast-header">
			<strong class="mr-auto" v-if="item.type != 'warn'"
				><font-awesome-icon class="text-danger" icon="exclamation-triangle" /> Error</strong
			>
			<strong class="mr-auto" v-if="item.type == 'warn'"
				><font-awesome-icon class="text-warning" icon="exclamation-triangle" />
				Warning</strong
			>
			<small v-if="item.status">HTTP {{ item.status }}</small>
			<button type="button" class="ml-2 mb-1 close" data-dismiss="toast" aria-label="Close">
				<span aria-hidden="true">&times;</span>
			</button>
		</div>
		<div class="toast-body">{{ item.message }}</div>
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
