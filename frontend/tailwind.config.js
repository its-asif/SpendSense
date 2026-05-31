/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        background: 'rgb(var(--color-background-rgb) / <alpha-value>)',
        foreground: 'rgb(var(--color-foreground-rgb) / <alpha-value>)',
        card: 'rgb(var(--color-card-rgb) / <alpha-value>)',
        'card-foreground': 'rgb(var(--color-card-foreground-rgb) / <alpha-value>)',
        popover: 'rgb(var(--color-popover-rgb) / <alpha-value>)',
        'popover-foreground': 'rgb(var(--color-popover-foreground-rgb) / <alpha-value>)',
        muted: 'rgb(var(--color-muted-rgb) / <alpha-value>)',
        'muted-foreground': 'rgb(var(--color-muted-foreground-rgb) / <alpha-value>)',
        border: 'rgb(var(--color-border-rgb) / <alpha-value>)',
        input: 'rgb(var(--color-input-rgb) / <alpha-value>)',
        ring: 'rgb(var(--color-ring-rgb) / <alpha-value>)',
        destructive: 'rgb(var(--color-destructive-rgb) / <alpha-value>)',
        'destructive-foreground': 'rgb(var(--color-destructive-foreground-rgb) / <alpha-value>)',
        'dark-bg': 'rgb(var(--color-dark-bg-rgb) / <alpha-value>)',
        'dark-card': 'rgb(var(--color-dark-card-rgb) / <alpha-value>)',
        'dark-elevated': 'rgb(var(--color-dark-elevated-rgb) / <alpha-value>)',
        'accent-blue': 'rgb(var(--color-accent-blue-rgb) / <alpha-value>)',
        'accent-green': 'rgb(var(--color-accent-green-rgb) / <alpha-value>)',
        'accent-amber': 'rgb(var(--color-accent-amber-rgb) / <alpha-value>)',
        'accent-red': 'rgb(var(--color-accent-red-rgb) / <alpha-value>)',
        'accent-purple': 'rgb(var(--color-accent-purple-rgb) / <alpha-value>)',
        'text-primary': 'rgb(var(--color-text-primary-rgb) / <alpha-value>)',
        'text-secondary': 'rgb(var(--color-text-secondary-rgb) / <alpha-value>)',
        'text-muted': 'rgb(var(--color-text-muted-rgb) / <alpha-value>)',
      },
      boxShadow: {
        card: '0 4px 6px -1px rgba(0,0,0,0.3)',
        cardHover: '0 10px 15px -3px rgba(0,0,0,0.35)',
        focus: '0 0 0 3px rgba(96, 165, 250, 0.15)',
      },
      borderRadius: {
        xl: '12px',
      },
      fontFamily: {
        sans: ['-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'Roboto', 'Oxygen', 'Ubuntu', 'Cantarell', 'Fira Sans', 'Droid Sans', 'Helvetica Neue', 'sans-serif'],
        mono: ['SFMono-Regular', 'Consolas', 'Liberation Mono', 'Menlo', 'monospace'],
      },
    },
  },
  plugins: [],
};