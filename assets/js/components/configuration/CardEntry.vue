<template>
	<div>
		<div class="card-title">
			<h5 class="mb-0" style="display: inline-block">
				{{ name }}
			</h5>
			&nbsp;
			<a href="#" @click.prevent="$emit('open')" v-show="!editMode">ändern</a>
			<div class="float-right text-right"><slot name="status"></slot></div>
		</div>
		<transition name="fade" mode="out-in">
			<div class="form pb-3" v-if="editMode">
				<slot name="form"></slot>
			</div>
			<p v-else class="card-text pb-3">
				<slot name="summary"></slot>
				<span class="text-success" v-if="isConfigured"> ✓</span>
				<span class="text-danger" v-if="isRequired">(Konfiguration erforderlich)</span>
			</p>
		</transition>
	</div>
</template>

<script>
export default {
	name: "CardEntry",
	props: {
		name: String,
		isRequired: Boolean,
		isConfigured: Boolean,
		editMode: Boolean,
	},
};
</script>
<style scoped>
.fade-leave {
	opacity: 1;
}
.fade-leave-active {
	transition: opacity 0.1s;
}
.fade-leave-to {
	opacity: 0;
}
.fade-enter {
	opacity: 0;
}
.fade-enter-active {
	transition: opacity 0.1s;
}
.fade-enter-to {
	opacity: 1;
}
.form {
	width: 75%;
}
</style>
