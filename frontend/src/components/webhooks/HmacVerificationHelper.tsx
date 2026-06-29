// ═══════════════════════════════════════════════════════════════════════
// HmacVerificationHelper — генератор кода для HMAC-верификации (P2-3.1)
//
// Features:
//   - Code snippets for JavaScript, Python, Go, curl
//   - Secret token masking (reveal on demand)
//   - Copy-to-clipboard for each snippet
//
// Compliance:
//   - OWASP ASVS V6 (Stored cryptography — secret field masking)
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { Shield, Copy, Check, Eye, EyeOff, Code } from '../ui/Icons';
import { Card, Button, Badge } from '../ui';

// ═══════════════════════════════════════════════════════════════════════
// Props
// ═══════════════════════════════════════════════════════════════════════

interface HmacVerificationHelperProps {
  secret: string;
}

// ═══════════════════════════════════════════════════════════════════════
// Language tabs
// ═══════════════════════════════════════════════════════════════════════

type Language = 'javascript' | 'python' | 'go' | 'curl';

const LANGUAGES: { key: Language; label: string }[] = [
  { key: 'javascript', label: 'JavaScript' },
  { key: 'python', label: 'Python' },
  { key: 'go', label: 'Go' },
  { key: 'curl', label: 'cURL' },
];

// ═══════════════════════════════════════════════════════════════════════
// Code generator
// ═══════════════════════════════════════════════════════════════════════

function generateSnippet(lang: Language, secret: string): string {
  const masked = secret || 'your-webhook-secret';

  switch (lang) {
    case 'javascript':
      return `// HMAC-SHA256 verification (Node.js)
const crypto = require('crypto');

function verifyWebhook(payload, signature, secret) {
  const hmac = crypto.createHmac('sha256', secret);
  const digest = hmac.update(
    typeof payload === 'string'
      ? payload
      : JSON.stringify(payload)
  ).digest('hex');
  return crypto.timingSafeEqual(
    Buffer.from(digest),
    Buffer.from(signature)
  );
}

// Usage:
// const isValid = verifyWebhook(req.body, req.headers['x-webhook-signature'], '${masked}');
// console.log('Signature valid:', isValid);`;

    case 'python':
      return `# HMAC-SHA256 verification (Python 3)
import hmac
import hashlib
import json

def verify_webhook(payload, signature, secret):
    """Verify webhook payload using HMAC-SHA256."""
    if isinstance(payload, dict):
        payload = json.dumps(payload, separators=(',', ':'))
    computed = hmac.new(
        secret.encode('utf-8'),
        payload.encode('utf-8'),
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(computed, signature)

# Usage:
# is_valid = verify_webhook(request.data, request.headers['X-Webhook-Signature'], '${masked}')
# print(f"Signature valid: {is_valid}")`;

    case 'go':
      return `// HMAC-SHA256 verification (Go)
package main

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
)

func VerifyWebhook(payload interface{}, signature string, secret string) bool {
    var data []byte
    switch p := payload.(type) {
    case string:
        data = []byte(p)
    default:
        data, _ = json.Marshal(p)
    }
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(data)
    expected := hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(expected), []byte(signature))
}

// Usage:
// secret := "${masked}"
// isValid := VerifyWebhook(payload, signature, secret)
// fmt.Printf("Signature valid: %v\\n", isValid)`;

    case 'curl':
      return `# Test webhook with HMAC signature (cURL)
# Note: Generate the signature server-side in production

curl -X POST https://your-webhook-endpoint.com \\
  -H "Content-Type: application/json" \\
  -H "X-Webhook-Signature: <computed_hmac_sha256>" \\
  -d '{
    "event": "device.offline",
    "timestamp": "${new Date().toISOString()}",
    "data": {
      "device_id": "cam-001",
      "device_name": "Test Camera"
    }
  }'

# To compute HMAC-SHA256 signature:
# echo -n '{"event":"device.offline","timestamp":"..."}' \\
#   | openssl dgst -sha256 -hmac "${masked}"`;

    default:
      return '';
  }
}

// ═══════════════════════════════════════════════════════════════════════
// Component
// ═══════════════════════════════════════════════════════════════════════

