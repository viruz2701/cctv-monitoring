import React from 'react';
import { Search } from 'lucide-react';

interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
    label?: string;
    error?: string;
    helperText?: string;
}

export function Input({
    label,
    error,
    helperText,
    className = '',
    id,
    ...props
}: InputProps) {
    const inputId = id || label?.toLowerCase().replace(/\s+/g, '-');

    return (
        <div className="w-full">
            {label && (
                <label
                    htmlFor={inputId}
                    className="block text-sm font-medium text-slate-700 dark:text-slate-200 mb-1.5"
                >
                    {label}
                </label>
            )}
            <input
                id={inputId}
                className={`
          w-full px-3.5 py-2.5 text-sm text-slate-900 dark:text-white
          bg-white dark:bg-slate-900 border rounded-lg
          placeholder:text-slate-400
          focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500
          disabled:bg-slate-50 disabled:text-slate-500 disabled:cursor-not-allowed
          ${error ? 'border-red-300 focus:ring-red-500 focus:border-red-500' : 'border-slate-300 dark:border-slate-700'}
          ${className}
        `}
                {...props}
                onClick={(e) => {
                    if (props.type === 'date' || props.type === 'time' || props.type === 'datetime-local') {
                        try {
                            e.currentTarget.showPicker();
                        } catch (err) {
                            // ignore
                        }
                    }
                    props.onClick?.(e);
                }}
            />
            {error && <p className="mt-1.5 text-sm text-red-600">{error}</p>}
            {helperText && !error && (
                <p className="mt-1.5 text-sm text-slate-500">{helperText}</p>
            )}
        </div>
    );
}

interface SearchInputProps extends Omit<InputProps, 'type'> {
    onSearch?: (value: string) => void;
}

export function SearchInput({
    placeholder = 'Search...',
    className = '',
    onSearch,
    ...props
}: SearchInputProps) {
    return (
        <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
            <input
                type="text"
                placeholder={placeholder}
                className={`
          w-full pl-10 pr-4 py-2.5 text-sm text-slate-900 dark:text-white
          bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-700 rounded-lg
          placeholder:text-slate-400
          focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500
          ${className}
        `}
                onChange={(e) => onSearch?.(e.target.value)}
                {...props}
            />
        </div>
    );
}

interface SelectProps extends React.SelectHTMLAttributes<HTMLSelectElement> {
    label?: string;
    error?: string;
    options: { value: string; label: string }[];
}

export function Select({
    label,
    error,
    options,
    className = '',
    id,
    ...props
}: SelectProps) {
    const selectId = id || label?.toLowerCase().replace(/\\s+/g, '-');
    const hasCustomWidth = className.includes('w-');

    return (
        <div className={hasCustomWidth ? '' : 'w-full'}>
            {label && (
                <label
                    htmlFor={selectId}
                    className="block text-sm font-medium text-slate-700 dark:text-slate-200 mb-1.5"
                >
                    {label}
                </label>
            )}
            <select
                id={selectId}
                className={`
          px-3.5 py-2.5 text-sm text-slate-900 dark:text-white
          bg-white dark:bg-slate-900 border rounded-lg appearance-none
          focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500
          disabled:bg-slate-50 disabled:text-slate-500 disabled:cursor-not-allowed
          ${error ? 'border-red-300' : 'border-slate-300 dark:border-slate-700'}
          ${hasCustomWidth ? '' : 'w-full'}
          ${className}
        `}
                {...props}
            >
                {options.map((opt) => (
                    <option key={opt.value} value={opt.value}>
                        {opt.label}
                    </option>
                ))}
            </select>
            {error && <p className="mt-1.5 text-sm text-red-600">{error}</p>}
        </div>
    );
}

interface TextareaProps extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {
    label?: string;
    error?: string;
}

export function Textarea({
    label,
    error,
    className = '',
    id,
    ...props
}: TextareaProps) {
    const textareaId = id || label?.toLowerCase().replace(/\s+/g, '-');

    return (
        <div className="w-full">
            {label && (
                <label
                    htmlFor={textareaId}
                    className="block text-sm font-medium text-slate-700 dark:text-slate-200 mb-1.5"
                >
                    {label}
                </label>
            )}
            <textarea
                id={textareaId}
                className={`
          w-full px-3.5 py-2.5 text-sm text-slate-900 dark:text-white
          bg-white dark:bg-slate-900 border rounded-lg
          placeholder:text-slate-400
          focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500
          disabled:bg-slate-50 disabled:text-slate-500 disabled:cursor-not-allowed
          resize-y min-h-[100px]
          ${error ? 'border-red-300' : 'border-slate-300 dark:border-slate-700'}
          ${className}
        `}
                {...props}
            />
            {error && <p className="mt-1.5 text-sm text-red-600">{error}</p>}
        </div>
    );
}
