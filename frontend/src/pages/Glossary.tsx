// ═══════════════════════════════════════════════════════════════════════
// Glossary — Справочник терминов CCTV Health Monitor
//
// P1-1.7: Contextual Tooltips
//   - Glossary page with search/filter
//   - Anchored entries (#term-id) from InfoTooltip links
//   - Category grouping
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useMemo, useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { useSearchParams } from 'react-router-dom';
import {
    BookOpen,
    Search,
    Info,
    ChevronDown,
    ChevronRight,
} from '../components/ui/Icons';
import { Input } from '../components/ui';

// ═══════════════════════════════════════════════════════════════════════
// Glossary Data
// ═══════════════════════════════════════════════════════════════════════

interface GlossaryEntry {
    id: string;
    term: string;
    definition: string;
    category: string;
    seeAlso?: string[];
}

const GLOSSARY_ENTRIES: GlossaryEntry[] = [
    // ── Device & Hardware ──────────────────────────────────────────
    {
        id: 'nvr',
        term: 'NVR (Network Video Recorder)',
        definition: 'Сетевой видеорегистратор — устройство для записи видео с IP-камер. Хранит видеопотоки, управляет записью по расписанию и по событиям.',
        category: 'device',
        seeAlso: ['dvr', 'camera'],
    },
    {
        id: 'dvr',
        term: 'DVR (Digital Video Recorder)',
        definition: 'Цифровой видеорегистратор — устройство для записи аналоговых видеосигналов в цифровом формате. Отличается от NVR типом подключаемых камер.',
        category: 'device',
        seeAlso: ['nvr'],
    },
    {
        id: 'mtbf',
        term: 'MTBF (Mean Time Between Failures)',
        definition: 'Среднее время наработки на отказ — показатель надёжности оборудования. Рассчитывается как отношение общего времени работы к числу отказов за период.',
        category: 'performance',
    },
    {
        id: 'mttr',
        term: 'MTTR (Mean Time To Repair)',
        definition: 'Среднее время восстановления — среднее время, необходимое для устранения неисправности и возврата устройства в рабочее состояние.',
        category: 'performance',
    },

    // ── Compliance & Security ──────────────────────────────────────
    {
        id: 'sla',
        term: 'SLA (Service Level Agreement)',
        definition: 'Соглашение об уровне обслуживания — договорённость между сервис-провайдером и заказчиком о допустимом времени реакции и устранения инцидентов.',
        category: 'compliance',
        seeAlso: ['sla-breach', 'kii'],
    },
    {
        id: 'sla-breach',
        term: 'SLA Breach (Нарушение SLA)',
        definition: 'Ситуация, когда время реакции или устранения инцидента превысило лимит, установленный в SLA. Требует немедленного вмешательства и формирования отчёта.',
        category: 'compliance',
        seeAlso: ['sla'],
    },
    {
        id: 'kii',
        term: 'КИИ (Критическая Информационная Инфраструктура)',
        definition: 'Объекты, информационные системы которых имеют критическое значение для национальной безопасности, экономики и общественной безопасности РБ. CCTV Health Monitor относится к классу KII-2.',
        category: 'compliance',
    },
    {
        id: 'asvs',
        term: 'OWASP ASVS Level 3',
        definition: 'Application Security Verification Standard — стандарт верификации безопасности приложений. Level 3 требует максимального уровня защиты, включая криптографию, контроль доступа и аудит.',
        category: 'compliance',
    },
    {
        id: 'iec-62443',
        term: 'IEC 62443',
        definition: 'Международный стандарт безопасности для систем промышленной автоматизации и управления (IACS). Определяет зоны безопасности (SL-1..SL-4) и требования к ним.',
        category: 'compliance',
        seeAlso: ['kii'],
    },

    // ── CCTV Operations ────────────────────────────────────────────
    {
        id: 'rca',
        term: 'RCA (Root Cause Analysis)',
        definition: 'Анализ первопричины — методика выявления исходной причины отказа или инцидента. Строит граф зависимостей устройств и определяет корневой элемент проблемы.',
        category: 'operations',
    },
    {
        id: 'blast-radius',
        term: 'Blast Radius (Радиус поражения)',
        definition: 'Метрика влияния отказа — количество устройств и сервисов, затронутых отказом одного элемента системы. Используется в RCA для оценки масштаба инцидента.',
        category: 'operations',
        seeAlso: ['rca'],
    },
    {
        id: 'health-score',
        term: 'Health Score (Индекс здоровья)',
        definition: 'Комплексная оценка состояния устройства на основе uptime, температуры, свободного места на диске, частоты ошибок и статуса записи.',
        category: 'operations',
    },

    // ── Work Orders & CMMS ─────────────────────────────────────────
    {
        id: 'work-order',
        term: 'Work Order (Наряд на работу)',
        definition: 'Электронный документ, содержащий задание на обслуживание или ремонт оборудования. Включает описание работ, приоритет, назначенного техника и SLA-таймер.',
        category: 'cmms',
    },
    {
        id: 'preventive-maintenance',
        term: 'Preventive Maintenance (Плановое ТО)',
        definition: 'Регламентное обслуживание оборудования по расписанию — замена расходников, проверка параметров, чистка. Цель — предотвращение отказов до их возникновения.',
        category: 'cmms',
        seeAlso: ['work-order'],
    },
    {
        id: 'corrective-maintenance',
        term: 'Corrective Maintenance (Внеплановое ТО)',
        definition: 'Внеплановое обслуживание по факту отказа или деградации оборудования. Инициируется автоматически при обнаружении аномалий или по заявке.',
        category: 'cmms',
        seeAlso: ['work-order', 'rca'],
    },

    // ── Network & Protocols ─────────────────────────────────────────
    {
        id: 'onvif',
        term: 'ONVIF (Open Network Video Interface Forum)',
        definition: 'Международный стандарт для IP-камер и VMS. Обеспечивает совместимость устройств разных производителей. Включает профили G (запись), S (базовый), T (аналитика).',
        category: 'network',
    },
    {
        id: 'rtsp',
        term: 'RTSP (Real-Time Streaming Protocol)',
        definition: 'Протокол управления потоками видео/аудио в реальном времени. Используется для запроса и передачи видеопотоков от камер к NVR и клиентам.',
        category: 'network',
        seeAlso: ['onvif'],
    },
    {
        id: 'poe',
        term: 'PoE (Power over Ethernet)',
        definition: 'Технология подачи электропитания через Ethernet-кабель. Стандарты: 802.3af (15.4W), 802.3at/PoE+ (30W), 802.3bt/PoE++ (60-100W).',
        category: 'network',
    },
    {
        id: 'vlan',
        term: 'VLAN (Virtual Local Area Network)',
        definition: 'Виртуальная локальная сеть — логическое разделение физической сети на изолированные сегменты. Используется для отделения CCTV-трафика от корпоративной сети.',
        category: 'network',
        seeAlso: ['qos'],
    },
    {
        id: 'qos',
        term: 'QoS (Quality of Service)',
        definition: 'Приоритизация сетевого трафика для гарантии пропускной способности для видеопотоков. Критически важна для CCTV с большим числом камер.',
        category: 'network',
    },
    {
        id: 'multicast',
        term: 'Multicast (Многоадресная рассылка)',
        definition: 'Передача одного видеопотока множеству получателей без дублирования пакетов. Используется для трансляции видео на несколько мониторов/VMS-клиентов.',
        category: 'network',
    },
    {
        id: 'nvr-storage',
        term: 'RAID (Redundant Array of Independent Disks)',
        definition: 'Технология объединения нескольких дисков в массив для отказоустойчивости (RAID 1/5/6/10) или повышения производительности (RAID 0). Критична для NVR.',
        category: 'network',
    },

    // ── Video & Codecs ──────────────────────────────────────────────
    {
        id: 'h264',
        term: 'H.264 / AVC (Advanced Video Coding)',
        definition: 'Стандарт сжатия видео с высокой эффективностью. Обеспечивает хорошее качество при умеренном битрейте. Самый распространённый кодек в CCTV.',
        category: 'video',
    },
    {
        id: 'h265',
        term: 'H.265 / HEVC (High Efficiency Video Coding)',
        definition: 'Стандарт сжатия видео следующего поколения. Обеспечивает снижение битрейта на 50% по сравнению с H.264 при том же качестве изображения.',
        category: 'video',
        seeAlso: ['h264'],
    },
    {
        id: 'fps',
        term: 'FPS (Frames Per Second)',
        definition: 'Количество кадров в секунду — показатель плавности видеопотока. Для CCTV: 15-30 FPS (наблюдение), 5-10 FPS (архив), 30+ FPS (LVL/BVR).',
        category: 'video',
    },
    {
        id: 'bitrate',
        term: 'Bitrate (Битрейт)',
        definition: 'Объём данных видеопотока в единицу времени (кбит/с, Мбит/с). Влияет на качество видео и требования к хранилищу. CBR (постоянный) vs VBR (переменный).',
        category: 'video',
        seeAlso: ['h264', 'h265'],
    },
    {
        id: 'resolution',
        term: 'Разрешение (HD/Full HD/4K/8K)',
        definition: 'Количество пикселей изображения: HD (1280×720), Full HD (1920×1080), 4K (3840×2160), 8K (7680×4320). Выше разрешение → больше деталей и объём хранилища.',
        category: 'video',
    },

    // ── Analytics & AI ──────────────────────────────────────────────
    {
        id: 'vca',
        term: 'VCA (Video Content Analytics)',
        definition: 'Анализ видеоконтента в реальном времени — обнаружение движения, распознавание лиц, номеров, подсчёт людей, детекция оставленных предметов.',
        category: 'analytics',
    },
    {
        id: 'motion-detection',
        term: 'Motion Detection (Детекция движения)',
        definition: 'Базовый метод обнаружения изменений в видеопотоке путём сравнения пикселей между кадрами. Используется для триггера записи по событию.',
        category: 'analytics',
        seeAlso: ['vca'],
    },
    {
        id: 'lpr',
        term: 'LPR / ANPR (License/Автоматическое распознавание номеров)',
        definition: 'Технология автоматического распознавания государственных регистрационных знаков. Используется на въездных группах и парковках.',
        category: 'analytics',
    },
    {
        id: 'fisheye',
        term: 'Fisheye / 360° камера',
        definition: 'Камера с широкоугольным объективом (180°-360°). Обеспечивает панорамный обзор. Требует dewarping (коррекции искажений) в VMS.',
        category: 'analytics',
    },

    // ── Security & Access Control ───────────────────────────────────
    {
        id: 'rbac',
        term: 'RBAC (Role-Based Access Control)',
        definition: 'Управление доступом на основе ролей — каждый пользователь имеет одну или несколько ролей (admin, technician, viewer), определяющих его права.',
        category: 'security',
    },
    {
        id: 'mfa',
        term: 'MFA / 2FA (Multi-Factor Authentication)',
        definition: 'Многофакторная аутентификация — подтверждение личности с использованием двух+ факторов (пароль + TOTP/SMS/WebAuthn). Требование КИИ РБ.',
        category: 'security',
        seeAlso: ['webauthn'],
    },
    {
        id: 'webauthn',
        term: 'WebAuthn / FIDO2',
        definition: 'Стандарт passwordless-аутентификации с использованием аппаратных токенов (YubiKey) или биометрии (Touch ID, Face ID). Поддержка в браузерах и мобильных приложениях.',
        category: 'security',
        seeAlso: ['mfa'],
    },
    {
        id: 'tls',
        term: 'TLS 1.3 (Transport Layer Security)',
        definition: 'Протокол шифрования данных при передаче. В CCTV Health Monitor используется mTLS 1.3 для всех соединений между зонами безопасности (IEC 62443).',
        category: 'security',
        seeAlso: ['iec-62443'],
    },
    {
        id: 'ldap',
        term: 'LDAP / Active Directory',
        definition: 'Протокол доступа к централизованному каталогу пользователей. Используется для интеграции с корпоративной аутентификацией (SSO).',
        category: 'security',
    },
    {
        id: 'oauth2',
        term: 'OAuth 2.0 / OpenID Connect',
        definition: 'Протокол авторизации и аутентификации. OAuth 2.0 для делегирования доступа, OpenID Connect (OIDC) для единой аутентификации (SSO).',
        category: 'security',
    },

    // ── CMMS & Maintenance ──────────────────────────────────────────
    {
        id: 'cmms-platform',
        term: 'CMMS (Computerized Maintenance Management System)',
        definition: 'Компьютеризированная система управления обслуживанием — управляет Work Orders, расписанием ТО, запчастями и персоналом. Адаптеры: Internal, Atlas.',
        category: 'cmms',
        seeAlso: ['work-order', 'preventive-maintenance'],
    },
    {
        id: 'eam',
        term: 'EAM (Enterprise Asset Management)',
        definition: 'Система управления активами предприятия — более широкая концепция, чем CMMS. Включает управление жизненным циклом активов, финансами и compliance.',
        category: 'cmms',
        seeAlso: ['asset-lifecycle'],
    },
    {
        id: 'asset-lifecycle',
        term: 'Asset Lifecycle (Жизненный цикл актива)',
        definition: 'Стадии жизни устройства: планирование → закупка → установка → эксплуатация → обслуживание → вывод из эксплуатации. Каждая стадия отслеживается в CMMS.',
        category: 'cmms',
    },
    {
        id: 'rcm',
        term: 'RCM (Reliability-Centered Maintenance)',
        definition: 'Методология определения оптимальной стратегии обслуживания на основе критичности и режимов отказов оборудования. Нацелена на максимизацию надёжности.',
        category: 'cmms',
    },
    {
        id: 'fmea',
        term: 'FMEA (Failure Mode and Effects Analysis)',
        definition: 'Анализ видов и последствий отказов — систематический метод выявления потенциальных отказов оборудования и оценки их влияния на систему.',
        category: 'cmms',
        seeAlso: ['rcm'],
    },
    {
        id: 'fifo',
        term: 'FIFO (First In, First Out)',
        definition: 'Метод управления запасами и запчастями — первая поступившая на склад единица используется первой. Предотвращает устаревание расходных материалов.',
        category: 'cmms',
    },
    {
        id: 'calibration',
        term: 'Калибровка и сертификация',
        definition: 'Периодическая поверка измерительного оборудования и датчиков в соответствии с требованиями ISO 9001 и отраслевыми стандартами. Отслеживается в CMMS.',
        category: 'cmms',
    },
    {
        id: 'depreciation',
        term: 'Амортизация (Depreciation)',
        definition: 'Постепенное снижение стоимости актива в процессе эксплуатации. Методы: линейный, уменьшаемого остатка, производственный. Важно для финансового учёта.',
        category: 'cmms',
    },

    // ── Monitoring & Metrics ────────────────────────────────────────
    {
        id: 'uptime',
        term: 'Uptime / Availability (Доступность)',
        definition: 'Процент времени, в течение которого устройство/система была работоспособна. Цель CCTV Health Monitor: 99.99% (Four Nines) для критических систем.',
        category: 'monitoring',
        seeAlso: ['sla'],
    },
    {
        id: 'snmp',
        term: 'SNMP (Simple Network Management Protocol)',
        definition: 'Протокол управления сетевыми устройствами. Используется для мониторинга состояния коммутаторов, NVR и других сетевых устройств (OID, MIB).',
        category: 'monitoring',
    },
    {
        id: 'syslog',
        term: 'Syslog',
        definition: 'Стандартный протокол сбора системных логов. Используется для централизованного аудита событий безопасности и диагностики оборудования.',
        category: 'monitoring',
    },
    {
        id: 'snmp-trap',
        term: 'SNMP Trap',
        definition: 'Асинхронное уведомление от SNMP-устройства о событии (отказ, превышение порога). Используется для мгновенного оповещения о критических событиях.',
        category: 'monitoring',
        seeAlso: ['snmp'],
    },

    // ── Compliance & Regulatory ─────────────────────────────────────
    {
        id: 'nis2',
        term: 'NIS2 Directive (EU 2022/2555)',
        definition: 'Директива ЕС о безопасности сетевых и информационных систем. Расширяет требования к КИИ, включая CCTV-системы в критической инфраструктуре.',
        category: 'compliance',
        seeAlso: ['gdpr', 'kii'],
    },
    {
        id: 'gdpr',
        term: 'GDPR (General Data Protection Regulation)',
        definition: 'Регламент ЕС о защите персональных данных. Требует DPIA для CCTV-систем, уведомления об обработке видеоизображений, право на удаление данных.',
        category: 'compliance',
        seeAlso: ['dpia'],
    },
    {
        id: 'dpia',
        term: 'DPIA (Data Protection Impact Assessment)',
        definition: 'Оценка воздействия на защиту данных — обязательная процедура для CCTV-систем. Оценивает риски для субъектов ПД и меры их минимизации.',
        category: 'compliance',
        seeAlso: ['gdpr'],
    },
    {
        id: 'hmac',
        term: 'HMAC (Hash-based Message Authentication Code)',
        definition: 'Код аутентификации сообщений на основе хеш-функции. В CCTV Health Monitor используется bash-256 HMAC для подписи audit_log (ISO 27001 A.12.4).',
        category: 'compliance',
        seeAlso: ['asvs', 'iec-62443'],
    },
    {
        id: 'oac-66',
        term: 'Приказ ОАЦ № 66',
        definition: 'Требования ОАЦ РБ к защите конечных узлов и сетей (п. 7.18). Включает уникальную идентификацию устройств, mTLS 1.3, контроль целостности и tamper detection.',
        category: 'compliance',
        seeAlso: ['kii'],
    },
    {
        id: 'stb-crypto',
        term: 'СТБ 34.101.30 (Криптография РБ)',
        definition: 'Стандарт криптографической защиты Республики Беларусь. Определяет алгоритмы: belt (шифрование), bign (ЭЦП), bash (хеширование). Заменяет AES/RSA/SHA.',
        category: 'compliance',
        seeAlso: ['iec-62443'],
    },
    {
        id: 'pki',
        term: 'PKI (Public Key Infrastructure)',
        definition: 'Инфраструктура открытых ключей — система управления сертификатами и ключами. Используется для mTLS-аутентификации устройств и подписи JWT.',
        category: 'compliance',
        seeAlso: ['tls'],
    },

    // ── SLA & Performance ───────────────────────────────────────────
    {
        id: 'slo',
        term: 'SLO (Service Level Objective)',
        definition: 'Целевой уровень обслуживания — конкретная метрика в рамках SLA (например, "MTTR ≤ 4 часа для P1"). Измеряется и отслеживается в реальном времени.',
        category: 'performance',
        seeAlso: ['sla'],
    },
    {
        id: 'sli',
        term: 'SLI (Service Level Indicator)',
        definition: 'Фактическое измерение соблюдения SLO. Например, процент инцидентов P1, решённых в пределах 4 часов за отчётный период.',
        category: 'performance',
        seeAlso: ['slo', 'sla'],
    },
    {
        id: 'oee',
        term: 'OEE (Overall Equipment Effectiveness)',
        definition: 'Общая эффективность оборудования — метрика, учитывающая доступность, производительность и качество. Используется для оценки эффективности ТО.',
        category: 'performance',
    },
    {
        id: 'fcr',
        term: 'FCR (First Call Resolution)',
        definition: 'Процент инцидентов, решённых при первом обращении. Высокий FCR (>70%) указывает на эффективную диагностику и квалификацию персонала.',
        category: 'performance',
    },
    {
        id: 'csat',
        term: 'CSAT (Customer Satisfaction Score)',
        definition: 'Оценка удовлетворённости заказчика после выполнения Work Order. Измеряется по шкале 1-5. Цель: CSAT ≥ 4.5.',
        category: 'performance',
    },
];

