// ──────────────────────────────────────────────────
// Глобальные типы Detox для E2E тестов
//
// Detox внедряет эти глобалы runtime-ом через
// detox/runners/jest/testEnvironment.
// TS не знает о них — предоставляем декларации.
// ──────────────────────────────────────────────────

/// <reference types="detox" />

declare const device: Detox.Device;
declare const element: Detox.Element;
declare const by: Detox.By;
declare const waitFor: Detox.WaitFor;
declare const expect: Detox.Expect<Detox.Expect>;

declare namespace NodeJS {
  interface Global {
    device: Detox.Device;
    element: Detox.Element;
    by: Detox.By;
    waitFor: Detox.WaitFor;
    expect: Detox.Expect<Detox.Expect>;
  }
}
