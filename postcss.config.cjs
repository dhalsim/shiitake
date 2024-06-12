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
            .replace(`var(${decl.prop})`, decl.value)
            .replace(/rgb\((\d+) (\d+) (\d+) \/ (\d+)\)/, 'rgba($1, $2, $3, $4)')
        })
        decl.remove()
        return
      }
    }
  }
}
