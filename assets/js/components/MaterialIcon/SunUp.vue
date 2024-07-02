<template>
	<svg :style="svgStyle" viewBox="0 0 48 48">
		<path
			fill="currentColor"
			d="M14.216 26.071A9.953 9.953 0 0 1 14 24c0-5.514 4.486-10 10-10s10 4.486 10 10c0 .71-.074 1.403-.216 2.071h-4.152A6.006 6.006 0 0 0 24 18a6.006 6.006 0 0 0-5.632 8.071h-4.152ZM22 4h4v6h-4zM4 22h6v4H4zM38 22h6v4h-6zM15.732 5.68l3 5.196-3.464 2-3-5.196 3.464-2ZM40.321 12.269l2 3.466-5.196 2.999-2-3.465 5.196-3ZM7.679 12.267l5.197 2.999-2 3.466-5.197-3 2-3.465ZM32.269 5.68l3.465 2.001-3 5.195-3.465-2.001 3-5.195Z"
		/>
		<path
			fill="currentColor"
			d="M21.956 35.739V44h4.088v-8.261l3.066 3.033L32 35.913 24 28l-8 7.913 2.89 2.859 3.066-3.033Z"
			style="fill-rule: nonzero"
		/>
	</svg>
</template>

<script>
import icon from "../../mixins/icon";

export default {
	name: "SunUp",
	mixins: [icon],
};
</script>
