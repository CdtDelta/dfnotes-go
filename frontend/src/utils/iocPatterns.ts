import type { IOCType } from './iocTypes';

export interface IOCPattern {
    type: IOCType;
    /** RegExp must have the global flag set. */
    regex: RegExp;
    /** Human-readable label for the context menu. */
    label: string;
}

// Order matters: more-specific patterns appear first.
// URL before domain; SHA256 before SHA1 before MD5.
// These patterns are used client-side for highlighting only.
// The authoritative detection and storage happens in Go at commit time.
export const IOC_PATTERNS: IOCPattern[] = [
    {
        type: 'cve',
        regex: /\bCVE-\d{4}-\d{4,7}\b/g,
        label: 'CVE',
    },
    {
        type: 'registry_key',
        regex: /\b(?:HKEY_LOCAL_MACHINE|HKEY_CURRENT_USER|HKEY_CLASSES_ROOT|HKEY_USERS|HKEY_CURRENT_CONFIG|HKLM|HKCU|HKCR)\\[^\s"'<>]+/gi,
        label: 'Registry Key',
    },
    {
        type: 'url',
        regex: /\b(?:https?|ftp):\/\/[^\s<>"']+/gi,
        label: 'URL',
    },
    {
        type: 'email',
        regex: /\b[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,6}\b/g,
        label: 'Email',
    },
    {
        type: 'sha256',
        regex: /\b[0-9a-fA-F]{64}\b/g,
        label: 'SHA-256',
    },
    {
        type: 'sha1',
        regex: /\b[0-9a-fA-F]{40}\b/g,
        label: 'SHA-1',
    },
    {
        type: 'md5',
        regex: /\b[0-9a-fA-F]{32}\b/g,
        label: 'MD5',
    },
    {
        type: 'ipv4',
        regex: /\b(?:(?:25[0-5]|2[0-4]\d|1\d{2}|[1-9]\d|\d)\.){3}(?:25[0-5]|2[0-4]\d|1\d{2}|[1-9]\d|\d)(?!\.\d)\b/g,
        label: 'IPv4',
    },
    {
        type: 'ipv6',
        regex: /\b(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}\b|\b(?:[0-9a-fA-F]{1,4}:)*::(?:[0-9a-fA-F]{1,4}:)*[0-9a-fA-F]{1,4}\b|\b::(?:[0-9a-fA-F]{1,4}:)*[0-9a-fA-F]{1,4}\b/gi,
        label: 'IPv6',
    },
    {
        type: 'file_path',
        // Windows paths only; Unix paths are too noisy in rendered markdown.
        // Final segment excludes whitespace to avoid capturing trailing sentence text.
        regex: /\b[A-Za-z]:\\(?:[^\\/:*?"<>|\r\n]+\\)*[^\s\\/:*?"<>|\r\n]*/g,
        label: 'File Path',
    },
    {
        type: 'domain',
        // Requires at least one dot and a 2-6 char TLD. High FP rate -- backend filters.
        // Negative lookahead excludes known non-TLD file extensions.
        regex: /\b(?:[a-zA-Z0-9](?:[a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+(?!(dll|exe|sys|bat|ps1|sh|log|txt|json|xml|csv|zip|tar|gz)\b)[a-zA-Z]{2,6}\b/g,
        label: 'Domain',
    },
];
