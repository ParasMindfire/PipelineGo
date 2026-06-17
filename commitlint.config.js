/** @type {import('@commitlint/types').UserConfig} */
module.exports = {
  extends: ['@commitlint/config-conventional'],
  rules: {
    // type must be one of these
    'type-enum': [
      2,
      'always',
      [
        'feat',     // new feature
        'fix',      // bug fix
        'docs',     // documentation only
        'style',    // formatting, no logic change
        'refactor', // code change that is neither feat nor fix
        'test',     // adding or fixing tests
        'chore',    // build system, tooling, dependencies
        'perf',     // performance improvement
        'ci',       // CI config changes
        'revert',   // revert a previous commit
      ],
    ],
    'type-case': [2, 'always', 'lower-case'],
    'subject-case': [2, 'always', 'lower-case'],
    'subject-empty': [2, 'never'],
    'subject-full-stop': [2, 'never', '.'],
    'header-max-length': [2, 'always', 100],
    'body-leading-blank': [1, 'always'],
    'footer-leading-blank': [1, 'always'],
  },
};
