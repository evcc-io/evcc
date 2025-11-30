<template>
	<div class="alert alert-success" data-testid="welcome-banner">
		<h5 class="alert-heading">Welcome to evcc!</h5>
		<p class="mb-4">Ready to get started? Choose how you'd like to configure your devices:</p>
		<div class="row g-5 mb-4">
			<div class="col-12 col-md-6">
				<strong>Configuration File</strong>
				<p>The traditional and <u>recommended</u> approach.</p>
				<p>
					Create an evcc.yaml file with all your device settings. Once set up, you can
					make day-to-day adjustments right here in the UI.
				</p>
				<p>Need help getting started?</p>
				<a
					href="https://docs.evcc.io/en/docs/installation/configuration"
					class="btn btn-outline-primary btn-sm"
					tabindex="0"
					target="_blank"
					>Configuration Guide</a
				>
			</div>
			<div class="col-12 col-md-6">
				<div>
					<strong>UI Configuration 🧪</strong>
					<p>The new but still <u>experimental</u> approach.</p>
					<p>
						Configure everything directly in this interface. While functional, we're
						still working on
						<a target="_blank" href="https://github.com/evcc-io/evcc/issues/6029"
							>improvements</a
						>
						and fixing
						<a
							target="_blank"
							href="https://github.com/evcc-io/evcc/issues?q=is%3Aissue%20state%3Aopen%20label%3Aexperimental"
							>known issues</a
						>.
					</p>
					<p>Want to try it out?</p>
					<button class="btn btn-sm" :class="buttonClass" tabindex="0" @click="toggle">
						{{ buttonLabel }}
					</button>
				</div>
			</div>
		</div>
		<p class="small mb-0">
			<strong>Note:</strong> This message disappears once you configure your first charging
			point (loadpoint).
		</p>
	</div>
</template>

<script>
import { setHiddenFeatures, getHiddenFeatures } from "@/featureflags.ts";

export default {
	name: "WelcomeBanner",
	computed: {
		buttonClass() {
			return this.hiddenFeatures() ? "btn-outline-secondary" : "btn-outline-primary";
		},
		buttonLabel() {
			return this.hiddenFeatures() ? "✔︎ enabled" : "Enable Experimental Features";
		},
	},
	methods: {
		toggle() {
			setHiddenFeatures(!this.hiddenFeatures());
		},
		hiddenFeatures() {
			return getHiddenFeatures();
		},
	},
};
</script>
