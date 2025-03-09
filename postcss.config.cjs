const tailwindcss = require('tailwindcss')

module.exports = {
  plugins: [
    tailwindcss(),
    gtk()
  ]
}

function gtk () {
  return {
    postcssPlugin: 'gtk',
    Declaration (decl) {
      if (decl.prop.startsWith('--')) {
        decl.parent.nodes.forEach(sibling => {
          sibling.value = sibling.value
            .replace(new RegExp(`var\\(${decl.prop}(?:, ([^)]*))?\\)`, 'g'), (match, defaultValue) => {
              return defaultValue ? `${decl.value || defaultValue}` : decl.value;
            })
            .replace(/rgb\((\d+) (\d+) (\d+) \/ (\d+)\)/, 'rgba($1, $2, $3, $4)')
            .replace(/rgb\((\d+) (\d+) (\d+) \/ var\(([^)]+)\)\)/, 'rgba($1, $2, $3, var($4))')
            .replace(/rgb\((\d+) (\d+) (\d+)\)/, 'rgb($1, $2, $3)')
        })
        decl.remove()
        return
      }
    }
  }
}
