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

// Mock IntersectionObserver for jsdom (needed by LazyImage, DataGrid LazyRow, etc.)
// Сразу симулируем, что все элементы в зоне видимости, чтобы DataGrid LazyRow рендерил строки
class MockIntersectionObserver {
    private callback: IntersectionObserverCallback;
    private elements: Set<Element> = new Set();

    constructor(callback: IntersectionObserverCallback) {
        this.callback = callback;
    }

    observe(target: Element): void {
        this.elements.add(target);
        // Симулируем, что элемент сразу в зоне видимости
        const entry: Partial<IntersectionObserverEntry> = {
            target,
            isIntersecting: true,
            intersectionRatio: 1,
            boundingClientRect: target.getBoundingClientRect(),
            intersectionRect: target.getBoundingClientRect(),
            rootBounds: null,
            time: Date.now(),
        };
        this.callback([entry as IntersectionObserverEntry], this as unknown as IntersectionObserver);
    }

    unobserve(target: Element): void {
        this.elements.delete(target);
    }
    disconnect(): void {
        this.elements.clear();
    }
    takeRecords(): IntersectionObserverEntry[] { return []; }
}

Object.defineProperty(window, 'IntersectionObserver', {
    writable: true,
    configurable: true,
    value: MockIntersectionObserver,
});
