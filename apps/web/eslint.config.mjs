// Formatting is Prettier's job: eslint-config-prettier turns off every rule
// that would disagree with it, including the Vue template ones.
import prettier from 'eslint-config-prettier/flat'
import withNuxt from './.nuxt/eslint.config.mjs'

export default withNuxt(
  {
    // shadcn-vue components are generated; they leave optional props
    // without defaults on purpose.
    files: ['app/components/ui/**'],
    rules: {
      'vue/require-default-prop': 'off',
    },
  },
  prettier,
)
