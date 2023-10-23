module.exports = {
  extends: ["@commitlint/config-conventional"],
  rules: {
    "subject-case": () => [
      2,
      "never",
      ["snake-case", "pascal-case", "upper-case"],
    ],
    "body-max-line-length": [0, "always", Infinity],
  },
};
