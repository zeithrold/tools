export default {
  files: ['src/styles/*.css', 'src/styles.css'],
  // Radix Popper writes these on the content and trigger at runtime.
  externalCustomProperties: [
    '--radix-dropdown-menu-content-transform-origin',
    '--radix-select-content-transform-origin',
    '--radix-select-trigger-width',
  ],
}
