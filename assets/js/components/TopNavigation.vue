<template>
	<div>
		<button
			type="button"
			data-bs-toggle="dropdown"
			data-bs-target="#navbarNavAltMarkup"
			aria-controls="navbarNavAltMarkup"
			aria-expanded="false"
			aria-label="Toggle navigation"
			class="btn btn-sm btn-outline-secondary"
		>
			<shopicon-regular-menu></shopicon-regular-menu>
		</button>
		<ul class="dropdown-menu dropdown-menu-end">
			<li>
				<a class="dropdown-item" href="https://docs.evcc.io/blog/" target="_blank">
					{{ $t("header.blog") }}
				</a>
			</li>
			<li>
				<a class="dropdown-item" href="https://docs.evcc.io/docs/Home/" target="_blank">
					{{ $t("header.docs") }}
				</a>
			</li>
			<li>
				<a class="dropdown-item" href="https://github.com/evcc-io/evcc" target="_blank">
					{{ $t("header.github") }}
				</a>
			</li>
			<li>
				<a class="dropdown-item" href="https://evcc.io/" target="_blank">
					{{ $t("header.about") }}
				</a>
			</li>
		</ul>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/menu";

export default {
	name: "TopNavigation",
};
</script>
