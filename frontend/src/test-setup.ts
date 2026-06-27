import '@testing-library/jest-dom';

// Mock window.matchMedia for jsdom (needed by themeStore)
Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: (query: string) => ({
        matches: false,
        media: query,
        onchange: null,
        addListener: () => {},
        removeListener: () => {},
        addEventListener: () => {},
        removeEventListener: () => {},
        dispatchEvent: () => false,
    }),
});

// Mock ResizeObserver for jsdom (needed by AssetTree AnimatedCollapse + components)
class MockResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
}

window.ResizeObserver = MockResizeObserver as any;
