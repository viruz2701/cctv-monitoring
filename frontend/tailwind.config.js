/** @type {import('tailwindcss').Config} */
export default {
    content: [
        "./index.html",
        "./src/**/*.{js,ts,jsx,tsx}",
    ],
    darkMode: 'class',
    theme: {
        extend: {
            // P3-LOW-02: Анимации для skip link (focus-visible entrance)
            keyframes: {
                'fade-slide-down': {
                    '0%': {
                        opacity: '0',
                        transform: 'translateY(-8px)',
                    },
                    '100%': {
                        opacity: '1',
                        transform: 'translateY(0)',
                    },
                },
            },
            animation: {
                'fade-slide-down': 'fade-slide-down 0.3s ease-out',
            },
        },
    },
    plugins: [],
}