export function HmacVerificationHelper({ secret }: HmacVerificationHelperProps) {
  const { t } = useTranslation();
  const [selectedLang, setSelectedLang] = useState<Language>('javascript');
  const [showSecret, setShowSecret] = useState(false);
  const [copiedLang, setCopiedLang] = useState<Language | null>(null);

  const snippet = generateSnippet(selectedLang, secret);

  const copySnippet = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(snippet);
      setCopiedLang(selectedLang);
      setTimeout(() => setCopiedLang(null), 2000);
    } catch {
      // fallback silently
    }
  }, [snippet, selectedLang]);

  if (!secret) {
    return (
      <Card>
        <div className="p-4 space-y-3">
          <div className="flex items-center gap-2">
            <Shield className="w-4 h-4 text-slate-400" />
            <h3 className="text-sm font-bold text-slate-800 dark:text-slate-200">
              {t('hmac_verification') || 'HMAC Verification'}
            </h3>
          </div>
          <p className="text-xs text-slate-400">
            {t('hmac_no_secret_hint') ||
              'Set a secret token above to generate HMAC verification code snippets.'}
          </p>
        </div>
      </Card>
    );
  }

  return (
    <Card>
      <div className="p-4 space-y-4">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Shield className="w-4 h-4 text-emerald-500" />
            <h3 className="text-sm font-bold text-slate-800 dark:text-slate-200">
              {t('hmac_verification') || 'HMAC Verification'}
            </h3>
            <Badge variant="success" size="sm">
              {t('configured') || 'Configured'}
            </Badge>
          </div>
          <button
            type="button"
            onClick={() => setShowSecret(!showSecret)}
            className="flex items-center gap-1.5 text-[10px] px-2 py-1 rounded text-slate-500 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
          >
            {showSecret ? (
              <>
                <EyeOff className="w-3 h-3" />
                {t('hide_secret') || 'Hide'}
              </>
            ) : (
              <>
                <Eye className="w-3 h-3" />
                {t('show_secret') || 'Show'}
              </>
            )}
          </button>
        </div>

        {/* Secret Display */}
        <div className="px-3 py-2 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700">
          <p className="text-[10px] text-slate-400 mb-1">
            {t('your_secret') || 'Your Secret'}:
          </p>
          <code className="text-xs font-mono text-slate-700 dark:text-slate-300 break-all select-all">
            {showSecret ? secret : '•'.repeat(Math.min(40, Math.max(20, secret.length)))}
          </code>
        </div>

        {/* Language Tabs */}
        <div className="flex items-center gap-1 border-b border-slate-200 dark:border-slate-700">
          {LANGUAGES.map((lang) => (
            <button
              key={lang.key}
              type="button"
              onClick={() => setSelectedLang(lang.key)}
              className={`px-3 py-1.5 text-xs font-medium rounded-t transition-colors ${
                selectedLang === lang.key
                  ? 'text-blue-600 border-b-2 border-blue-600 bg-blue-50/50 dark:bg-blue-900/20 dark:text-blue-400 dark:border-blue-400'
                  : 'text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200'
              }`}
            >
              {lang.label}
            </button>
          ))}
        </div>

        {/* Code Snippet */}
        <div className="relative">
          <pre className="text-[11px] font-mono leading-relaxed bg-slate-900 text-slate-100 rounded-lg p-4 overflow-auto max-h-64 whitespace-pre">
            {snippet}
          </pre>
          <button
            type="button"
            onClick={copySnippet}
            className="absolute top-2 right-2 p-1.5 rounded bg-slate-800 hover:bg-slate-700 text-slate-400 hover:text-white transition-colors"
            title={t('copy_code') || 'Copy code'}
          >
            {copiedLang === selectedLang ? (
              <Check className="w-3.5 h-3.5 text-emerald-400" />
            ) : (
              <Copy className="w-3.5 h-3.5" />
            )}
          </button>
        </div>

        {/* Hint */}
        <p className="text-[10px] text-slate-400 leading-relaxed">
          {t('hmac_hint') ||
            'Use the code above to verify incoming webhook payloads on your end. The signature is sent in the X-Webhook-Signature header with each request.'}
        </p>
      </div>
    </Card>
  );
}
