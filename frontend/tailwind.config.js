/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        gray: {
          100: 'var(--text-primary)',
          200: 'var(--text-primary)',
          300: 'var(--text-secondary)',
          400: 'var(--text-secondary)',
          500: 'var(--text-muted)',
          600: 'var(--border-color)',
          700: 'var(--bg-tertiary)',
          800: 'var(--bg-secondary)',
          900: 'var(--bg-primary)',
        },
      },
      borderColor: {
        gray: {
          600: 'var(--border-color)',
          700: 'var(--border-color)',
        },
      },
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
