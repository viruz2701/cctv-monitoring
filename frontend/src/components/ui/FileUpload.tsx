import React, { useRef, useState, useCallback } from 'react';
import { Upload, X, File, Image, FileText, Loader2 } from 'lucide-react';

interface FileUploadProps {
  onUpload: (files: File[]) => Promise<void>;
  accept?: string;
  multiple?: boolean;
  maxFiles?: number;
  maxSizeMB?: number;
  disabled?: boolean;
  className?: string;
  label?: string;
  hint?: string;
}

interface FilePreview {
  file: File;
  id: string;
  progress: number;
  error?: string;
}

export function FileUpload({
  onUpload,
  accept = '*',
  multiple = true,
  maxFiles = 10,
  maxSizeMB = 50,
  disabled = false,
  className = '',
  label = 'Перетащите файлы или нажмите для выбора',
  hint,
}: FileUploadProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [isDragOver, setIsDragOver] = useState(false);
  const [previews, setPreviews] = useState<FilePreview[]>([]);
  const [uploading, setUploading] = useState(false);

  const validateFile = (file: File): string | null => {
    if (file.size > maxSizeMB * 1024 * 1024) {
      return `Файл ${file.name} превышает ${maxSizeMB}MB`;
    }
    return null;
  };

  const addFiles = useCallback(
    (files: FileList | File[]) => {
      const fileArray = Array.from(files);
      const remaining = maxFiles - previews.length;
      if (remaining <= 0) return;

      const toAdd = fileArray.slice(0, remaining).map((file) => ({
        file,
        id: `${Date.now()}-${Math.random().toString(36).slice(2)}`,
        progress: 0,
        error: validateFile(file) || undefined,
      }));

      setPreviews((prev) => [...prev, ...toAdd]);
    },
    [previews.length, maxFiles],
  );

  const removeFile = (id: string) => {
    setPreviews((prev) => prev.filter((p) => p.id !== id));
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragOver(false);
    if (disabled) return;
    addFiles(e.dataTransfer.files);
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragOver(true);
  };

  const handleDragLeave = () => setIsDragOver(false);

  const handleClick = () => {
    if (!disabled) inputRef.current?.click();
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files) addFiles(e.target.files);
    e.target.value = '';
  };

  const handleUpload = async () => {
    const valid = previews.filter((p) => !p.error);
    if (valid.length === 0) return;

    setUploading(true);
    try {
      await onUpload(valid.map((p) => p.file));
      setPreviews([]);
    } catch {
      // Error handled by caller
    } finally {
      setUploading(false);
    }
  };

  const getFileIcon = (fileName: string) => {
    const ext = fileName.split('.').pop()?.toLowerCase();
    if (['jpg', 'jpeg', 'png', 'gif', 'svg', 'webp', 'bmp'].includes(ext || '')) {
      return <Image size={16} className="text-blue-500" />;
    }
    if (['pdf', 'doc', 'docx', 'txt', 'csv', 'xlsx'].includes(ext || '')) {
      return <FileText size={16} className="text-amber-500" />;
    }
    return <File size={16} className="text-slate-500" />;
  };

  const formatSize = (bytes: number): string => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  return (
    <div className={className}>
      <div
        onDrop={handleDrop}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onClick={handleClick}
        className={`border-2 border-dashed rounded-xl p-8 text-center cursor-pointer transition-colors ${
          isDragOver
            ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/10'
            : 'border-slate-300 dark:border-slate-600 hover:border-slate-400 dark:hover:border-slate-500'
        } ${disabled ? 'opacity-50 cursor-not-allowed' : ''}`}
      >
        <input
          ref={inputRef}
          type="file"
          className="hidden"
          accept={accept}
          multiple={multiple}
          onChange={handleChange}
          disabled={disabled}
        />
        <Upload className="mx-auto mb-3 text-slate-400" size={32} />
        <p className="text-sm text-slate-600 dark:text-slate-300 font-medium">{label}</p>
        {hint && (
          <p className="text-xs text-slate-400 dark:text-slate-500 mt-1">{hint}</p>
        )}
        <p className="text-xs text-slate-400 dark:text-slate-500 mt-2">
          Макс. {maxFiles} файлов, до {maxSizeMB}MB каждый
        </p>
      </div>

      {previews.length > 0 && (
        <div className="mt-4 space-y-2">
          {previews.map((preview) => (
            <div
              key={preview.id}
              className={`flex items-center gap-3 p-3 rounded-lg border ${
                preview.error
                  ? 'border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-900/10'
                  : 'border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800'
              }`}
            >
              {getFileIcon(preview.file.name)}
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-slate-700 dark:text-slate-300 truncate">
                  {preview.file.name}
                </p>
                <p className="text-xs text-slate-400 dark:text-slate-500">
                  {formatSize(preview.file.size)}
                  {preview.error && (
                    <span className="text-red-500 ml-2">{preview.error}</span>
                  )}
                </p>
              </div>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  removeFile(preview.id);
                }}
                className="p-1 text-slate-400 hover:text-red-500 rounded transition-colors"
              >
                <X size={16} />
              </button>
            </div>
          ))}

          <button
            onClick={handleUpload}
            disabled={uploading || previews.some((p) => p.error)}
            className="w-full py-2.5 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 text-white text-sm font-medium rounded-lg transition-colors flex items-center justify-center gap-2"
          >
            {uploading ? (
              <>
                <Loader2 size={16} className="animate-spin" />
                Загрузка...
              </>
            ) : (
              <>
                <Upload size={16} />
                Загрузить {previews.filter((p) => !p.error).length} файлов
              </>
            )}
          </button>
        </div>
      )}
    </div>
  );
}