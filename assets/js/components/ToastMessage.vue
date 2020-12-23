<template>
	<div class="toast" data-delay="10000" :data-autohide="true">
		<div class="toast-header">
			<strong class="mr-auto" v-if="item.type != 'warn'"
				><fa-icon class="text-danger" icon="exclamation-triangle"></fa-icon> Error</strong
			>
			<strong class="mr-auto" v-if="item.type == 'warn'"
				><fa-icon class="text-warning" icon="exclamation-triangle"></fa-icon>
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
import "../icons";
import $ from "jquery";

export default {
	name: "ToastMessage",
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
