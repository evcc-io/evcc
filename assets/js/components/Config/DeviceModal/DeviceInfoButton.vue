<template>
	<div class="dropdown dropdown-center">
		<button
			type="button"
			class="btn btn-link evcc-default-text p-0"
			data-bs-toggle="dropdown"
			aria-expanded="false"
		>
			<shopicon-regular-info></shopicon-regular-info>
		</button>
		<ul class="dropdown-menu">
			<li>
				<span class="dropdown-item-text font-monospace"> db:{{ id }} </span>
			</li>
		</ul>
	</div>
</template>

<script>
import { defineComponent } from "vue";
import "@h2d2/shopicons/es/regular/info";

export default defineComponent({
	name: "DeviceInfoButton",
	props: {
		id: { type: Number, required: true },
	},
});
</script>

<style scoped>
.dropdown-menu {
	min-width: auto;
}
</style>
