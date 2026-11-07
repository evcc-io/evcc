<template>
	<template v-if="customLogo">
		<img
			:src="'./custom-logo-light'"
			class="custom-logo custom-logo--light"
			:alt="customBrand || 'evcc'"
		/>
		<img
			:src="'./custom-logo-dark'"
			class="custom-logo custom-logo--dark"
			:alt="customBrand || 'evcc'"
		/>
	</template>
	<Logo v-else class="logo" />
</template>

<script lang="ts">
import { defineComponent } from "vue";
import Logo from "./Footer/Logo.vue";

// evcc logo, or the white label replacement served by the backend
export default defineComponent({
	name: "BrandLogo",
	components: { Logo },
	props: {
		customLogo: { type: Boolean, default: () => window.evcc?.customLogo ?? false },
		customBrand: { type: String, default: () => window.evcc?.customBrand ?? "" },
	},
});
</script>

<style scoped>
.logo {
	height: 2.5rem;
}
.custom-logo {
	height: 4rem;
	width: auto;
	max-width: 100%;
	object-fit: contain;
}
.custom-logo--dark {
	display: none;
}
:root.dark .custom-logo--light {
	display: none;
}
:root.dark .custom-logo--dark {
	display: inline;
}
</style>
