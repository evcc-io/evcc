<template>
	<div class="btn-group btn-group-toggle bg-white shadow-none">
		<label class="btn btn-outline-primary disabled caption font-weight-bold" v-if="caption">
			Ladeziel
		</label>
		<label
			class="btn btn-outline-primary"
			v-for="(level, id) in levelsOrDefault"
			:key="id"
			v-bind:level="level"
			:id="id"
			:class="{ active: soc == level, first: !caption && id == 0 }"
		>
			<input type="radio" v-bind:value="level" v-on:click="targetSoC(level)" />{{ level }}%
		</label>
	</div>
</template>

<script>
export default {
	name: "Soc",
	props: ["soc", "caption", "levels"],
	computed: {
		levelsOrDefault: function () {
			if (this.levels == null || this.levels.length == 0) {
				return []; // disabled, or use 30, 50, 80, 100
			}
			return this.levels;
		},
	},
	methods: {
		targetSoC: function (mode) {
			this.$emit("updated", mode);
		},
	},
};
</script>
