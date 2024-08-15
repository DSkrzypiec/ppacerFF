module.exports = {
  content: ["./css/**/*.{html,js}", "./views/**/*.{html,js}"],
  theme: {
    container: {
      center: true,
      padding: '2rem',
    },
    extend: {
      colors: {
        customOrange: '#FF865B',
      },
    },
  },
  plugins: [
    require('daisyui'),
  ],
  daisyui: {
    themes: ["light", "dark", "forest", "sunset"],
  },
}
