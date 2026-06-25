// ═══════════════════════════════════════════════════════════════════════
// Fuzzy Search — weighted scoring algorithm for Command Palette
// UX-14.2.6: Global Search (⌘K) v2 — improved fuzzy search
//
// Scoring tiers:
//   exact match on label  >  startsWith on label  >  contains on label  >  fuzzy on label
//   >  exact on keywords  >  fuzzy on keywords    >  fuzzy on description
//
// Priority weights: label > keywords > description
// ═══════════════════════════════════════════════════════════════════════

export interface FuzzyResult<T> {
  item: T;
  score: number;
  matchField: 'label' | 'keywords' | 'description';
  matchType: 'exact' | 'startsWith' | 'contains' | 'fuzzy';
}

export interface FuzzySearchOptions {
  /** Minimum score to include a result (default: 10) */
  threshold?: number;
  /** Fields to search (default: all) */
  fields?: ('label' | 'keywords' | 'description')[];
}

// ───────────────────────────────────────────────────────────────────────
// Core fuzzy matching (character-by-character)
// ───────────────────────────────────────────────────────────────────────

function fuzzyMatch(text: string, query: string): boolean {
  const lower = text.toLowerCase();
  const q = query.toLowerCase();
  let qi = 0;
  for (let i = 0; i < lower.length && qi < q.length; i++) {
    if (lower[i] === q[qi]) qi++;
  }
  return qi === q.length;
}

/**
 * Computes a "quality" score for a fuzzy match — rewards
 * early/consecutive matches and penalizes gaps.
 * Higher is better.
 */
function fuzzyQuality(text: string, query: string): number {
  const lower = text.toLowerCase();
  const q = query.toLowerCase();
  let score = 0;
  let qi = 0;
  let consecutive = 0;
  for (let i = 0; i < lower.length && qi < q.length; i++) {
    if (lower[i] === q[qi]) {
      qi++;
      consecutive++;
      // Bonus for consecutive matches
      if (consecutive > 1) {
        score += 2 * consecutive;
      } else {
        score += 1;
      }
      // Bonus for match after word boundary
      if (i > 0 && lower[i - 1] === ' ') {
        score += 3;
      }
    } else {
      consecutive = 0;
    }
  }
  return score;
}

// ───────────────────────────────────────────────────────────────────────
// Scoring helpers
// ───────────────────────────────────────────────────────────────────────

const SCORE = {
  EXACT_LABEL: 100,
  STARTSWITH_LABEL: 80,
  CONTAINS_LABEL: 60,
  FUZZY_LABEL: 40,
  EXACT_KEYWORD: 30,
  FUZZY_KEYWORD: 20,
  FUZZY_DESCRIPTION: 10,
} as const;

function scoreLabelMatch(
  label: string,
  queryLower: string
): { score: number; matchType: 'exact' | 'startsWith' | 'contains' | 'fuzzy' } | null {
  const lowerLabel = label.toLowerCase();

  // Exact match (case-insensitive)
  if (lowerLabel === queryLower) {
    return { score: SCORE.EXACT_LABEL, matchType: 'exact' };
  }

  // Starts with
  if (lowerLabel.startsWith(queryLower)) {
    return { score: SCORE.STARTSWITH_LABEL, matchType: 'startsWith' };
  }

  // Contains
  if (lowerLabel.includes(queryLower)) {
    return { score: SCORE.CONTAINS_LABEL, matchType: 'contains' };
  }

  // Fuzzy
  if (fuzzyMatch(label, queryLower)) {
    const quality = fuzzyQuality(label, queryLower);
    return { score: SCORE.FUZZY_LABEL + quality, matchType: 'fuzzy' };
  }

  return null;
}

function scoreKeywordMatch(
  keywords: string[],
  queryLower: string
): { score: number; matchType: 'exact' | 'fuzzy' } | null {
  let bestScore = 0;
  let bestType: 'exact' | 'fuzzy' = 'fuzzy';

  for (const kw of keywords) {
    const lowerKw = kw.toLowerCase();

    if (lowerKw === queryLower || lowerKw.includes(queryLower)) {
      if (SCORE.EXACT_KEYWORD > bestScore) {
        bestScore = SCORE.EXACT_KEYWORD;
        bestType = 'exact';
      }
    } else if (fuzzyMatch(kw, queryLower)) {
      const quality = fuzzyQuality(kw, queryLower);
      const candidate = SCORE.FUZZY_KEYWORD + quality;
      if (candidate > bestScore) {
        bestScore = candidate;
        bestType = 'fuzzy';
      }
    }
  }

  return bestScore > 0 ? { score: bestScore, matchType: bestType } : null;
}

