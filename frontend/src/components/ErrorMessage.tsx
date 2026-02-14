interface ErrorMessageProps {
    message: string;
    onDismiss?: () => void;
}

export default function ErrorMessage({ message, onDismiss }: ErrorMessageProps) {
    if (!message) return null;

    return (
        <div className="bg-red-900/50 border border-red-500 text-red-200 px-4 py-3 rounded relative mb-4">
            <span>{message}</span>
            {onDismiss && (
                <button
                    className="absolute top-0 right-0 px-4 py-3 text-red-200 hover:text-red-100"
                    onClick={onDismiss}
                >
                    &times;
                </button>
            )}
        </div>
    );
}
