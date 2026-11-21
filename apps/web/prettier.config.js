/**
 * @see https://prettier.io/docs/configuration
 * @type {import("prettier").Config}
 */
export default {
    semi: true,
    singleQuote: false,
    trailingComma: "all",
    tabWidth: 4,
    overrides: [{ files: "*.vue", options: { tabWidth: 2 } }],
    plugins: ["@trivago/prettier-plugin-sort-imports"],
    importOrder: [
        "^vue$",
        "^vue-router$",
        "<THIRD_PARTY_MODULES>",
        "^@/icons$",
        "^@/components/(.*)$",
        "^@/(.*)$",
        "^[./]",
    ],
    importOrderSeparation: true,
    importOrderSortSpecifiers: true,
};
