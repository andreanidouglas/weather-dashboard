/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["src/**/*.html", "../template/**/*.templ", "../template/**/*.go"],
  theme: {
    extend: {},
  },
  plugins: [require("@catppuccin/tailwindcss")],
};
