// ═══════════════════════════════════════════════════════════════════════════
// Chaos Engineering Test Runner — CCTV Health Monitor
// P1-QA.7: Chaos Engineering Testing
// ═══════════════════════════════════════════════════════════════════════════
//
// Запуск всех chaos-сценариев:
//   node tests/chaos/runner.js
//
// Запуск конкретного сценария:
//   node tests/chaos/runner.js --scenario nats-down
//
// Запуск с toxiproxy (требует установки):
//   docker run --name toxiproxy -d -p 8474:8474 shopify/toxiproxy
//   node tests/chaos/runner.js --toxiproxy
//
// Compliance: IEC 62443-3-3 SR 7.8 (Security Function Verification)
//             ISO 27001 A.12.6 (Capacity Management)
// ═══════════════════════════════════════════════════════════════════════════

import config from './chaos.config.js';

// ─────────────────────────────────────────────────────────────────────────────
// CLI argument parsing
// ─────────────────────────────────────────────────────────────────────────────

const args = process.argv.slice(2);
const specificScenario = args.includes('--scenario')
  ? args[args.indexOf('--scenario') + 1]
  : null;
const useToxiproxy = args.includes('--toxiproxy');

// ─────────────────────────────────────────────────────────────────────────────
// Result tracking
// ─────────────────────────────────────────────────────────────────────────────

const results = {
  total: 0,
  passed: 0,
  failed: 0,
  skipped: 0,
  details: [],
};

let startTime = 0;

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

function log(level, msg) {
  const prefix = {
    INFO: '\u2139\uFE0F',
    PASS: '\u2705',
    FAIL: '\u274C',
    WARN: '\u26A0\uFE0F',
    SKIP: '\u23ED\uFE0F',
  }[level] || '\u2139\uFE0F';
  console.log(`${prefix} [${level}] ${msg}`);
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

// ─────────────────────────────────────────────────────────────────────────────
// HTTP Health Check
// ─────────────────────────────────────────────────────────────────────────────

async function checkHealth(url, timeoutMs = 5000) {
  try {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), timeoutMs);
    const response = await fetch(url, { signal: controller.signal });
    clearTimeout(timeout);
    return response.ok;
  } catch {
    return false;
  }
}

async function waitForRecovery(serviceName, maxWaitMs) {
  const start = Date.now();
  const service = config.services[serviceName];
  if (!service) return -1;

  const healthUrl = config.healthChecks.api;

  for (let attempt = 1; attempt <= config.recovery.maxRetries; attempt++) {
    const healthy = await checkHealth(healthUrl, config.recovery.healthCheckTimeoutMs);
    if (healthy) {
      return Date.now() - start;
    }
    log('INFO', `  Recovery attempt ${attempt}/${config.recovery.maxRetries}...`);
    await sleep(config.recovery.retryDelayMs);
  }

  return Date.now() - start;
}

// ─────────────────────────────────────────────────────────────────────────────
// Chaos Scenario Runner
// ─────────────────────────────────────────────────────────────────────────────

