// ═══════════════════════════════════════════════════════════════════════
// LazyImage — Unit Tests
// P1-PERF.2: Image Lazy Loading
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { render, screen } from '@testing-library/react';
import { LazyImage } from '../LazyImage';

// Mock IntersectionObserver