const CATEGORIES = [
    { key: 'device', label: 'Device & Hardware' },
    { key: 'network', label: 'Network & Protocols' },
    { key: 'video', label: 'Video & Codecs' },
    { key: 'analytics', label: 'Analytics & AI' },
    { key: 'performance', label: 'Performance & Reliability' },
    { key: 'compliance', label: 'Compliance & Security' },
    { key: 'security', label: 'Security & Access Control' },
    { key: 'operations', label: 'CCTV Operations' },
    { key: 'cmms', label: 'Work Orders & CMMS' },
    { key: 'monitoring', label: 'Monitoring & Metrics' },
];

// ═══════════════════════════════════════════════════════════════════════
// Component
// ═══════════════════════════════════════════════════════════════════════

export function Glossary() {
    const { t } = useTranslation();
    const [searchParams] = useSearchParams();
    const [search, setSearch] = useState('');
    const [expandedEntries, setExpandedEntries] = useState<Set<string>>(new Set());
    const entriesRef = useRef<Map<string, HTMLDivElement>>(new Map());

    // Handle URL hash for deep-linking from InfoTooltip
    useEffect(() => {
        const hash = window.location.hash.replace('#', '');
        if (hash) {
            setExpandedEntries(prev => new Set(prev).add(hash));
            setTimeout(() => {
                const el = entriesRef.current.get(hash);
                if (el) {
                    el.scrollIntoView({ behavior: 'smooth', block: 'center' });
                }
            }, 100);
        }
    }, []);

    const filtered = useMemo(() => {
        if (!search.trim()) return GLOSSARY_ENTRIES;
        const q = search.toLowerCase();
        return GLOSSARY_ENTRIES.filter(
            e => e.term.toLowerCase().includes(q)
                || e.definition.toLowerCase().includes(q)
                || e.category.toLowerCase().includes(q)
        );
    }, [search]);

    const grouped = useMemo(() => {
        const groups = new Map<string, GlossaryEntry[]>();
        for (const entry of filtered) {
            const existing = groups.get(entry.category) ?? [];
            existing.push(entry);
            groups.set(entry.category, existing);
        }
        return groups;
    }, [filtered]);

    const toggleEntry = (id: string) => {
        setExpandedEntries(prev => {
            const next = new Set(prev);
            if (next.has(id)) next.delete(id);
            else next.add(id);
            return next;
        });
    };

    return (
        <div className="p-4 md:p-6 max-w-4xl mx-auto space-y-6">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900 dark:text-white flex items-center gap-2">
                        <BookOpen className="w-6 h-6" />
                        {t('glossary') || 'Glossary'}
                    </h1>
                    <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
                        {t('glossary_description') || 'Reference of technical terms used in CCTV Health Monitor'}
                    </p>
                </div>
            </div>

            {/* Search */}
            <div className="relative max-w-md">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
                <input
                    type="text"
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    placeholder={t('search_glossary') || 'Search terms...'}
                    className="w-full pl-10 pr-4 py-2.5 text-sm border border-slate-200 dark:border-slate-700 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    aria-label={t('search_glossary') || 'Search glossary terms'}
                />
            </div>

            {/* Results count */}
            <p className="text-xs text-slate-400">
                {filtered.length} {t('terms') || 'terms'}
                {search && ` matching "${search}"`}
            </p>

            {/* Glossary Entries by Category */}
            {Array.from(grouped.entries()).map(([category, entries]) => {
                const cat = CATEGORIES.find(c => c.key === category);
                return (
                    <div key={category} className="space-y-2">
                        <h2 className="text-sm font-semibold text-slate-600 dark:text-slate-400 uppercase tracking-wider px-1">
                            {cat?.label || category}
                        </h2>
                        <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 divide-y divide-slate-100 dark:divide-slate-700/50 overflow-hidden">
                            {entries.map((entry) => (
                                <div
                                    key={entry.id}
                                    ref={(el) => { if (el) entriesRef.current.set(entry.id, el); }}
                                    id={entry.id}
                                    className="scroll-mt-20"
                                >
                                    <button
                                        onClick={() => toggleEntry(entry.id)}
                                        className="w-full flex items-start gap-3 px-4 py-3.5 text-left hover:bg-slate-50 dark:hover:bg-slate-700/50 transition-colors group"
                                        aria-expanded={expandedEntries.has(entry.id)}
                                    >
                                        <div className="flex-1 min-w-0">
                                            <div className="flex items-center gap-2">
                                                {expandedEntries.has(entry.id) ? (
                                                    <ChevronDown className="w-4 h-4 text-slate-400 shrink-0 transition-transform" />
                                                ) : (
                                                    <ChevronRight className="w-4 h-4 text-slate-400 shrink-0 transition-transform" />
                                                )}
                                                <span className="text-sm font-medium text-slate-900 dark:text-white group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">
                                                    {entry.term}
                                                </span>
                                            </div>
                                        </div>
                                        <Info className="w-4 h-4 text-slate-300 dark:text-slate-600 shrink-0 mt-0.5" />
                                    </button>

                                    {expandedEntries.has(entry.id) && (
                                        <div className="px-4 pb-4 pt-1 pl-12">
                                            <p className="text-sm text-slate-600 dark:text-slate-300 leading-relaxed">
                                                {entry.definition}
                                            </p>
                                            {entry.seeAlso && entry.seeAlso.length > 0 && (
                                                <div className="flex items-center gap-2 mt-2 flex-wrap">
                                                    <span className="text-xs text-slate-400">{t('see_also') || 'See also:'}</span>
                                                    {entry.seeAlso.map((ref) => {
                                                        const refEntry = GLOSSARY_ENTRIES.find(e => e.id === ref);
                                                        return refEntry ? (
                                                            <button
                                                                key={ref}
                                                                onClick={() => {
                                                                    toggleEntry(ref);
                                                                    const el = entriesRef.current.get(ref);
                                                                    if (el) el.scrollIntoView({ behavior: 'smooth', block: 'center' });
                                                                }}
                                                                className="text-xs font-medium text-blue-600 dark:text-blue-400 hover:underline"
                                                            >
                                                                {refEntry.term}
                                                            </button>
                                                        ) : null;
                                                    })}
                                                </div>
                                            )}
                                        </div>
                                    )}
                                </div>
                            ))}
                        </div>
                    </div>
                );
            })}

            {/* Empty state */}
            {filtered.length === 0 && (
                <div className="text-center py-12">
                    <BookOpen className="w-12 h-12 text-slate-300 dark:text-slate-600 mx-auto mb-3" />
                    <p className="text-sm font-medium text-slate-500 dark:text-slate-400">
                        {t('no_terms_found') || 'No terms found'}
                    </p>
                    <p className="text-xs text-slate-400 mt-1">
                        {t('try_different_search') || 'Try a different search term'}
                    </p>
                </div>
            )}
        </div>
    );
}