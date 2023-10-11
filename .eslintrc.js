module.exports = {
  env: {
    browser: true,
    node: true,
    es6: true,
  },
  extends: [
    "plugin:vue/vue3-essential",
    "eslint:recommended",
    "@vue/eslint-config-typescript",
    "@vue/eslint-config-prettier/skip-formatting",
  ],
  parserOptions: {
    ecmaVersion: "latest",
  },
  rules: {
    "vue/require-default-prop": "off",
    "vue/attribute-hyphenation": "off",
    "vue/multi-word-component-names": "off",
    "vue/no-reserved-component-names": "off",
  },
};
