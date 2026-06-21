#!/usr/bin/env python3
"""Заменяет http.Error(w, ...) на respondError(w, r, ...) во всех .go файлах."""

import re, os, sys

API_DIR = "/home/viruz/cctv-monitoring/backend/internal/api"

STATUS_MAP = {
    "http.StatusBadRequest":          ("NewBadRequestError",      "BadRequestError"),
    "http.StatusUnauthorized":        ("NewUnauthorizedError",    "UnauthorizedError"),
    "http.StatusForbidden":           ("NewForbiddenError",       "ForbiddenError"),
    "http.StatusNotFound":            ("NewNotFoundError",        "NotFoundError"),
    "http.StatusConflict":            ("NewConflictError",        "ConflictError"),
    "http.StatusInternalServerError": ("NewInternalError",        "InternalError"),
    "http.StatusBadGateway":          ("NewExternalServiceError", "ExternalServiceError"),
    "http.StatusTooManyRequests":     ("NewRateLimitError",       "RateLimitError"),
    "http.StatusServiceUnavailable":  ("NewExternalServiceError", "ExternalServiceError"),
}

def extract_json_message(arg: str) -> str:
    """Extract message from `{"error":"msg"}` or '{"error":"msg"}'."""
    m = re.search(r'["\x60]\{"error":\s*"([^"]+)"', arg)
    if m:
        return m.group(1)
    return None

def replace_http_error_in_file(filepath: str) -> int:
    with open(filepath, 'r') as f:
        content = f.read()
    
    original = content
    count = 0
    
    # Pattern: http.Error(w, <message>, <status>)
    # We need to handle multi-line cases too
    # Strategy: find each http.Error( and parse manually
    
    lines = content.split('\n')
    new_lines = []
    i = 0
    in_http_error = False
    
    while i < len(lines):
        line = lines[i]
        stripped = line.strip()
        
        if 'http.Error(' in stripped:
            # Check if it's a complete statement (ends with ) on the same line)
            # or spans multiple lines
            match = re.match(r'^(\s*)http\.Error\((.*)\)\s*$', stripped)
            if match:
                # Single-line http.Error
                indent = match.group(1) if not line.startswith('\t') else line[:len(line) - len(line.lstrip())]
                args_str = match.group(2)
                new_line = transform_single_line_http_error(indent, args_str, line)
                if new_line is not None:
                    new_lines.append(new_line)
                    count += 1
                else:
                    new_lines.append(line)
                i += 1
                continue
            
            # Multi-line http.Error - collect lines until matching )
            indent = line[:len(line) - len(line.lstrip())]
            collected = [stripped]
            j = i + 1
            paren_depth = stripped.count('(') - stripped.count(')')
            while j < len(lines) and paren_depth > 0:
                collected.append(lines[j].strip())
                paren_depth += lines[j].count('(') - lines[j].count(')')
                j += 1
            
            joined = ' '.join(collected)
            new_block = transform_multi_line_http_error(indent, joined)
            if new_block is not None:
                new_lines.append(new_block)
                count += 1
            else:
                # Keep original lines
                for k in range(i, j):
                    new_lines.append(lines[k])
            i = j
            continue
        
        new_lines.append(line)
        i += 1
    
    if count > 0:
        new_content = '\n'.join(new_lines)
        with open(filepath, 'w') as f:
            f.write(new_content)
        print(f"  {filepath}: {count} replacements")
    
    return count


def transform_single_line_http_error(indent: str, args_str: str, original_line: str) -> str | None:
    """Transform a single-line http.Error call."""
    # Parse args: w, <message>, <status>
    # Skip the first arg (w) - it's always there
    # The message can be: "string", `backtick string`, err.Error(), variable
    
    # Find the last comma that separates message from status
    # We need to handle: funcCall(), "string with, comma", `backtick with,`
    
    # Try to find the status at the end
    status_match = re.search(r',\s*(http\.Status\w+)\s*$', args_str)
    if not status_match:
        return None
    
    status = status_match.group(1)
    # Everything between "w, " and the status is the message
    message_part = args_str[len("w, "):status_match.start()].strip()
    
    constructor, _ = STATUS_MAP.get(status, (None, None))
    if constructor is None:
        print(f"  WARNING: unknown status {status} in {original_line[:80]}")
        return None
    
    new_message = transform_message(message_part, status, constructor, indent)
    if new_message is None:
        return None
    
    return f'{indent}respondError(w, r, {new_message})'


def transform_multi_line_http_error(indent: str, joined: str) -> str | None:
    """Transform a multi-line http.Error call."""
    # Remove extra spaces
    joined = re.sub(r'\s+', ' ', joined)
    
    m = re.match(r'http\.Error\(\s*w,\s*(.+?),\s*(http\.Status\w+)\s*\)', joined)
    if not m:
        return None
    
    message_part = m.group(1).strip()
    status = m.group(2)
    
    constructor, _ = STATUS_MAP.get(status, (None, None))
    if constructor is None:
        return None
    
    new_message = transform_message(message_part, status, constructor, indent)
    if new_message is None:
        return None
    
    return f'{indent}respondError(w, r, {new_message})'


def transform_message(message_part: str, status: str, constructor: str, indent: str) -> str | None:
    """Transform the message part of http.Error to a respondError constructor call."""
    
    # Case 1: err.Error() → NewInternalError with wrapped error
    if 'err.Error()' in message_part:
        # Extract the variable name
        err_var = message_part.split('.Error()')[0].strip()
        return f'NewInternalError("operation failed", {err_var})'
    
    # Case 2: JSON backtick string like `{"error":"msg"}`
    json_msg = extract_json_message(message_part)
    if json_msg:
        return f'{constructor}("{json_msg}")'
    
    # Case 3: Plain string "message" or `message`
    quoted = message_part.strip()
    if (quoted.startswith('"') and quoted.endswith('"')) or \
       (quoted.startswith('`') and quoted.endswith('`')):
        # Use the string as-is
        inner = quoted[1:-1]
        # Escape double quotes in the message
        inner = inner.replace('"', '\\"')
        return f'{constructor}("{inner}")'
    
    # Case 4: Complex expression - keep as-is (already a function call, etc.)
    # This shouldn't happen for our codebase
    print(f"  WARNING: unhandled message pattern: {message_part[:60]}")
    return None


def main():
    total = 0
    for fname in sorted(os.listdir(API_DIR)):
        if not fname.endswith('.go'):
            continue
        fpath = os.path.join(API_DIR, fname)
        n = replace_http_error_in_file(fpath)
        total += n
    
    print(f"\nTotal replacements: {total}")

if __name__ == '__main__':
    main()