// ═══════════════════════════════════════════════════════════════════════════
// Chaos Engineering Configuration — CCTV Health Monitor
// P1-QA.7: Chaos Engineering Testing
// Tool: toxiproxy-node / chaos patterns
// Compliance: IEC 62443-3-3 SR 7.8, ISO 27001 A.12.6
// ═══════════════════════════════════════════════════════════════════════════

const config = {
  services: {
    nats: { host: 'localhost', port: 4222, name: 'NATS JetStream' },
    postgres: { host: 'localhost', port: 5432, name: 'PostgreSQL' },
    redis: { host: 'localhost', port: 6379, name: 'Redis' },
    api: { host: 'localhost', port: 8080, name: 'API Gateway' },
  },

  toxiproxy: {
    apiHost: 'localhost',
    apiPort: 8474,
  },

  scenarios: [
    {
      id: 'nats-down',
      name: 'NATS Outage',
      description: 'Симуляция отказа NATS JetStream — проверка graceful degradation',
      service: 'nats',
      type: 'disconnect',
      duration: '30s',
      expectedRecovery: '< 5s',
      assertions: [
        'API health показывает degraded для NATS',
        'Кэшированные данные доступны',
        'Новые запросы не теряются (queue)',
        'Auto-recovery после восстановления NATS',
      ],
    },
    {
      id: 'nats-high-latency',
      name: 'NATS High Latency',
      description: 'Добавление 2s задержки на NATS — проверка timeout и circuit breaker',
      service: 'nats',
      type: 'latency',
      latencyMs: 2000,
      duration: '30s',
      expectedRecovery: '< 2s',
      assertions: [
        'Circuit breaker открывается при timeout',
        'Fallback на кэш работает',
        'После снижения задержки circuit breaker закрывается',
        'Метрики отображают задержки',
      ],
    },
    {
      id: 'postgres-down',
      name: 'PostgreSQL Outage',
      description: 'Симуляция отказа БД — проверка кэширования и fallback',
      service: 'postgres',
      type: 'disconnect',
      duration: '45s',
      expectedRecovery: '< 5s',
      assertions: [
        'Read-only режим активируется',
        'Кэшированные данные возвращаются из Redis',
        'Write операции ставятся в очередь',
        'После восстановления БД очередь дозаписывается',
        'Audit log фиксирует инцидент',
      ],
    },
    {
      id: 'postgres-high-latency',
      name: 'PostgreSQL Slow Queries',
      description: 'Симуляция медленных запросов к БД — timeout и retry',
      service: 'postgres',
      type: 'latency',
      latencyMs: 3000,
      duration: '20s',
      expectedRecovery: '< 1s',
      assertions: [
        'SQL запросы с timeout не блокируют API',
        'Connection pool не исчерпывается',
        'Retry logic срабатывает',
        'Метрики latency обновляются',
      ],
    },
    {
      id: 'redis-down',
      name: 'Redis Outage',
      description: 'Симуляция отказа Redis — проверка in-memory fallback',
      service: 'redis',
      type: 'disconnect',
      duration: '30s',
      expectedRecovery: '< 3s',
      assertions: [
        'Rate limiter переключается на in-memory',
        'Session management работает без Redis',
        'Кэш прозрачно деградирует',
        'После восстановления Redis кэш перезаполняется',
      ],
    },
    {
      id: 'api-high-load',
      name: 'API Gateway High Load',
      description: 'Симуляция высокой нагрузки — rate limiting и circuit breaker',
      service: 'api',
      type: 'latency',
      latencyMs: 1500,
      duration: '40s',
      expectedRecovery: '< 2s',
      assertions: [
        'Rate limiter отклоняет избыточные запросы (429)',
        'Circuit breaker защищает downstream сервисы',
        'Health endpoint остаётся доступным',
        'После нормализации нагрузки circuit breaker закрывается',
      ],
    },
    {
      id: 'packet-loss',
      name: 'Network Packet Loss (10%)',
      description: 'Симуляция 10% потери пакетов — проверка retry и resilience',
      service: 'nats',
      type: 'packet-loss',
      packetLossPercent: 10,
      duration: '30s',
      expectedRecovery: '< 5s',
      assertions: [
        'Пакеты с retry доставляются',
        'Нет потери данных',
        'Метрики отображают потерю пакетов',
      ],
    },
  ],

  healthChecks: {
    api: 'http://localhost:8080/api/v1/health',
    ready: 'http://localhost:8080/health/ready',
  },

  recovery: {
    maxRetries: 5,
    retryDelayMs: 1000,
    healthCheckTimeoutMs: 10000,
    maxRecoveryTimeMs: 10000,
  },
};

export default config;
