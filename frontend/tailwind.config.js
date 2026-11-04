/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        canvas: '#f3f3f3',
        surface: '#ffffff',
        ink: '#1b1b1b',
        muted: '#616161',
        faint: '#8a8a8a',
        line: '#e3e3e3',
        brand: '#0078d4',
        'brand-soft': '#eaf4fd',
        danger: '#c42b1c',
      },
      fontFamily: {
        sans: ['"Segoe UI Variable"', '"Segoe UI"', 'system-ui', 'sans-serif'],
      },
      boxShadow: {
        flyout: '0 12px 32px rgba(0, 0, 0, 0.14), 0 2px 6px rgba(0, 0, 0, 0.08)',
      },
    },
  },
  plugins: [],
};
