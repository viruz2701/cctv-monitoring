// =============================================================================
// Lighthouse CI Configuration — CCTV Health Monitor
// =============================================================================
// Compliance: OWASP ASVS L3 V2 (Authentication), V3 (Session Management),
//             V5 (Validation), V7 (Error Handling), V11 (Business Logic)
//             ISO 27001 A.12.6 (Application Security Review)
//             IEC 62443-3-3 SR 7.8 (Security Function Verification)
// =============================================================================

/** @type {import('@lhci/utils/src/lighthouserc.js').LHCI.Config} */
const config = {
  ci: {
    // ── Collect ──────────────────────────────────────────────────
    collect: {
      staticDistDir: 'frontend/dist',
      numberOfRuns: 3,
      startServerCommand: 'npx vite preview --port 4173',
      // URL to collect (default: http://localhost:4173)
      url: ['http://localhost:4173/'],
      // Collect settings
      settings: {
        // Emulate mobile by default for stricter metrics
        // preset: 'desktop' — uncomment for desktop audit
        chromeFlags: '--no-sandbox --headless',
      },
    },

    // ── Assert ───────────────────────────────────────────────────
    assert: {
      assertions: {
        // Performance thresholds
        'categories:performance': ['warn', { minScore: 0.90 }],

        // Accessibility thresholds (WCAG AA/AAA compliance)
        'categories:accessibility': ['error', { minScore: 0.95 }],

        // Best practices threshold
        'categories:best-practices': ['warn', { minScore: 0.90 }],

        // SEO threshold
        'categories:seo': ['warn', { minScore: 0.90 }],

        // Security-sensitive audits (OWASP ASVS L3)
        // Disable old TLS versions via server config; audit as informative
        'is-on-https': ['warn'],
        'no-vulnerable-libraries': ['error'],
        'csp-xss': ['error'],
        'charset': ['error'],

        // Performance budgets
        'first-contentful-paint': ['warn', { maxNumericValue: 2500 }],
        'largest-contentful-paint': ['warn', { maxNumericValue: 4000 }],
        'total-blocking-time': ['warn', { maxNumericValue: 300 }],
        'cumulative-layout-shift': ['warn', { maxNumericValue: 0.1 }],
        'speed-index': ['warn', { maxNumericValue: 4000 }],
      },
      // Fail build if any error-level assertion fails
      preset: 'lighthouse:recommended',
    },

    // ── Upload ───────────────────────────────────────────────────
    upload: {
      target: 'filesystem',
      outputDir: 'lhci-reports',
      reportFilenamePattern: 'lighthouse-report-%%EXTENSION%%',
    },

    // ── Server (optional — for historical comparison) ────────────
    // Uncomment and configure with LHCI_SERVER_BASE_URL secret
    // to store history and get performance regression alerts.
    // server: {
    //   baseUrl: process.env.LHCI_SERVER_BASE_URL || '',
    // },
  },
};

export default config;
