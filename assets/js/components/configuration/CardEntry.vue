<template>
	<div>
		<div class="card-title">
			<h5 class="mb-0" style="display: inline-block">{{ name }}</h5>
			&nbsp;
			<a href="#" @click.prevent="editMode = true" v-show="!editMode">ändern</a>
			<div class="float-right text-right"><slot name="status"></slot></div>
		</div>
		<transition name="fade" mode="out-in">
			<div class="form pb-3" v-if="editMode">
				<slot name="form" @close="editMode = false"></slot>
			</div>
			<p v-else class="card-text pb-3">
				<slot name="summary"></slot>
			</p>
		</transition>
	</div>
</template>

<script>
export default {
	name: "CardEntry",
	props: {
		name: String,
	},
	data: function () {
		return { editMode: false };
	},
};
</script>
<style scoped>
.fade-leave {
	opacity: 1;
}
.fade-leave-active {
	transition: opacity 0.2s;
}
.fade-leave-to {
	opacity: 0;
}
.fade-enter {
	opacity: 0;
}
.fade-enter-active {
	transition: opacity 0.2s;
}
.fade-enter-to {
	opacity: 1;
}
.form {
	width: 75%;
}
</style>
