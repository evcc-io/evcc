<template>
	<svg :style="svgStyle" viewBox="0 0 24 24">
		<path
			fill="currentColor"
			d="M6 19v1q0 .425-.288.713T5 21H4q-.425 0-.712-.288T3 20v-8l2.1-6q.15-.45.538-.725T6.5 5h11q.475 0 .863.275T18.9 6l2.1 6v8q0 .425-.287.713T20 21h-1q-.425 0-.712-.288T18 20v-1zm-.2-9h12.4l-1.05-3H6.85zM5 12v5zm2.5 4q.625 0 1.063-.437T9 14.5t-.437-1.062T7.5 13t-1.062.438T6 14.5t.438 1.063T7.5 16m9 0q.625 0 1.063-.437T18 14.5t-.437-1.062T16.5 13t-1.062.438T15 14.5t.438 1.063T16.5 16M5 17h14v-5H5z"
		/>
	</svg>
</template>

<script>
import icon from "../../mixins/icon";

export default {
	name: "VehicleLimit",
	mixins: [icon],
};
</script>
