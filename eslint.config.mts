import globals from "globals";
import js from "@eslint/js";
import pluginVue from "eslint-plugin-vue";
import { defineConfigWithVueTs, vueTsConfigs } from "@vue/eslint-config-typescript";
import skipFormattingConfig from "@vue/eslint-config-prettier/skip-formatting";

export default defineConfigWithVueTs(
	js.configs.recommended,
	...pluginVue.configs["flat/recommended"],
	vueTsConfigs.recommended,
	skipFormattingConfig,
	{
		languageOptions: {
			globals: {
				...globals.browser,
				...globals.node,
			},

			ecmaVersion: "latest",
			sourceType: "module",
		},

		rules: {
			"vue/require-default-prop": "off",
			"vue/attribute-hyphenation": "off",
			"vue/multi-word-component-names": "off",
			"vue/no-reserved-component-names": "off",
			/*"vue/no-undef-properties": "warn",*/
			"no-param-reassign": "error",
			"vue/block-lang": "off",
			"@typescript-eslint/no-explicit-any": "off",
		},
	}
);
