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
			<li v-if="providerLogins.length > 0" class="nav-item dropdown">
				<a
					class="nav-link dropdown-toggle"
					data-bs-toggle="dropdown"
					href="#"
					role="button"
					aria-expanded="false"
					>{{ $t("header.login") }}
					<span v-if="logoutCount > 0" class="badge bg-secondary">{{ logoutCount }}</span>
				</a>
				<ul class="dropdown-menu">
					<li v-for="login in providerLogins" :key="login.title" class="dropdown-item">
						<button
							class="dropdown-item"
							type="button"
							@click="handleProviderAuthorization(login)"
						>
							{{ login.title }}
							{{
								$t(login.loggedIn ? "main.provider.logout" : "main.provider.login")
							}}
						</button>
					</li>
				</ul>
			</li>
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
	computed: {
		logoutCount() {
			return this.providerLogins.filter((login) => !login.loggedIn).length;
		},
		providerLogins() {
			return this.store.state.auth
				? Object.entries(this.store.state.auth.vehicles).map(([k, v]) => ({
						title: k,
						loggedIn: v.authenticated,
						loginPath: v.uri + "/login",
						logoutPath: v.uri + "/logout",
				  }))
				: [];
		},
	},
};
</script>
