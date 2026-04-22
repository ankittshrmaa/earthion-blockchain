/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        // Earthion Green - #0cc930
        earth: {
          50: '#e6ffe6',
          100: '#ccffcc',
          200: '#99ff99',
          300: '#66ff66',
          400: '#33ff33',
          500: '#0cc930',
          600: '#0cc930',
          700: '#00b329',
          800: '#009922',
          900: '#00801b',
        },
        // Coin gold palette
        coin: {
          50: '#fefce8',
          100: '#fef9c3',
          200: '#fef08a',
          300: '#fde047',
          400: '#facc15',
          500: '#eab308',
          600: '#ca8a04',
          700: '#a16207',
          800: '#854d0e',
          900: '#713f12',
        },
      },
      backgroundImage: {
        'gradient-earth': 'linear-gradient(135deg, #e6ffe6 0%, #ccffcc 50%, #f5f5f0 100%)',
        'gradient-warm': 'linear-gradient(180deg, #fdfcf9 0%, #faf8f0 50%, #e6ffe6 100%)',
        'gradient-hero': 'linear-gradient(135deg, #e6ffe6 0%, #e8e8e3 50%, #ccffcc 100%)',
      },
    },
  },
  plugins: [],
}