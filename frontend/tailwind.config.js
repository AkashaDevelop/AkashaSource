// tailwind.config.js
/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        akasha: {
          50: '#f5f0ff', 100: '#ede0ff', 200: '#d8c0ff', 300: '#b894fd',
          400: '#a78bfa', 500: '#7c3aed', 600: '#6d28d9', 700: '#5b21b6',
          800: '#4c1d95', 900: '#2e1065',
        },
        cosmic: {
          50: '#ecfeff', 100: '#cffafe', 200: '#a5f3fc', 300: '#67e8f9',
          400: '#22d3ee', 500: '#0891b2', 600: '#0e7490', 700: '#155e75',
        },
        starlight: {
          300: '#fcd34d', 400: '#fbbf24', 500: '#f59e0b', 600: '#d97706',
        },
        space: {
          900: '#06060f', 800: '#0d0a1e', 700: '#14102e',
          600: '#1e1640', 500: '#2d2060',
        },
      },
      boxShadow: {
        'akasha':    '0 4px 24px rgba(124,58,237,0.25)',
        'akasha-lg': '0 8px 40px rgba(124,58,237,0.35)',
        'glow-sm':   '0 0 12px rgba(167,139,250,0.4)',
        'glow-md':   '0 0 24px rgba(167,139,250,0.5)',
        'night-card':'0 4px 24px rgba(0,0,0,0.4)',
      },
      animation: {
        'cosmic-float': 'cosmicFloat 8s ease-in-out infinite',
        'nebula-pulse':  'nebulaPulse 4s ease-in-out infinite',
        'star-twinkle':  'starTwinkle 3s ease-in-out infinite',
        'fade-in-up':    'fadeInUp 0.5s ease-out',
      },
      keyframes: {
        cosmicFloat: {
          '0%,100%': { transform: 'translateY(0px) rotate(0deg)' },
          '50%':     { transform: 'translateY(-10px) rotate(3deg)' },
        },
        nebulaPulse: {
          '0%,100%': { opacity: '0.6', transform: 'scale(1)' },
          '50%':     { opacity: '1',   transform: 'scale(1.05)' },
        },
        starTwinkle: {
          '0%,100%': { opacity: '1',   transform: 'scale(1)' },
          '50%':     { opacity: '0.4', transform: 'scale(0.8)' },
        },
        fadeInUp: {
          from: { opacity: '0', transform: 'translateY(16px)' },
          to:   { opacity: '1', transform: 'translateY(0)' },
        },
      },
    },
  },
  darkMode: ["class", '[data-theme="dark"]'],
  plugins: [],
}
