/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./static/**/*.html",
    "./static/**/*.js",
    "./views/**/*.html",
    "./views/*.html",
  ],
  theme: {
    extend: {},
  },
  plugins: [
    /*require("daisyui")*/
  ],
  // daisyUi: {
  //   themes: ["light"]
  // }
};
