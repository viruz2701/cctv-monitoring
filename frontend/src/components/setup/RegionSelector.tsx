import { useState } from 'react';

export interface RegionInfo {
  region: string;
  name: string;
  description: string;
  compliance: string[];
  crypto_info: {
    encryption: string;
    hash: string;
    signature: string;
    key_size: number;
  };
  legal_notice: string;
}

interface RegionSelectorProps {
  regions: RegionInfo[];
  selected: string;
  onSelect: (region: string) => void;
  loading?: boolean;
}

export function RegionSelector({ regions, selected, onSelect, loading }: RegionSelectorProps) {
  const [expandedRegion, setExpandedRegion] = useState<string | null>(null);

  return (
    <div className="space-y-4" role="radiogroup" aria-label="Region selection">
      <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
        Select Deployment Region
      </h2>
      <p className="text-sm text-gray-500 dark:text-gray-400 mb-4">
        Choose the region for compliance requirements. This selection cannot be changed after setup.
      </p>

      {regions.map((region) => (
        <div
          key={region.region}
          className={`relative border rounded-lg p-4 cursor-pointer transition-all duration-200
            ${selected === region.region
              ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20 shadow-sm'
              : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600'
            }`}
          onClick={() => onSelect(region.region)}
          role="radio"
          aria-checked={selected === region.region}
          tabIndex={0}
          onKeyDown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') {
              e.preventDefault();
              onSelect(region.region);
            }
          }}
        >
          <div className="flex items-start justify-between">
            <div className="flex-1">
              <h3 className="text-md font-medium text-gray-900 dark:text-gray-100">
                {region.name}
              </h3>
              <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                {region.description}
              </p>
            </div>
            <div className="ml-4 flex-shrink-0">
              <div className={`w-5 h-5 rounded-full border-2 flex items-center justify-center
                ${selected === region.region
                  ? 'border-blue-500'
                  : 'border-gray-300 dark:border-gray-600'
                }`}
              >
                {selected === region.region && (
                  <div className="w-3 h-3 rounded-full bg-blue-500" />
                )}
              </div>
            </div>
          </div>

          {/* Compliance standards */}
          <div className="mt-3 flex flex-wrap gap-1.5">
            {region.compliance.map((std) => (
              <span
                key={std}
                className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium
                  bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300"
              >
                {std}
              </span>
            ))}
          </div>

          {/* Crypto details toggle */}
          <button
            type="button"
            className="mt-2 text-xs text-blue-600 dark:text-blue-400 hover:underline focus:outline-none"
            onClick={(e) => {
              e.stopPropagation();
              setExpandedRegion(expandedRegion === region.region ? null : region.region);
            }}
            aria-expanded={expandedRegion === region.region}
          >
            {expandedRegion === region.region ? 'Hide crypto details' : 'Show crypto details'}
          </button>

          {expandedRegion === region.region && (
            <div className="mt-2 p-3 bg-gray-50 dark:bg-gray-800/50 rounded text-xs space-y-1">
              <p><strong>Encryption:</strong> {region.crypto_info.encryption}</p>
              <p><strong>Hash:</strong> {region.crypto_info.hash}</p>
              <p><strong>Signature:</strong> {region.crypto_info.signature}</p>
              <p><strong>Key size:</strong> {region.crypto_info.key_size} bits</p>
            </div>
          )}

          {/* Legal notice */}
          <div className="mt-2 p-2 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded text-xs text-yellow-800 dark:text-yellow-200">
            <span role="img" aria-label="warning">⚠️</span> {region.legal_notice}
          </div>
        </div>
      ))}

      {loading && (
        <div className="text-center py-4 text-sm text-gray-500" role="status">
          <span className="animate-pulse">Loading regions...</span>
        </div>
      )}
    </div>
  );
}
