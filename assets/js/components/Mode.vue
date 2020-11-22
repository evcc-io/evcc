<template>
	<div class="btn-group btn-group-toggle bg-white shadow-none">
		<label
			class="btn btn-outline-primary value disabled caption font-weight-bold"
			v-if="caption"
		>
			Modus
		</label>
		<label
			class="btn btn-outline-primary"
			v-bind:class="{ active: mode == 'off', first: !caption }"
		>
			<input type="radio" value="off" v-on:click="targetMode('off')" />Stop
		</label>
		<label class="btn btn-outline-primary" v-bind:class="{ active: mode == 'now' }">
			<input type="radio" value="now" v-on:click="targetMode('now')" />Sofort
		</label>
		<label class="btn btn-outline-primary" v-bind:class="{ active: mode == 'minpv' }" v-if="pv">
			<input type="radio" value="minpv" v-on:click="targetMode('minpv')" />
			<span class="d-inline d-lg-none">Min</span>
			<span class="d-none d-lg-inline">Min + PV</span>
		</label>
		<label class="btn btn-outline-primary" v-bind:class="{ active: mode == 'pv' }" v-if="pv">
			<input type="radio" value="pv" v-on:click="targetMode('pv')" />
			<span class="d-inline d-md-none">PV</span>
			<span class="d-none d-md-inline">Nur PV</span>
		</label>
	</div>
</template>

<script>
export default {
	name: "Mode",
	props: ["mode", "pv", "caption"],
	methods: {
		targetMode: function (mode) {
			this.$emit("updated", mode);
		},
	},
};
</script>
