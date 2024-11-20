<template>
	<svg :style="svgStyle" viewBox="0 0 24 24">
		<path
			fill="currentColor"
			d="M15 19v-1h-2q-.825 0-1.412-.587T11 16V8H9v1q0 .825-.587 1.413T7 11H4q-.825 0-1.412-.587T2 9V5q0-.825.588-1.412T4 3h3q.825 0 1.413.588T9 5v1h6V5q0-.825.588-1.412T17 3h3q.825 0 1.413.588T22 5v4q0 .825-.587 1.413T20 11h-3q-.825 0-1.412-.587T15 9V8h-2v8h2v-1q0-.825.588-1.412T17 13h3q.825 0 1.413.588T22 15v4q0 .825-.587 1.413T20 21h-3q-.825 0-1.412-.587T15 19M4 5v4zm13 10v4zm0-10v4zm0 4h3V5h-3zm0 10h3v-4h-3zM4 9h3V5H4z"
		/>
	</svg>
</template>

<script>
import icon from "../../mixins/icon";

export default {
	name: "ModbusProxy",
	mixins: [icon],
};
</script>
