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
import Vue from "vue";

export default {
	name: "Toasts",
	data: function () {
		return {
			items: {},
			count: 0,
		};
	},
	methods: {
		raise: function (msg) {
			let found = false;
			Object.keys(this.items).forEach(function (k) {
				let m = this.items[k];
				if (m.type == msg.type && m.message == msg.message) {
					found = true;
				}
			}, this);
			if (!found) {
				msg.id = this.count++;
				Vue.set(this.items, msg.id, msg);
			}
		},
		error: function (msg) {
			msg.type = "error";
			this.raise(msg);
		},
		warn: function (msg) {
			msg.type = "warn";
			this.raise(msg);
		},
		remove: function (msg) {
			Vue.delete(this.items, msg.id);
		},
	},
};
</script>
