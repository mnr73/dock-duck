import pluginJs from "@eslint/js";
import prettierRecommended from "eslint-plugin-prettier/recommended";
import pluginVue from "eslint-plugin-vue";
import globals from "globals";
import tseslint from "typescript-eslint";
import vueParser from "vue-eslint-parser";

export default [
    { ignores: ["dist/**", "node_modules/**", "public/**"] },
    { languageOptions: { globals: globals.browser } },
    { files: ["*.config.{js,ts}"], languageOptions: { globals: globals.node } },
    pluginJs.configs.recommended,
    ...tseslint.configs.recommended,
    ...pluginVue.configs["flat/recommended"],
    {
        files: ["**/*.vue"],
        languageOptions: {
            parser: vueParser,
            parserOptions: {
                parser: tseslint.parser,
                extraFileExtensions: [".vue"],
            },
        },
    },
    {
        rules: {
            "@typescript-eslint/no-unused-vars": [
                "warn",
                { argsIgnorePattern: "^_" },
            ],
            "vue/multi-word-component-names": "off",
            "vue/require-default-prop": "off",
            // icons must come from the single module src/icons.ts
            "no-restricted-imports": [
                "error",
                {
                    patterns: [
                        {
                            group: ["~icons/*"],
                            message: "Import icons from @/icons only.",
                        },
                    ],
                },
            ],
        },
    },
    { files: ["src/icons.ts"], rules: { "no-restricted-imports": "off" } },
    prettierRecommended,
];
