<template>
	<div
		class="label-bar"
		:class="{
			'label-bar--hide-icon': hideIcon,
			'label-bar--hidden': !value,
			'label-bar--top': top,
			'label-bar--bottom': bottom,
			'label-bar--first': first,
			'label-bar--last': last,
		}"
	>
		<div class="label-bar-scale">
			<div class="label-bar-icon">
				<slot />
			</div>
		</div>
	</div>
</template>

<script>
export default {
	name: "LabelBar",
	props: {
		value: { type: Number, default: 0 },
		hideIcon: { type: Boolean },
		top: { type: Boolean },
		bottom: { type: Boolean },
		first: { type: Boolean },
		last: { type: Boolean },
	},
};
</script>
<style scoped>
.label-bar {
	width: 0;
	margin: 0;
	padding: 10px 0;
	opacity: 1;
	overflow: hidden;
}
.label-bar-scale--hidden {
	opacity: 0;
}
.label-bar-scale {
	border: 1px solid var(--bs-gray-medium);
	height: 14px;
	background: none;
	display: flex;
	justify-content: center;
	align-items: center;
	white-space: nowrap;
	border-radius: 0;
	transition: border-radius 500ms linear;
}
.label-bar--top .label-bar-scale {
	border-top-left-radius: 10px;
	border-top-right-radius: 10px;
	border-bottom: none;
}
.label-bar--bottom .label-bar-scale {
	border-bottom-left-radius: 10px;
	border-bottom-right-radius: 10px;
	border-top: none;
}
.label-bar-icon {
	background-color: white;
	color: var(--bs-gray-dark);
	padding: 0 0.3rem;
	opacity: 1;
}
.label-bar--top .label-bar-icon {
	margin-top: -12px;
}
.label-bar--bottom .label-bar-icon {
	margin-top: 12px;
}
.label-bar--hide-icon .label-bar-icon {
	opacity: 0;
}
.label-bar--hidden {
	opacity: 0;
}
</style>
