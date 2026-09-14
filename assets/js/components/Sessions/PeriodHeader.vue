<template>
	<div class="header-outer sticky-top">
		<div class="container px-4">
			<div class="row py-3 py-sm-3 d-flex flex-column flex-sm-row gap-3 gap-lg-0 mb-lg-2">
				<div class="col-lg-5 d-flex mb-lg-0">
					<slot name="period" />
				</div>
				<div v-if="$slots['navigator']" class="col-lg-6 offset-lg-1">
					<slot name="navigator" />
				</div>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";

export default defineComponent({
	name: "PeriodHeader",
});
</script>

<style scoped>
@import "../../../css/breakpoints.css";

.header-outer {
	--vertical-shift: 0rem;
	left: 0;
	right: 0;
	top: max(0rem, env(safe-area-inset-top)) !important;
	margin: 0 calc(calc(1.5rem + var(--vertical-shift)) * -1);
	background-color: var(--evcc-background);
	box-shadow: 0 1px 8px 0px var(--evcc-background);
}

@supports (backdrop-filter: blur(1px)) {
	.header-outer {
		background-color: #0000;
		backdrop-filter: var(--evcc-backdrop-blur);
	}
}

@media (--sm-and-up) {
	.header-outer {
		--vertical-shift: calc((100vw - 560px) / 2);
	}
}

@media (--md-and-up) {
	.header-outer {
		--vertical-shift: calc((100vw - 740px) / 2);
	}
}

@media (--lg-and-up) {
	.header-outer {
		--vertical-shift: calc((100vw - 980px) / 2);
	}
}

@media (--xl-and-up) {
	.header-outer {
		--vertical-shift: calc((100vw - 1160px) / 2);
	}
}

@media (--xxl-and-up) {
	.header-outer {
		--vertical-shift: calc((100vw - 1340px) / 2);
	}
}
</style>
