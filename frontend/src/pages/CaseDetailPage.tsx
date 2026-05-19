import { useEffect, useRef, useState, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { GetCase, UnlockCase, LockCase, ListEvidence } from '../../wailsjs/go/main/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';
import { services } from '../../wailsjs/go/models';
import ClassificationBadge from '../components/ClassificationBadge';
import CaseOverviewTab from '../components/CaseOverviewTab';
import EvidenceTab from '../components/EvidenceTab';
import EvidenceNotesTab from '../components/EvidenceNotesTab';
import MasterNotesTab from '../components/MasterNotesTab';
import IOCSummaryTab from '../components/IOCSummaryTab';
import TimelineTab from '../components/TimelineTab';
import TaskListTab from '../components/TaskListTab';
import ErrorMessage from '../components/ErrorMessage';
import ExportDialog from '../components/ExportDialog';

type PageState = 'loading' | 'locked' | 'unlocked';

export default function CaseDetailPage() {
    const { caseId } = useParams<{ caseId: string }>();
    const navigate = useNavigate();

    const [pageState, setPageState] = useState<PageState>('loading');
    const [caseData, setCaseData] = useState<services.CaseResponse | null>(null);
    const [activeTab, setActiveTab] = useState<string>('overview');
    const [password, setPassword] = useState('');
    const [error, setError] = useState('');
    const [unlocking, setUnlocking] = useState(false);
    const [evidenceItems, setEvidenceItems] = useState<services.EvidenceResponse[]>([]);
    const [showExportDialog, setShowExportDialog] = useState(false);

    // Always-current ref so the unmount cleanup can read the latest pageState
    const pageStateRef = useRef<PageState>('loading');
    useEffect(() => { pageStateRef.current = pageState; });

    // Auto-lock when navigating away while the case is unlocked so caseKeys
    // are never left populated after the user leaves the case page.
    useEffect(() => {
        return () => {
            if (pageStateRef.current === 'unlocked' && caseId) {
                LockCase(caseId).catch(() => {});
            }
        };
    }, [caseId]);

    const fetchEvidenceItems = useCallback(() => {
        if (!caseId) return;
        ListEvidence(caseId)
            .then((result) => setEvidenceItems(result || []))
            .catch(() => {});
    }, [caseId]);

    useEffect(() => {
        if (!caseId) return;
        GetCase(caseId)
            .then((result) => {
                setCaseData(result);
                setPageState('locked');
            })
            .catch((err: unknown) => {
                setError(err instanceof Error ? err.message : String(err));
                setPageState('locked');
            });
    }, [caseId]);

    const handleUnlock = async () => {
        if (!caseId || !password) return;
        setUnlocking(true);
        setError('');
        try {
            await UnlockCase({ case_id: caseId, case_password: password } as services.UnlockCaseRequest);
            setPassword('');
            setPageState('unlocked');
            fetchEvidenceItems();
        } catch (err: unknown) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setUnlocking(false);
        }
    };

    const handleLock = async () => {
        if (!caseId) return;
        try {
            await LockCase(caseId);
            setPageState('locked');
            setActiveTab('overview');
        } catch (err: unknown) {
            setError(err instanceof Error ? err.message : String(err));
        }
    };

    const handlePasswordKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter') {
            handleUnlock();
        }
    };

    const handleEvidenceClick = useCallback((evidenceItemId: string) => {
        setActiveTab(`evidence-notes-${evidenceItemId}`);
    }, []);

    const handleNavigate = useCallback((tab: string, blockId?: string) => {
        setActiveTab(tab);
        if (blockId) {
            setTimeout(() => {
                const el = document.getElementById(blockId);
                if (!el) return;
                el.scrollIntoView({ behavior: 'smooth', block: 'start' });
                el.classList.add('block-target');
                setTimeout(() => el.classList.remove('block-target'), 1500);
            }, 80);
        }
    }, []);

    // Listen for menu Export Case event -- only triggers when case is unlocked
    useEffect(() => {
        const cleanup = EventsOn('menu:export-case', () => {
            if (pageState === 'unlocked' && caseId) {
                setShowExportDialog(true);
            }
        });
        return cleanup;
    }, [pageState, caseId]);

    // Listen for menu Lock Case event
    useEffect(() => {
        const cleanup = EventsOn('menu:lock-case', () => {
            if (pageState === 'unlocked' && caseId) {
                LockCase(caseId)
                    .then(() => {
                        setPageState('locked');
                        setActiveTab('overview');
                    })
                    .catch((err: unknown) => {
                        setError(err instanceof Error ? err.message : String(err));
                    });
            }
        });
        return cleanup;
    }, [pageState, caseId]);

    if (pageState === 'loading') {
        return (
            <div className="flex items-center justify-center min-h-screen">
                <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-400"></div>
            </div>
        );
    }

    const sortedEvidenceItems = [...evidenceItems].sort((a, b) =>
        a.created_at.localeCompare(b.created_at)
    );
    const evidenceTabs = sortedEvidenceItems.map((item, index) => ({
        id: `evidence-notes-${item.evidence_item_id}`,
        label: `E${String(index + 1).padStart(3, '0')}`,
    }));

    const tabs: { id: string; label: string }[] = [
        { id: 'overview', label: 'Case Overview' },
        { id: 'notes', label: 'Master Notes' },
        ...evidenceTabs,
        { id: 'iocs', label: 'IOC Summary' },
        { id: 'timeline', label: 'Timeline' },
        { id: 'tasks', label: 'Tasks' },
        { id: 'evidence', label: 'Evidence' },
    ];

    return (
        <>
        <div className="min-h-screen p-6">
            <div>
                {/* Header */}
                <div className="mb-6">
                    <button
                        onClick={() => navigate('/')}
                        className="text-sm text-gray-400 hover:text-gray-200 mb-4 inline-flex items-center gap-1 transition-colors"
                    >
                        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 19.5L8.25 12l7.5-7.5" />
                        </svg>
                        Back to Cases
                    </button>

                    {caseData && (
                        <div className="flex items-start justify-between">
                            <div>
                                <div className="flex items-center gap-3 mb-1">
                                    <h1 className="text-2xl font-bold text-gray-100">{caseData.title}</h1>
                                    <ClassificationBadge level={caseData.classification} />
                                </div>
                                <p className="text-sm text-gray-400 font-mono">{caseData.case_number}</p>
                            </div>
                            {pageState === 'unlocked' && (
                                <button
                                    onClick={handleLock}
                                    className="flex items-center gap-1.5 text-sm text-gray-400 hover:text-red-400 transition-colors"
                                >
                                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                                        <path strokeLinecap="round" strokeLinejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z" />
                                    </svg>
                                    Lock Case
                                </button>
                            )}
                        </div>
                    )}
                </div>

                <ErrorMessage message={error} onDismiss={() => setError('')} />

                {/* Locked State */}
                {pageState === 'locked' && (
                    <div className="flex justify-center py-16">
                        <div className="w-full max-w-sm">
                            <div className="text-center mb-6">
                                <svg className="w-12 h-12 mx-auto text-gray-500 mb-3" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z" />
                                </svg>
                                <h2 className="text-lg font-semibold text-gray-200">Case Locked</h2>
                                <p className="text-sm text-gray-400 mt-1">Enter the case password to unlock</p>
                            </div>
                            <div className="space-y-4">
                                <input
                                    type="password"
                                    value={password}
                                    onChange={(e) => setPassword(e.target.value)}
                                    onKeyDown={handlePasswordKeyDown}
                                    placeholder="Case password"
                                    className="w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded text-gray-100 focus:outline-none focus:border-blue-500 placeholder-gray-600"
                                    autoFocus
                                />
                                <button
                                    onClick={handleUnlock}
                                    disabled={unlocking || !password}
                                    className="w-full bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 disabled:text-gray-500 text-white py-2 px-4 rounded transition-colors"
                                >
                                    {unlocking ? 'Unlocking...' : 'Unlock Case'}
                                </button>
                            </div>
                        </div>
                    </div>
                )}

                {/* Unlocked State */}
                {pageState === 'unlocked' && caseData && (
                    <>
                        {/* Tab Bar */}
                        <nav className="flex flex-wrap gap-2 mb-6">
                            {tabs.map((tab) => (
                                <button
                                    key={tab.id}
                                    onClick={() => {
                                        setActiveTab(tab.id);
                                        if (tab.id === 'evidence') fetchEvidenceItems();
                                    }}
                                    className={`px-4 py-2 text-sm font-medium rounded border transition-colors ${
                                        activeTab === tab.id
                                            ? 'border-[var(--border-accent-bright)] bg-[var(--bg-accent)] text-gray-100'
                                            : 'border-gray-700 bg-gray-800 text-gray-400 hover:text-gray-100 hover:border-[var(--border-accent-bright)]'
                                    }`}
                                >
                                    {tab.label}
                                </button>
                            ))}
                        </nav>

                        {/* Tab Content */}
                        {activeTab === 'overview' && <CaseOverviewTab caseData={caseData} />}
                        {activeTab === 'evidence' && <EvidenceTab caseId={caseData.case_id} onEvidenceChanged={fetchEvidenceItems} />}
                        {activeTab.startsWith('evidence-notes-') && (
                            <EvidenceNotesTab
                                caseId={caseData.case_id}
                                evidenceItemId={activeTab.replace('evidence-notes-', '')}
                                evidenceItems={evidenceItems}
                                onEvidenceClick={handleEvidenceClick}
                                onNavigateToTask={(taskId) => handleNavigate('tasks', taskId)}
                            />
                        )}
                        {activeTab === 'iocs' && (
                            <IOCSummaryTab
                                caseId={caseData.case_id}
                                evidenceItems={evidenceItems}
                                onNavigate={handleNavigate}
                            />
                        )}
                        {activeTab === 'timeline' && (
                            <TimelineTab
                                caseId={caseData.case_id}
                                evidenceItems={evidenceItems}
                                onNavigate={handleNavigate}
                            />
                        )}
                        {activeTab === 'tasks' && (
                            <TaskListTab
                                caseId={caseData.case_id}
                                evidenceItems={evidenceItems}
                                onNavigate={handleNavigate}
                            />
                        )}
                        {activeTab === 'notes' && (
                            <MasterNotesTab
                                caseId={caseData.case_id}
                                evidenceItems={evidenceItems}
                                onEvidenceClick={handleEvidenceClick}
                                onNavigateToTask={(taskId) => handleNavigate('tasks', taskId)}
                            />
                        )}
                    </>
                )}
            </div>
        </div>

        {showExportDialog && caseId && caseData && (
            <ExportDialog
                caseID={caseId}
                caseNumber={caseData.case_number}
                onClose={() => setShowExportDialog(false)}
            />
        )}
        </>
    );
}
