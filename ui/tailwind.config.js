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
        hover: '#f2f5f9',
        danger: {
          DEFAULT: '#d33a3a',
          deep: '#b32424',
          soft: '#fdecec',
          line: '#f0c2c2',
        },
        warning: {
          DEFAULT: '#c2620b',
          soft: '#fff7ed',
        },
        success: {
          DEFAULT: '#16a34a',
          soft: '#dcfce7',
        },
      },
      boxShadow: {
        brand: '0 12px 26px rgba(31,107,255,0.3)',
        'brand-lg': '0 16px 32px rgba(31,107,255,0.38)',
        badge: '0 14px 30px rgba(31,107,255,0.35)',
        card: '0 6px 20px rgba(15,39,72,0.08)',
      },
      keyframes: {
        shimmer: {
          '100%': { transform: 'translateX(100%)' },
        },
      },
      animation: {
        shimmer: 'shimmer 1.4s infinite',
      },
      fontFamily: {
        mono: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'monospace'],
      },
    },
  },
  plugins: [],
}