function scoreDescriptionMatch(
  description: string | undefined,
  queryLower: string
): { score: number; matchType: 'fuzzy' } | null {
  if (!description) return null;

  if (fuzzyMatch(description, queryLower)) {
    const quality = fuzzyQuality(description, queryLower);
    return { score: SCORE.FUZZY_DESCRIPTION + quality, matchType: 'fuzzy' };
  }

  return null;
}

// ───────────────────────────────────────────────────────────────────────
// Main search function
// ───────────────────────────────────────────────────────────────────────

export interface SearchableItem {
  id: string;
  label: string;
  description?: string;
  keywords: string[];
}

/**
 * Performs a weighted fuzzy search across items.
 * Returns results sorted by score (descending), filtered by threshold.
 *
 * @param items — array of items to search
 * @param query — raw search string (will be trimmed + lowercased internally)
 * @param options — threshold and field selection
 * @returns sorted array of FuzzyResult
 */
export function weightedFuzzySearch<T extends SearchableItem>(
  items: T[],
  query: string,
  options: FuzzySearchOptions = {}
): FuzzyResult<T>[] {
  const { threshold = 10, fields = ['label', 'keywords', 'description'] } = options;
  const trimmed = query.trim();
  if (!trimmed) {
    // Return all items with neutral score
    return items.map((item) => ({
      item,
      score: 0,
      matchField: 'label' as const,
      matchType: 'contains' as const,
    }));
  }

  const queryLower = trimmed.toLowerCase();
  const results: FuzzyResult<T>[] = [];

  for (const item of items) {
    let bestScore = -Infinity;
    let bestField: 'label' | 'keywords' | 'description' = 'label';
    let bestMatchType: 'exact' | 'startsWith' | 'contains' | 'fuzzy' = 'contains';

    // 1. Label (highest priority)
    if (fields.includes('label')) {
      const labelResult = scoreLabelMatch(item.label, queryLower);
      if (labelResult && labelResult.score > bestScore) {
        bestScore = labelResult.score;
        bestField = 'label';
        bestMatchType = labelResult.matchType;
      }
    }

    // 2. Keywords (medium priority)
    if (fields.includes('keywords') && item.keywords.length > 0) {
      const kwResult = scoreKeywordMatch(item.keywords, queryLower);
      if (kwResult && kwResult.score > bestScore) {
        bestScore = kwResult.score;
        bestField = 'keywords';
        bestMatchType = kwResult.matchType;
      }
    }

    // 3. Description (lowest priority)
    if (fields.includes('description')) {
      const descResult = scoreDescriptionMatch(item.description, queryLower);
      if (descResult && descResult.score > bestScore) {
        bestScore = descResult.score;
        bestField = 'description';
        bestMatchType = descResult.matchType;
      }
    }

    if (bestScore >= threshold) {
      results.push({ item, score: bestScore, matchField: bestField, matchType: bestMatchType });
    }
  }

  // Sort by score descending
  results.sort((a, b) => b.score - a.score);

  return results;
}

// ───────────────────────────────────────────────────────────────────────
// Highlight utility — returns character-level match info
// ───────────────────────────────────────────────────────────────────────

export interface CharMatch {
  char: string;
  matched: boolean;
}

/**
 * Returns character-level match data for rendering highlighted text.
 * Works with the same fuzzy logic — highlights characters in order.
 */
export function getCharMatches(text: string, query: string): CharMatch[] {
  if (!query.trim()) return text.split('').map((char) => ({ char, matched: false }));

  const lower = text.toLowerCase();
  const q = query.toLowerCase();
  const result: CharMatch[] = [];
  let qi = 0;

  for (let i = 0; i < text.length; i++) {
    const isMatch = qi < q.length && lower[i] === q[qi];
    if (isMatch) qi++;
    result.push({ char: text[i], matched: isMatch });
  }

  return result;
}
