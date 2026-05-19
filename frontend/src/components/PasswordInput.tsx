import { useState } from 'react';
import { Eye, EyeOff } from 'lucide-react';
import { ClipboardGetText } from '../../wailsjs/runtime/runtime';

interface PasswordInputProps {
    value: string;
    onChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
    placeholder?: string;
    id?: string;
    showPaste?: boolean;
    className?: string;
    onKeyDown?: (e: React.KeyboardEvent<HTMLInputElement>) => void;
    autoFocus?: boolean;
}

export default function PasswordInput({
    value,
    onChange,
    placeholder,
    id,
    showPaste = false,
    className = '',
    onKeyDown,
    autoFocus,
}: PasswordInputProps) {
    const [visible, setVisible] = useState(false);

    const handlePaste = async () => {
        try {
            const text = await ClipboardGetText();
            if (text) {
                onChange({ target: { value: text } } as React.ChangeEvent<HTMLInputElement>);
            }
        } catch {
            // clipboard unavailable
        }
    };

    // Reserve right padding for buttons: eye toggle (~2rem) + optional paste (~3rem)
    const rightPadding = showPaste ? '5.5rem' : '2.5rem';

    return (
        <div className="relative">
            <input
                type={visible ? 'text' : 'password'}
                value={value}
                onChange={onChange}
                placeholder={placeholder}
                id={id}
                onKeyDown={onKeyDown}
                autoFocus={autoFocus}
                className={className}
                style={{ paddingRight: rightPadding }}
            />
            <div className="absolute inset-y-0 right-0 flex items-center pr-1.5 gap-0.5">
                {showPaste && (
                    <button
                        type="button"
                        onClick={handlePaste}
                        tabIndex={-1}
                        style={{ color: 'var(--text-secondary)' }}
                        className="text-xs px-1.5 py-0.5 rounded hover:opacity-75 transition-opacity select-none"
                    >
                        Paste
                    </button>
                )}
                <button
                    type="button"
                    onClick={() => setVisible(v => !v)}
                    tabIndex={-1}
                    style={{ color: 'var(--text-secondary)' }}
                    className="flex items-center justify-center w-6 h-full rounded hover:opacity-75 transition-opacity"
                    aria-label={visible ? 'Hide password' : 'Show password'}
                >
                    {visible ? <EyeOff size={16} /> : <Eye size={16} />}
                </button>
            </div>
        </div>
    );
}
