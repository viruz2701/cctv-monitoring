import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// https://vitejs.dev/config/
export default defineConfig({
    plugins: [react()],
    // Prevent vite from obscuring rust errors
    clearScreen: false,
    server: {
        port: 5173,
        strictPort: true,
    },
    envPrefix: ['VITE_', 'TAURI_'],
    build: {
        // Tauri uses chromium on Windows and webkit on macOS/Linux
        target: process.env.TAURI_PLATFORM === 'windows' ? 'chrome105' : 'safari14',
        // Don't minify for debug builds
        minify: !process.env.TAURI_DEBUG ? 'esbuild' : false,
        // Produce sourcemaps for debug builds
        sourcemap: !!process.env.TAURI_DEBUG,
    },
});
