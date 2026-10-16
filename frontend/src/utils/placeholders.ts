// Mirrors internal/config/placeholders.go. A placeholder is {{NAME}} with no
// braces inside; the name is the inner text trimmed and is case-sensitive.
const PLACEHOLDER_SOURCE = '\\{\\{([^{}]*)\\}\\}';

function placeholderRegex(): RegExp {
    // Fresh instance per call: a shared /g regex carries lastIndex state.
    return new RegExp(PLACEHOLDER_SOURCE, 'g');
}

// Single-pass replacement so a value that looks like a placeholder is
// inserted literally. Names without a non-blank value are left as-is, which
// is what the apply preview wants for unfilled boxes.
export function substitutePlaceholders(s: string, values: Record<string, string>): string {
    return s.replace(placeholderRegex(), (match: string, inner: string) => {
        const v = values[inner.trim()];
        return v !== undefined && v.trim() !== '' ? v.trim() : match;
    });
}

// True if s contains {{}} or a placeholder whose name is only whitespace.
export function hasEmptyPlaceholder(s: string): boolean {
    for (const m of s.matchAll(placeholderRegex())) {
        if (m[1].trim() === '') return true;
    }
    return false;
}
