export default {
  files: [
    'src/styles/*.css',
    'src/styles.css',
    'src/tailwind.css',
  ],
  classFiles: ['src/**/*.{ts,tsx}'],
  // Radix Popper writes these on the content and trigger at runtime.
  externalCustomProperties: [
    '--radix-dropdown-menu-content-transform-origin',
    '--radix-select-content-transform-origin',
    '--radix-select-trigger-width',
    '--radix-select-content-available-height',
    '--initial-transform',
    '--snap-point-height',
  ],
}