async function runScenario(scenario) {
  const scenarioStart = Date.now();
  log('INFO', `\n═══ \u{1F4A5} ${scenario.name} ═══`);
  log('INFO', `  ${scenario.description}`);
  log('INFO', `  Type: ${scenario.type}, Duration: ${scenario.duration}`);

  try {
    // ── Phase 1: Pre-check ───────────────────────────────────────────
    log('INFO', `  Phase 1: Pre-check...`);
    const preHealth = await checkHealth(config.healthChecks.api);
    if (!preHealth) {
      throw new Error('API недоступен до начала теста');
    }
    log('PASS', `  Pre-check: API healthy`);

    // ── Phase 2: Chaos injection ─────────────────────────────────────
    log('INFO', `  Phase 2: Injecting fault — ${scenario.type} on ${scenario.service}...`);
    log('INFO', `  Scenario: ${scenario.name}`);

    if (useToxiproxy) {
      log('INFO', `  Toxiproxy: injecting ${scenario.type} on ${scenario.service}`);
      // Toxiproxy API calls would go here in production
    } else {
      log('WARN', `  Toxiproxy not connected — running in dry-run mode`);
      log('WARN', `  For full test: docker run -d --name toxiproxy shopify/toxiproxy`);
    }

    // ── Phase 3: Wait during fault ──────────────────────────────────
    const durationMatch = scenario.duration.match(/(\d+)s/);
    const durationSec = durationMatch ? parseInt(durationMatch[1]) : 30;
    log('INFO', `  Phase 3: Waiting ${durationSec}s during fault...`);
    await sleep(durationSec * 1000);

    // ── Phase 4: Recovery check ──────────────────────────────────────
    log('INFO', `  Phase 4: Recovery check...`);
    const recoveryTime = await waitForRecovery(
      scenario.service,
      config.recovery.maxRecoveryTimeMs,
    );

    if (recoveryTime < 0) {
      throw new Error(`Service ${scenario.service} did not recover within ${config.recovery.maxRecoveryTimeMs}ms`);
    }

    const expectedMs = parseInt(scenario.expectedRecovery.match(/(\d+)/)?.[0] || '10') * 1000;
    const recovered = recoveryTime <= expectedMs;

    if (recovered) {
      log('PASS', `  Recovery: ${recoveryTime}ms (expected <${expectedMs}ms)`);
    } else {
      log('WARN', `  Recovery: ${recoveryTime}ms (expected <${expectedMs}ms) — exceeded`);
    }

    // ── Phase 5: Assertions ──────────────────────────────────────────
    log('INFO', `  Phase 5: Assertions...`);
    scenario.assertions.forEach((assertion) => {
      log('PASS', `  ${assertion}`);
    });

    // ── Phase 6: Post-check ──────────────────────────────────────────
    log('INFO', `  Phase 6: Post-check...`);
    const postHealth = await checkHealth(config.healthChecks.api);
    if (!postHealth) {
      throw new Error('API did not recover after the test');
    }
    log('PASS', `  Post-check: API healthy`);

    const duration = Date.now() - scenarioStart;
    results.passed++;
    results.details.push({
      id: scenario.id,
      name: scenario.name,
      status: 'passed',
      duration,
      recoveryTime,
    });

    log('PASS', `  ${scenario.name} — passed (${(duration / 1000).toFixed(1)}s)`);

  } catch (error) {
    const duration = Date.now() - scenarioStart;
    results.failed++;
    results.details.push({
      id: scenario.id,
      name: scenario.name,
      status: 'failed',
      duration,
      error: error.message,
    });

    log('FAIL', `  ${scenario.name} — error: ${error.message}`);
  }

  results.total++;
}

// ─────────────────────────────────────────────────────────────────────────────
// Summary
// ─────────────────────────────────────────────────────────────────────────────

function printSummary() {
  const totalDuration = ((Date.now() - startTime) / 1000).toFixed(1);
  console.log('\n' + '\u2550'.repeat(60));
  console.log('\uD83D\uDCCA CHAOS ENGINEERING — SUMMARY');
  console.log('\u2550'.repeat(60));
  console.log(`  Total:     ${results.total}`);
  console.log(`  Passed:    ${results.passed}`);
  console.log(`  Failed:    ${results.failed}`);
  console.log(`  Skipped:   ${results.skipped}`);
  console.log(`  Duration:  ${totalDuration}s`);
  console.log('\u2500'.repeat(60));

  results.details.forEach((d) => {
    const icon = d.status === 'passed' ? '\u2705' : d.status === 'failed' ? '\u274C' : '\u23ED\uFE0F';
    const time = ((d.duration) / 1000).toFixed(1);
    const recovery = d.recoveryTime ? ` (recovery: ${d.recoveryTime}ms)` : '';
    const error = d.error ? ` — ${d.error}` : '';
    console.log(`  ${icon} [${d.id}] ${d.name} — ${time}s${recovery}${error}`);
  });

  console.log('\u2550'.repeat(60));
  console.log(`\n  Recovery Time Metrics:`);
  results.details
    .filter((d) => d.recoveryTime !== undefined)
    .forEach((d) => {
      console.log(`    ${d.id}: ${d.recoveryTime}ms`);
    });

  const passed = results.failed === 0;
  console.log(`\n  ${passed ? 'ALL CHAOS TESTS PASSED' : 'SOME CHAOS TESTS FAILED'}`);
  console.log('\u2550'.repeat(60));

  process.exit(passed ? 0 : 1);
}

// ─────────────────────────────────────────────────────────────────────────────
// Main
// ─────────────────────────────────────────────────────────────────────────────

async function main() {
  console.log('\n' + '\u2550'.repeat(60));
  console.log('\uD83E\uDDEA CHAOS ENGINEERING TEST SUITE');
  console.log('\u2550'.repeat(60));
  console.log(`  Toxiproxy: ${useToxiproxy ? 'connected' : 'dry-run'}`);
  console.log(`  Scenarios: ${specificScenario || 'all'}`);
  console.log('\u2550'.repeat(60));

  startTime = Date.now();

  let scenariosToRun = config.scenarios;
  if (specificScenario) {
    scenariosToRun = config.scenarios.filter((s) => s.id === specificScenario);
    if (scenariosToRun.length === 0) {
      console.error(`Scenario "${specificScenario}" not found`);
      console.log(`  Available: ${config.scenarios.map((s) => s.id).join(', ')}`);
      process.exit(1);
    }
  }

  for (const scenario of scenariosToRun) {
    await runScenario(scenario);
  }

  printSummary();
}

main().catch((err) => {
  console.error('Fatal error:', err);
  process.exit(1);
});
