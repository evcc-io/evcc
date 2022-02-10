module.exports = {
  env: {
    browser: true,
    node: true,
    es6: true,
  },
  extends: [
    "eslint:recommended",
    "plugin:vue/recommended",
    "plugin:prettier/recommended",
    "prettier",
    "plugin:storybook/recommended",
  ],
  rules: {
    "vue/require-default-prop": "off",
    "vue/attribute-hyphenation": "off",
  },
};
