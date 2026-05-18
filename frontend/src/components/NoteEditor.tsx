import { useState, useRef, useCallback, useEffect } from 'react';
import TurndownService from 'turndown';
import { SaveAttachment, AttachImage } from '../../wailsjs/go/main/App';
import { services } from '../../wailsjs/go/models';
import MarkdownRenderer from './MarkdownRenderer';

interface NoteEditorProps {
    caseId: string;
    content: string;
    onChange: (content: string) => void;
    evidenceItems?: services.EvidenceResponse[];
    onEvidenceClick?: (evidenceItemId: string) => void;
}

const turndown = new TurndownService({ headingStyle: 'atx', codeBlockStyle: 'fenced' });

interface AutocompleteItem {
    label: string;
    insert: string;
}

export default function NoteEditor({ caseId, content, onChange, evidenceItems, onEvidenceClick }: NoteEditorProps) {
    const [previewMode, setPreviewMode] = useState(false);
    const [autocomplete, setAutocomplete] = useState<AutocompleteItem[] | null>(null);
    const [acIndex, setAcIndex] = useState(0);
    const [acPos, setAcPos] = useState<{ top: number; left: number } | null>(null);
    const textareaRef = useRef<HTMLTextAreaElement>(null);
    const acTriggerPos = useRef<number | null>(null);

    const sortedEvidence = (evidenceItems || []).slice().sort((a, b) =>
        a.created_at.localeCompare(b.created_at)
    );

    const checkAutocomplete = useCallback(() => {
        const textarea = textareaRef.current;
        if (!textarea || !sortedEvidence.length) {
            setAutocomplete(null);
            return;
        }
        const cursorPos = textarea.selectionStart;
        const textBefore = textarea.value.substring(0, cursorPos);
        const match = textBefore.match(/\[\[([^\]]*)$/);
        if (!match) {
            setAutocomplete(null);
            acTriggerPos.current = null;
            return;
        }
        acTriggerPos.current = cursorPos - match[1].length - 2; // position of [[
        const query = match[1].toLowerCase();
        const items: AutocompleteItem[] = sortedEvidence
            .map((e, i) => {
                const label = `E${String(i + 1).padStart(3, '0')}`;
                return { label: `${label}: ${e.name}`, insert: `${label}` };
            })
            .filter((item) => item.label.toLowerCase().includes(query));

        if (items.length > 0) {
            setAutocomplete(items);
            setAcIndex(0);
            // Rough positioning: use textarea scroll and cursor position
            const linesBefore = textBefore.split('\n');
            const lineNum = linesBefore.length - 1;
            const charInLine = linesBefore[linesBefore.length - 1].length;
            setAcPos({
                top: (lineNum + 1) * 20 + 8 - textarea.scrollTop,
                left: Math.min(charInLine * 7.5, textarea.clientWidth - 200),
            });
        } else {
            setAutocomplete(null);
        }
    }, [sortedEvidence]);

    const insertAutocomplete = useCallback((item: AutocompleteItem) => {
        const textarea = textareaRef.current;
        if (!textarea || acTriggerPos.current === null) return;
        const before = textarea.value.substring(0, acTriggerPos.current);
        const cursorPos = textarea.selectionStart;
        const after = textarea.value.substring(cursorPos);
        // Remove the partial [[ and insert [[item]]
        const insertion = `[[${item.insert}]]`;
        const newValue = before + insertion + after;
        onChange(newValue);
        setAutocomplete(null);
        acTriggerPos.current = null;
        // Set cursor after insertion
        setTimeout(() => {
            const newPos = before.length + insertion.length;
            textarea.selectionStart = newPos;
            textarea.selectionEnd = newPos;
            textarea.focus();
        }, 0);
    }, [onChange]);

    const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
        if (!autocomplete) return;
        if (e.key === 'ArrowDown') {
            e.preventDefault();
            setAcIndex((i) => Math.min(i + 1, autocomplete.length - 1));
        } else if (e.key === 'ArrowUp') {
            e.preventDefault();
            setAcIndex((i) => Math.max(i - 1, 0));
        } else if (e.key === 'Enter' || e.key === 'Tab') {
            e.preventDefault();
            insertAutocomplete(autocomplete[acIndex]);
        } else if (e.key === 'Escape') {
            setAutocomplete(null);
        }
    };

    const saveImageAndInsert = (file: File) => {
        const reader = new FileReader();
        reader.onload = async () => {
            const base64 = (reader.result as string).split(',')[1];
            if (!base64) return;
            try {
                const resp = await SaveAttachment({
                    case_id: caseId,
                    filename: `pasted-image-${Date.now()}.${file.type.split('/')[1] || 'png'}`,
                    content_type: file.type || 'image/png',
                    data: base64,
                } as services.SaveAttachmentRequest);
                insertAtCursor(`![pasted image](attachment:${resp.attachment_id})`);
            } catch { /* ignore */ }
        };
        reader.readAsDataURL(file);
    };

    const saveBase64ImageAndInsert = (base64: string, contentType: string) => {
        SaveAttachment({
            case_id: caseId,
            filename: `pasted-image-${Date.now()}.${contentType.split('/')[1] || 'png'}`,
            content_type: contentType,
            data: base64,
        } as services.SaveAttachmentRequest)
            .then((resp) => {
                insertAtCursor(`![pasted image](attachment:${resp.attachment_id})`);
            })
            .catch(() => {});
    };

    const handlePaste = (e: React.ClipboardEvent<HTMLTextAreaElement>) => {
        const clipboard = e.clipboardData;

        // 1. Try clipboard.items for image files (use index-based loop for WebView compat)
        for (let i = 0; i < clipboard.items.length; i++) {
            const item = clipboard.items[i];
            if (item.kind === 'file') {
                const file = item.getAsFile();
                if (file && file.type.startsWith('image/')) {
                    e.preventDefault();
                    saveImageAndInsert(file);
                    return;
                }
            }
        }

        // 2. Try clipboard.files (some WebViews populate this instead of items)
        for (let i = 0; i < clipboard.files.length; i++) {
            const file = clipboard.files[i];
            if (file.type.startsWith('image/')) {
                e.preventDefault();
                saveImageAndInsert(file);
                return;
            }
        }

        // 3. Check HTML for embedded images (WebKit often sends screenshots as <img src="data:...">)
        const html = clipboard.getData('text/html');
        if (html) {
            const imgMatch = html.match(/<img[^>]+src="data:(image\/[^;]+);base64,([^"]+)"/);
            if (imgMatch) {
                e.preventDefault();
                saveBase64ImageAndInsert(imgMatch[2], imgMatch[1]);
                return;
            }

            const plain = clipboard.getData('text/plain');
            // Convert formatted HTML to markdown
            if (html.includes('<p') || html.includes('<h') || html.includes('<li') ||
                html.includes('<b') || html.includes('<strong') || html.includes('<em') ||
                html.includes('<table') || html.includes('<code') || html.includes('<a ')) {
                e.preventDefault();
                const md = turndown.turndown(html);
                insertAtCursor(md || plain);
                return;
            }
        }
        // Otherwise: plain text pastes normally (default behavior)
    };

    const insertAtCursor = (text: string) => {
        const textarea = textareaRef.current;
        if (!textarea) return;
        const start = textarea.selectionStart;
        const end = textarea.selectionEnd;
        // Read current value from DOM (not closure) to avoid stale state
        const currentValue = textarea.value;
        const before = currentValue.substring(0, start);
        const after = currentValue.substring(end);
        onChange(before + text + after);
        setTimeout(() => {
            const newPos = start + text.length;
            textarea.selectionStart = newPos;
            textarea.selectionEnd = newPos;
            textarea.focus();
        }, 0);
    };

    const handleAttachImage = async () => {
        try {
            const resp = await AttachImage(caseId);
            if (!resp) return; // user cancelled
            insertAtCursor(`![${resp.filename}](attachment:${resp.attachment_id})`);
        } catch { /* ignore */ }
    };

    // Close autocomplete when clicking outside
    useEffect(() => {
        if (!autocomplete) return;
        const handleClick = () => setAutocomplete(null);
        document.addEventListener('click', handleClick);
        return () => document.removeEventListener('click', handleClick);
    }, [autocomplete]);

    return (
        <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
            <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-3">
                    <h3 className="text-sm font-medium text-gray-300">New Note</h3>
                    {!previewMode && (
                        <button
                            onClick={handleAttachImage}
                            className="text-xs text-gray-400 hover:text-gray-200 transition-colors inline-flex items-center gap-1"
                            title="Attach image from file"
                        >
                            <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 15.75l5.159-5.159a2.25 2.25 0 013.182 0l5.159 5.159m-1.5-1.5l1.409-1.409a2.25 2.25 0 013.182 0l2.909 2.909M3.75 21h16.5A2.25 2.25 0 0022.5 18.75V5.25A2.25 2.25 0 0020.25 3H3.75A2.25 2.25 0 001.5 5.25v13.5A2.25 2.25 0 003.75 21z" />
                            </svg>
                            Attach Image
                        </button>
                    )}
                </div>
                <div className="flex items-center gap-2">
                    <button
                        onClick={() => setPreviewMode(false)}
                        className={`px-3 py-1 text-xs rounded transition-colors ${
                            !previewMode ? 'bg-gray-600 text-gray-100' : 'text-gray-400 hover:text-gray-200'
                        }`}
                    >
                        Edit
                    </button>
                    <button
                        onClick={() => setPreviewMode(true)}
                        className={`px-3 py-1 text-xs rounded transition-colors ${
                            previewMode ? 'bg-gray-600 text-gray-100' : 'text-gray-400 hover:text-gray-200'
                        }`}
                    >
                        Preview
                    </button>
                </div>
            </div>

            {previewMode ? (
                <div className="prose prose-sm max-w-none min-h-[160px] p-3 bg-gray-900 rounded border border-gray-700 break-words overflow-hidden">
                    {content.trim() ? (
                        <MarkdownRenderer
                            content={content}
                            caseId={caseId}
                            evidenceItems={evidenceItems}
                            onEvidenceClick={onEvidenceClick}
                        />
                    ) : (
                        <p className="text-gray-500 italic">Nothing to preview</p>
                    )}
                </div>
            ) : (
                <div className="relative">
                    <textarea
                        ref={textareaRef}
                        value={content}
                        onChange={(e) => { onChange(e.target.value); checkAutocomplete(); }}
                        onKeyUp={checkAutocomplete}
                        onKeyDown={handleKeyDown}
                        onPaste={handlePaste}
                        placeholder="Write your note in markdown... Type [[ to link evidence items."
                        className="w-full h-40 px-3 py-2 bg-gray-900 border border-gray-700 rounded text-gray-100 font-mono text-sm resize-y focus:outline-none focus:border-blue-500 placeholder-gray-600 break-all"
                    />
                    {autocomplete && acPos && (
                        <div
                            className="absolute z-20 bg-gray-800 border border-gray-600 rounded shadow-xl max-h-40 overflow-y-auto"
                            style={{ top: acPos.top, left: acPos.left, minWidth: '200px' }}
                        >
                            {autocomplete.map((item, i) => (
                                <button
                                    key={i}
                                    onMouseDown={(e) => { e.preventDefault(); insertAutocomplete(item); }}
                                    className={`w-full text-left px-3 py-1.5 text-xs transition-colors ${
                                        i === acIndex ? 'bg-blue-600 text-white' : 'text-gray-200 hover:bg-gray-700'
                                    }`}
                                >
                                    {item.label}
                                </button>
                            ))}
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}
