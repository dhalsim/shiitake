/** @type {import('tailwindcss').Config} */
export default {
  content: ['./**.go', '../nostr-gtk/**.go'],
  theme: {
    extend: {},
  },
  plugins: [],
  corePlugins: {
    visibility: false,
    display: false,
    boxShadow: false,
    boxShadowColor: false
  }
}
