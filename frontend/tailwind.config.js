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
        wechat: {
          green: '#07C160',
          'green-dark': '#06AD56',
          bg: '#EDEDED',
          'bg-dark': '#111111',
          chat: '#FFFFFF',
          'chat-dark': '#1E1E1E',
          bubble: '#95EC69',
          'bubble-dark': '#1C5E2E',
          'bubble-ai': '#FFFFFF',
          'bubble-ai-dark': '#2A2A2A',
          text: '#191919',
          'text-dark': '#E5E5E5',
          'text-secondary': '#999999',
          'text-secondary-dark': '#666666',
          border: '#E5E5E5',
          'border-dark': '#2A2A2A',
          header: '#EDEDED',
          'header-dark': '#1E1E1E',
          input: '#FFFFFF',
          'input-dark': '#1E1E1E',
        },
      },
      maxWidth: {
        'chat-bubble': '70%',
      },
    },
  },
  plugins: [],
}
