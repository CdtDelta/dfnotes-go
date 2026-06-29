import { useState } from 'react';
import { VerifyChain } from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';
import ErrorMessage from './ErrorMessage';

interface ChainVerificationTabProps {
    caseId: string;
    onNavigateToBlock: (blockId: string, evidenceItemId: string | null) => void;
}

export default function ChainVerificationTab({ caseId, onNavigateToBlock }: ChainVerificationTabProps) {
    const [running, setRunning] = useState(false);
    const [result, setResult] = useState<main.ChainVerificationResult | null>(null);
    const [error, setError] = useState('');

    const run = async () => {
        setRunning(true);
        setError('');
        try {
            const res = await VerifyChain(caseId);
            setResult(res);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setRunning(false);
        }
    };

    return (
        <div>
            <div className="flex items-center gap-4 mb-4">
                <button
                    onClick={run}
                    disabled={running}
                    className="px-4 py-2 text-sm bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 disabled:text-gray-500 text-white rounded transition-colors"
                >
                    {running ? 'Verifying...' : 'Run verification'}
                </button>
                {result && !running && (
                    <span className="text-xs text-gray-500">Last run: {result.verified_at}</span>
                )}
            </div>

            <ErrorMessage message={error} onDismiss={() => setError('')} />

            {!result && !running && !error && (
                <p className="text-sm text-gray-400">
                    Verify the integrity of every committed block in this case.
                    This decrypts each block, recomputes its hash, checks its signature, and confirms
                    the chain links are unbroken.
                </p>
            )}

            {running && (
                <div className="flex items-center justify-center py-16">
                    <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-400"></div>
                </div>
            )}

            {result && !running && (
                <>
                    {result.total_blocks === 0 ? (
                        <div className="mb-4 px-4 py-3 rounded border border-gray-600 bg-gray-800 text-gray-300 text-sm">
                            No committed blocks to verify yet.
                        </div>
                    ) : result.chain_intact ? (
                        <div className="mb-4 px-4 py-3 rounded border border-green-700 bg-green-900/30 text-sm">
                            <span className="inline-flex items-center gap-1.5 font-medium text-green-400">
                                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                                </svg>
                                Chain intact. {result.total_blocks} block(s) verified at {result.verified_at}.
                            </span>
                        </div>
                    ) : (
                        <div className="mb-4 px-4 py-3 rounded border border-red-700 bg-red-900/30 text-sm">
                            <span className="inline-flex items-center gap-1.5 font-medium text-red-400">
                                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
                                </svg>
                                Chain integrity check failed. {result.failed_blocks} of {result.total_blocks} block(s) failed. First failure: block {result.first_failure_seq}.
                            </span>
                        </div>
                    )}

                    {result.total_blocks > 0 && (
                        <div className="overflow-x-auto">
                            <table className="w-full text-sm border-collapse">
                                <thead>
                                    <tr className="border-b border-gray-700 text-left text-xs text-gray-500 uppercase tracking-wide">
                                        <th className="pb-2 pr-4 font-medium">#</th>
                                        <th className="pb-2 pr-4 font-medium">Block ID</th>
                                        <th className="pb-2 pr-4 font-medium">Source</th>
                                        <th className="pb-2 pr-4 font-medium">Committed</th>
                                        <th className="pb-2 pr-4 font-medium">Hash</th>
                                        <th className="pb-2 pr-4 font-medium">Signature</th>
                                        <th className="pb-2 pr-4 font-medium">Link</th>
                                        <th className="pb-2 font-medium">Result</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {result.blocks.map((row) => {
                                        const failed = row.verdict !== 'verified';
                                        return (
                                            <tr
                                                key={row.block_id}
                                                onClick={() => onNavigateToBlock(
                                                    row.block_id,
                                                    row.evidence_item_id === '' ? null : row.evidence_item_id
                                                )}
                                                className={`border-b border-gray-800 cursor-pointer transition-colors hover:bg-gray-800/60 ${failed ? 'bg-red-900/10' : ''}`}
                                            >
                                                <td className="py-2 pr-4 text-gray-400">{row.sequence}</td>
                                                <td className="py-2 pr-4">
                                                    <span
                                                        className="font-mono text-xs text-gray-300"
                                                        title={row.block_id}
                                                    >
                                                        {row.block_id.length > 16
                                                            ? row.block_id.slice(0, 16) + '...'
                                                            : row.block_id}
                                                    </span>
                                                </td>
                                                <td className="py-2 pr-4 text-gray-300">
                                                    {row.source}{row.is_amendment ? ' (amendment)' : ''}
                                                </td>
                                                <td className="py-2 pr-4 text-gray-400 text-xs whitespace-nowrap">
                                                    {row.committed_at}
                                                </td>
                                                <td className="py-2 pr-4">
                                                    {!row.hash_checked ? (
                                                        <span className="text-gray-500">n/a</span>
                                                    ) : row.hash_valid ? (
                                                        <span className="text-green-400">pass</span>
                                                    ) : (
                                                        <span className="text-red-400 font-medium">fail</span>
                                                    )}
                                                </td>
                                                <td className="py-2 pr-4">
                                                    {row.signature_valid ? (
                                                        <span className="text-green-400">pass</span>
                                                    ) : (
                                                        <span className="text-red-400 font-medium">fail</span>
                                                    )}
                                                </td>
                                                <td className="py-2 pr-4">
                                                    {row.is_genesis ? (
                                                        <span className="text-gray-500">n/a (genesis)</span>
                                                    ) : row.link_valid ? (
                                                        <span className="text-green-400">linked</span>
                                                    ) : (
                                                        <span className="text-red-400 font-medium">broken</span>
                                                    )}
                                                </td>
                                                <td className="py-2">
                                                    {row.verdict === 'verified' ? (
                                                        <span className="text-green-400">Verified</span>
                                                    ) : row.verdict === 'tampered' ? (
                                                        <span className="text-red-400 font-medium">TAMPERED - {row.detail}</span>
                                                    ) : (
                                                        <span className="text-red-400 font-medium">CHAIN BREAK - {row.detail}</span>
                                                    )}
                                                </td>
                                            </tr>
                                        );
                                    })}
                                </tbody>
                            </table>
                        </div>
                    )}
                </>
            )}
        </div>
    );
}
