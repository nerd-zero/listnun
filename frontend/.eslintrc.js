module.exports = {
  root: true,
  env: {
    node: true,
    // es2022: true,
  },
  plugins: ['vue'],
  extends: [
    'eslint:recommended',
    'plugin:vue/vue3-essential',
    'plugin:vue/vue3-strongly-recommended',
    '@vue/eslint-config-airbnb',
  ],
  parser: 'vue-eslint-parser',
  parserOptions: {
    parser: '@typescript-eslint/parser',
    ecmaVersion: 2022,
    sourceType: 'module',
  },
  rules: {
    'class-methods-use-this': 'off',
    'vue/multi-word-component-names': 'off',
    'vue/quote-props': 'off',
    'vue/first-attribute-linebreak': 'off',
    'vue/no-child-content': 'off',
    'vue/max-attributes-per-line': 'off',
    'vue/html-indent': 'off',
    'vue/html-closing-bracket-newline': 'off',
    'vue/singleline-html-element-content-newline': 'off',
    'vue/max-len': ['error', {
      code: 200,
      template: 200,
      comments: 200,
    }],
    'vuejs-accessibility/label-has-for': 'off',
    'vuejs-accessibility/click-events-have-key-events': 'off',
    'vuejs-accessibility/anchor-has-content': 'off',
    'import/no-unresolved': ['error', { ignore: ['@primeuix/themes'] }],
    'import/extensions': ['error', 'ignorePackages', {
      js: 'never',
      ts: 'never',
      vue: 'always',
    }],
  },
  settings: {
    'import/resolver': {
      typescript: {
        alwaysTryTypes: true,
        project: './tsconfig.json',
      },
    },
  },
  ignorePatterns: ['src/email-builder.js'],
  overrides: [
    {
      // Cypress specs and support code run in Cypress's mocha runner.
      files: ['cypress/**/*.js'],
      env: { mocha: true },
      globals: { cy: 'readonly', Cypress: 'readonly', expect: 'readonly' },
    },
    {
      files: ['*.ts'],
      parser: '@typescript-eslint/parser',
      plugins: ['@typescript-eslint'],
      extends: [
        'plugin:@typescript-eslint/recommended',
      ],
      rules: {
        '@typescript-eslint/no-explicit-any': 'off',
        '@typescript-eslint/no-unused-vars': ['warn', { argsIgnorePattern: '^_' }],
        'no-unused-vars': 'off',
        'import/extensions': 'off',
        'import/no-unresolved': 'off',
        '@typescript-eslint/ban-ts-comment': 'off',
      },
    },
    {
      files: ['*.vue'],
      parser: 'vue-eslint-parser',
      parserOptions: {
        parser: '@typescript-eslint/parser',
        ecmaVersion: 2022,
        sourceType: 'module',
      },
      plugins: ['@typescript-eslint'],
      extends: [
        'plugin:@typescript-eslint/recommended',
      ],
      rules: {
        '@typescript-eslint/no-explicit-any': 'off',
        '@typescript-eslint/no-unused-vars': ['warn', { argsIgnorePattern: '^_' }],
        'no-unused-vars': 'off',
        'import/extensions': 'off',
        'import/no-unresolved': 'off',
        '@typescript-eslint/ban-ts-comment': 'off',
        'no-use-before-define': ['error', { functions: false, classes: false }],
        'no-restricted-syntax': 'off',
      },
    },
    {
      files: ['**/*.test.ts', 'src/test/**/*.ts'],
      rules: {
        'import/no-extraneous-dependencies': 'off',
        'import/prefer-default-export': 'off',
        'no-restricted-syntax': 'off',
        'import/first': 'off',
      },
    },
  ],
};
