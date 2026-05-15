/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      typography: {
        DEFAULT: {
          css: {
            // Tailwind Typography wraps inline <code> with backtick pseudo-elements
            // by default. Suppress them -- the <code> element itself is the signal.
            'code::before': { content: '""' },
            'code::after': { content: '""' },
            code: { fontWeight: 'normal' },
          },
        },
      },
    },
  },
  plugins: [require('@tailwindcss/typography')],
}
