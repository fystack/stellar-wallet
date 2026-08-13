/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        ink: '#0f2748',
        'ink-soft': '#5b6b82',
        muted: '#8a97a8',
        brand: {
          DEFAULT: '#1f6bff',
          deep: '#0b53e6',
          soft: '#e8f0ff',
        },
        card: '#f4f6f9',
        line: '#eceef2',
      },
      boxShadow: {
        brand: '0 12px 26px rgba(31,107,255,0.3)',
        'brand-lg': '0 16px 32px rgba(31,107,255,0.38)',
        badge: '0 14px 30px rgba(31,107,255,0.35)',
      },
      fontFamily: {
        mono: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'monospace'],
      },
    },
  },
  plugins: [],
}
