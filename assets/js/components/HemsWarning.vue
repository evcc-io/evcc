<template>
	<div v-if="lpcLimit" class="alert alert-warning" data-testid="hems-warning">
		<strong>{{ $t("main.hemsWarning.title") }}</strong>
		{{ $t("main.hemsWarning.description", { limit: fmtW(lpcLimit, POWER_UNIT.KW) }) }}
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import { type Circuit } from "@/types/evcc";
import formatter from "@/mixins/formatter";

export default defineComponent({
	name: "HemsWarning",
	mixins: [formatter],
	props: {
		circuits: { type: Object as PropType<Record<string, Circuit>> },
	},
	computed: {
		lpcLimit(): number | null {
			return this.circuits?.["lpc"]?.maxPower || null;
		},
	},
});
</script>
