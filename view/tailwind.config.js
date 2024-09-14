/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["src/**/*.html", "../template/**/*.templ"],
  theme: {
    extend: {},
  },
  plugins: [require("@catppuccin/tailwindcss")],
};
