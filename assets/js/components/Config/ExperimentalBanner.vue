<template>
	<div class="alert alert-warning my-4 pb-0" role="alert" data-testid="experimental-banner">
		<p>
			<strong>Experimental! 🧪</strong>
			The UI configuration is still in development. Found an issue? Please
			<a href="https://github.com/evcc-io/evcc/issues" target="_blank">report it on GitHub</a
			>. Need to reset? Delete <code>evcc.db</code> or run <code>evcc migrate --reset</code>.
			Note: Disabling experimental features only hides the UI features. Your configuration
			stays untouched.
		</p>
	</div>
</template>

<script>
export default {
	name: "ExperimentalBanner",
};
</script>
