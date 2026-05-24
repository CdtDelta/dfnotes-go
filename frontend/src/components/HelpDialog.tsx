import { useEffect, useState } from 'react';
import type { ReactNode } from 'react';

interface HelpSection {
    title: string;
    content: ReactNode;
}

interface HelpDialogProps {
    onClose: () => void;
}

const helpSections: HelpSection[] = [
    {
        title: 'Getting Started',
        content: (
            <div className="help-text">
                <p>
                    The dashboard lists all cases in the current database. Cases are shown with their case number,
                    title, classification level, and creation date.
                </p>
                <h3>Creating a Case</h3>
                <p>Click <strong>New Case</strong> from the dashboard to open the case creation form. All fields are required:</p>
                <ul>
                    <li><strong>Case Number</strong> -- your organization's case reference (e.g. DF-2025-001)</li>
                    <li><strong>Title</strong> -- a short descriptive name for the case</li>
                    <li><strong>Examiner</strong> -- the name of the lead examiner</li>
                    <li><strong>Organization</strong> -- the agency or team conducting the investigation</li>
                    <li><strong>Classification</strong> -- the sensitivity level (Unclassified through Top Secret)</li>
                    <li><strong>Ticket Number</strong> -- your organization's ticketing system reference</li>
                    <li><strong>Case Password</strong> -- used to encrypt the case key at rest. Choose a strong, unique password.
                        This password cannot be recovered. If lost, the case data is unrecoverable.</li>
                </ul>
                <h3>Opening and Unlocking a Case</h3>
                <p>
                    Click a case on the dashboard to open it. You will be prompted for the case password.
                    Enter the password and click Unlock Case (or press Enter). The case remains unlocked
                    until you explicitly lock it or navigate away from the case page.
                </p>
                <h3>Locking a Case</h3>
                <p>
                    Click the lock icon next to the case title, or use File &gt; Lock Case (Ctrl+L / Cmd+L).
                    Locking immediately purges the decryption key from memory. All case data requires the
                    password again to access.
                </p>
            </div>
        ),
    },
    {
        title: 'Note Taking',
        content: (
            <div className="help-text">
                <p>
                    The note editor supports Markdown formatting. Use the Preview toggle to switch between
                    the raw editor and the rendered preview before committing.
                </p>
                <h3>Commit Workflow</h3>
                <p>
                    Write your note in the editor, optionally preview it, then click <strong>Commit Note</strong>.
                    Committed blocks are encrypted and added to the tamper-evident hash chain. A committed block
                    cannot be edited or deleted.
                </p>
                <h3>Amendments</h3>
                <p>
                    If a committed block contains an error, use the amendment workflow rather than editing.
                    Click the amendment button on the block, enter an amendment reason, and write the corrected content.
                    The original block remains intact in the chain. The amendment creates a new block that references
                    the original. Both blocks are preserved in the record.
                </p>
                <h3>Note Levels</h3>
                <p>
                    Notes exist at two levels. <strong>Master Notes</strong> (the last tab) are case-wide
                    observations that are not specific to any evidence item. Each evidence item also has its
                    own note editor on its tab (E001, E002, etc.), used for item-specific analysis notes.
                </p>
            </div>
        ),
    },
    {
        title: 'Evidence Management',
        content: (
            <div className="help-text">
                <p>
                    Open the <strong>Evidence</strong> tab to register and manage evidence items for the case.
                </p>
                <h3>Registering Evidence</h3>
                <p>
                    Click <strong>Add Evidence Item</strong> and complete the form: name, evidence type, and
                    acquisition hash (the hash of the original source material, e.g. MD5 or SHA256 of a disk image).
                    An optional description can also be added.
                </p>
                <h3>Status Lifecycle</h3>
                <p>Each evidence item moves through the following statuses:</p>
                <ul>
                    <li><strong>Collected</strong> -- item has been received and logged</li>
                    <li><strong>Analyzing</strong> -- active examination in progress</li>
                    <li><strong>Processed</strong> -- analysis complete</li>
                    <li><strong>Archived</strong> -- item has been returned or secured in long-term storage</li>
                    <li><strong>Withdrawn</strong> -- item removed from the case. Withdrawn is a soft delete:
                        the item remains in the record and retains all its notes, but is marked as no longer active.</li>
                </ul>
                <h3>Chain of Custody</h3>
                <p>
                    Every status change generates an automatic chain of custody entry with a timestamp and
                    the authenticated user. Manual custody entries can also be added at any time from the
                    evidence item detail view.
                </p>
                <h3>Evidence Tabs</h3>
                <p>
                    Each registered evidence item gets its own tab in the case tab bar (E001, E002, and so on,
                    ordered by registration date). That tab contains the note editor for item-specific notes.
                </p>
            </div>
        ),
    },
    {
        title: 'IOC Detection',
        content: (
            <div className="help-text">
                <p>
                    IOC (Indicator of Compromise) detection runs automatically each time a note block is committed.
                    No manual action is needed to trigger detection.
                </p>
                <h3>Detected Types</h3>
                <p>The following 12 IOC types are detected:</p>
                <ul>
                    <li>IPv4 address</li>
                    <li>IPv6 address</li>
                    <li>Domain name</li>
                    <li>URL</li>
                    <li>Email address</li>
                    <li>MD5 hash</li>
                    <li>SHA1 hash</li>
                    <li>SHA256 hash</li>
                    <li>Windows file path</li>
                    <li>Unix file path</li>
                    <li>Windows registry key</li>
                    <li>CVE identifier</li>
                </ul>
                <h3>Highlight Colors</h3>
                <ul>
                    <li><strong>Yellow</strong> -- detected but not yet reviewed</li>
                    <li><strong>Red</strong> -- confirmed IOC of interest</li>
                    <li><strong>Gray strikethrough</strong> -- false positive, dismissed</li>
                </ul>
                <h3>Reviewing IOCs</h3>
                <p>
                    Right-click any highlighted value in a committed block to open the context menu.
                    From there you can confirm the IOC, dismiss it as a false positive, or restore
                    a previously reviewed IOC back to detected status.
                </p>
                <h3>IOC Summary Tab</h3>
                <p>
                    The IOC Summary tab shows every detected IOC across the entire case. Values are displayed
                    in defanged form (for example, <code>192[.]168[.]1[.]1</code> or <code>hxxps://example[.]com</code>)
                    to prevent accidental activation. You can filter by IOC type and by evidence item.
                    Click the source reference to navigate directly to the originating note block.
                    Raw values are stored in the database; defanging is applied only in the summary view.
                </p>
                <p>
                    Unix file paths are detected and appear in the IOC Summary tab but are not highlighted
                    inline in committed block text, as they produce too many false positives in typical notes.
                </p>
            </div>
        ),
    },
    {
        title: 'Timeline',
        content: (
            <div className="help-text">
                <p>
                    The Timeline tab provides a chronological record of events in the investigation.
                    Entries are created manually -- the timeline does not auto-populate from notes.
                </p>
                <h3>Adding Entries</h3>
                <p>
                    Click <strong>Add Entry</strong> and complete the form:
                </p>
                <ul>
                    <li><strong>Timestamp</strong> -- required. Must be ISO 8601 format in UTC
                        (e.g. <code>2025-03-15T14:30:00Z</code>)</li>
                    <li><strong>Event Description</strong> -- required. A brief description of the event.</li>
                    <li><strong>Investigator Notes</strong> -- optional. Additional context or observations.</li>
                    <li><strong>Secondary Timezone</strong> -- optional. An IANA timezone name
                        (e.g. <code>America/Chicago</code> or <code>Europe/London</code>). The timestamp
                        is also displayed converted to this timezone. Search by city name or UTC offset
                        in the timezone picker.</li>
                    <li><strong>Evidence Item</strong> -- optional. Links the entry to a specific evidence item.
                        The Source column on the timeline shows this link and is clickable.</li>
                </ul>
                <h3>Sorting</h3>
                <p>
                    Entries are sorted by timestamp, oldest first by default. Click the Timestamp column
                    header to reverse the sort order.
                </p>
            </div>
        ),
    },
    {
        title: 'Task List',
        content: (
            <div className="help-text">
                <p>
                    The Tasks tab provides a per-case checklist for tracking investigation activities.
                    Tasks can be assigned to a specific evidence item or to the case overall.
                </p>
                <h3>Task Statuses</h3>
                <ul>
                    <li><strong>Open</strong> -- not yet started</li>
                    <li><strong>In Progress</strong> -- actively being worked</li>
                    <li><strong>Blocked</strong> -- cannot proceed, waiting on something</li>
                    <li><strong>Complete</strong> -- finished. A completion timestamp is recorded automatically.</li>
                    <li><strong>Not Applicable</strong> -- determined to be out of scope for this case</li>
                </ul>
                <p>All status transitions are permitted in any direction -- there is no enforced order.</p>
                <h3>Templates</h3>
                <p>
                    Templates are named task sets configured in File &gt; Settings under Task Templates.
                    Click <strong>Apply Template</strong> from the Tasks tab to instantiate a template.
                    You will be prompted to select a template and assign it to an evidence item or to the
                    case overall. All tasks in the template are added at once. Templates are useful for
                    standard workflows such as hard drive imaging or malware triage.
                </p>
                <h3>Linking Notes to Tasks</h3>
                <p>
                    Committed note blocks can be linked to tasks as documentation of the work performed.
                    Open a task's detail panel (click the task title) and click <strong>Link a Note</strong>
                    to attach one or more committed blocks. A note can link to multiple tasks, and a task
                    can have multiple linked notes. At report time, linked notes show exactly where the
                    supporting detail was recorded.
                </p>
                <p>
                    You can also link from the note side: each committed block card shows a
                    <strong> Link to Task</strong> button that opens a task picker.
                </p>
            </div>
        ),
    },
    {
        title: 'Tagging',
        content: (
            <div className="help-text">
                <p>
                    Tags help categorize and filter note blocks and evidence items. dfnotes-go includes
                    28 predefined standard tags organized into four categories.
                </p>
                <h3>Analysis Tags</h3>
                <ul>
                    <li>Malware, Phishing, Data Exfiltration, Unauthorized Access, Insider Threat, Ransomware</li>
                </ul>
                <h3>Status Tags</h3>
                <ul>
                    <li>Follow Up, Confirmed, Unconfirmed, False Positive</li>
                </ul>
                <h3>Priority Tags</h3>
                <ul>
                    <li>Critical, High, Medium, Low</li>
                </ul>
                <h3>Evidence Type Tags</h3>
                <ul>
                    <li>Disk Image, Memory Dump, Network Capture, Log File, Mobile Device, Cloud Data</li>
                </ul>
                <h3>Custom Tags</h3>
                <p>
                    Custom tags can be created per case. Click the tag selector on any note block or
                    evidence item and type a new tag name.
                </p>
                <h3>Filtering</h3>
                <p>
                    Each notes tab (Master Notes and each evidence item tab) has a tag filter in the
                    top right of the committed blocks area. Select one or more tags to show only blocks
                    that carry at least one of the selected tags. The filter count shows how many blocks
                    match out of the total.
                </p>
            </div>
        ),
    },
    {
        title: 'Backup',
        content: (
            <div className="help-text">
                <p>
                    Automated backups copy the case database on a regular schedule.
                    Configure backups in File &gt; Settings under the <strong>Backup</strong> section.
                </p>
                <h3>Configuration</h3>
                <ul>
                    <li><strong>Backup Directory</strong> -- the folder where backup files are written</li>
                    <li><strong>Interval</strong> -- how often to back up, in hours (default: 6)</li>
                    <li><strong>Keep</strong> -- how many backups to retain. The oldest file is pruned
                        automatically when this limit is exceeded.</li>
                </ul>
                <h3>Backup Files</h3>
                <p>
                    Backup files are raw copies of the case database named with an ISO 8601 timestamp:
                </p>
                <p><code>dfnotes-go_backup_20250315T143000Z.db</code></p>
                <p>
                    The data inside is already encrypted at the application layer -- no additional
                    archive password is required for the backup files themselves.
                </p>
                <h3>Manual Backup</h3>
                <p>
                    Click <strong>Back Up Now</strong> in the Backup section of Settings to trigger an
                    immediate backup outside of the scheduled interval.
                </p>
                <h3>Failure Notifications</h3>
                <p>
                    If a scheduled backup fails, a notification banner appears at the top of the screen.
                    It does not auto-dismiss. Use <strong>Snooze</strong> to delay the notification or
                    <strong>Dismiss</strong> to clear it. The last backup time and status are shown in
                    Settings and persist across application restarts.
                </p>
            </div>
        ),
    },
    {
        title: 'Export',
        content: (
            <div className="help-text">
                <p>
                    Use File &gt; Export Case to export the full case to an AES-256 encrypted 7z archive.
                    The case must be unlocked to export.
                </p>
                <h3>Export Process</h3>
                <p>
                    You will be prompted for an archive password (separate from the case password) and
                    a save location. The archive password is required to open the exported archive.
                </p>
                <h3>Archive Contents</h3>
                <ul>
                    <li><code>README.txt</code> -- description and chain verification instructions</li>
                    <li><code>case_metadata.json</code> -- case metadata in plaintext JSON</li>
                    <li><code>[CASENUMBER].db</code> -- the encrypted case database</li>
                    <li><code>master_notes/</code> -- one markdown file per committed block from Master Notes</li>
                    <li><code>evidence/[ITEM]/</code> -- metadata and block files for each evidence item</li>
                    <li><code>ioc_summary.json</code> -- all IOCs with raw and defanged values</li>
                    <li><code>timeline.json</code> -- all timeline entries</li>
                    <li><code>tasks.json</code> -- all tasks with status and linked block references</li>
                    <li><code>chain_verification.json</code> -- full hash chain data for independent verification</li>
                </ul>
                <h3>Important</h3>
                <p>
                    Once extracted, the archive contents are not encrypted. Handle extracted files
                    according to your organization's data handling policy for sensitive case material.
                </p>
                <h3>Opening the Archive</h3>
                <p>
                    Use the 7z command line tool to open exports. GUI archive managers including
                    Ubuntu's Archive Manager do not reliably support AES-256 encrypted 7z archives.
                </p>
                <p>Install on Linux: <code>sudo apt install p7zip-full</code></p>
                <p>Extract command:</p>
                <p><code>7z x -p[password] archive.7z -o./output_dir/</code></p>
            </div>
        ),
    },
    {
        title: 'Settings',
        content: (
            <div className="help-text">
                <p>Open File &gt; Settings (Ctrl+, / Cmd+,) to access application settings.</p>
                <h3>Themes</h3>
                <p>
                    Select a theme via View &gt; Theme. The theme is applied immediately and persisted
                    across restarts. Available themes:
                </p>
                <ul>
                    <li>Forensic Dark (default), Classic Dark, High Contrast</li>
                    <li>Light, Solarized Dark, Monokai, Dracula</li>
                    <li>Nord, Gruvbox, Matrix, Forensic Blue</li>
                </ul>
                <h3>Database Location</h3>
                <p>
                    The current database path is shown in Settings. Click <strong>Change Location</strong>
                    to move the database to a new path (the file is moved and the original is deleted),
                    or to point the application at a different existing database (the original file is
                    untouched). The case must be locked before changing the database location.
                </p>
                <h3>Config File Location</h3>
                <ul>
                    <li>Linux: <code>~/.config/dfnotes-go/config.json</code></li>
                    <li>macOS: <code>~/Library/Application Support/dfnotes-go/config.json</code></li>
                    <li>Windows: <code>%APPDATA%\dfnotes-go\config.json</code></li>
                </ul>
                <h3>Task Templates</h3>
                <p>
                    Create and manage named task sets in the <strong>Task Templates</strong> section of
                    Settings. Each template has a name and a list of tasks (each with a title and optional
                    description). Templates created here are available in the Tasks tab via
                    <strong> Apply Template</strong>.
                </p>
            </div>
        ),
    },
    {
        title: 'Documentation Reminder',
        content: (
            <div className="help-text">
                <p>
                    The documentation reminder fires a modal alert when too much time has passed without
                    committing a note, adding a timeline entry, or recording a custody action. It is designed
                    to prevent tunnel vision during long analysis sessions -- getting absorbed in examination
                    and letting documentation slip.
                </p>
                <h3>Enabling and Configuring</h3>
                <p>
                    Open File &gt; Settings and find the <strong>Documentation Reminder</strong> section.
                    Enable the toggle and set the interval in minutes (default: 30). An inline warning
                    appears when the interval is below 30 minutes but does not block saving.
                </p>
                <h3>What Resets the Timer</h3>
                <ul>
                    <li><strong>Full reset</strong> (restarts at the full interval): committing a note block,
                        adding a timeline entry, editing a timeline entry</li>
                    <li><strong>Partial reset</strong> (restarts at half the interval): changing an evidence
                        item's status, adding a manual custody log entry</li>
                </ul>
                <h3>When the Reminder Fires</h3>
                <p>
                    The application window comes to the foreground and a modal appears showing how many
                    minutes have elapsed since the last documentation action. Four actions are available:
                </p>
                <ul>
                    <li><strong>Document Now</strong> -- closes the modal and moves focus to the note editor
                        in the active tab. Does not reset the timer -- only an actual commit does that.</li>
                    <li><strong>Snooze 15 min</strong> -- dismisses the reminder for 15 minutes, then restarts
                        at the full interval</li>
                    <li><strong>Snooze 30 min</strong> -- dismisses the reminder for 30 minutes, then restarts
                        at the full interval</li>
                    <li><strong>Pause reminders</strong> -- suspends all reminders for the rest of the session</li>
                </ul>
                <h3>Pause Indicator</h3>
                <p>
                    When reminders are paused, a bar appears below the case tab bar reading
                    "Documentation reminders paused." Click <strong>Resume</strong> to re-enable them.
                    The pause state clears automatically when the case is locked -- the next unlock starts fresh.
                </p>
                <h3>Timer Scope</h3>
                <p>
                    The timer runs only while a case is unlocked. It does not run on the dashboard,
                    login screen, or setup wizard. Opening Settings does not interrupt the timer.
                </p>
            </div>
        ),
    },
    {
        title: 'Tips and Shortcuts',
        content: (
            <div className="help-text">
                <h3>Evidence Linking</h3>
                <p>
                    Type <code>[[E001]]</code> in any note to create a clickable link to that evidence
                    item's tab. An autocomplete dropdown appears as you type <code>[[</code>, listing
                    registered evidence items.
                </p>
                <h3>Paste Handling</h3>
                <p>
                    Pasting formatted text (from a browser or document) converts it to Markdown
                    automatically. Plain text pastes as-is without conversion.
                </p>
                <h3>Images</h3>
                <p>
                    Clipboard paste of images is not supported on Linux. Use the
                    <strong> Attach Image</strong> button in the editor toolbar to attach images via
                    the file dialog on any platform.
                </p>
                <h3>IOC Detection in Code Spans</h3>
                <p>
                    IOC detection runs on all committed block text including content inside backtick
                    code spans. A hash value or CVE inside a code span will be detected and highlighted.
                </p>
                <h3>Unix File Paths</h3>
                <p>
                    Unix file paths are detected by the backend and appear in the IOC Summary tab,
                    but they are not highlighted inline in the committed block view to avoid excessive
                    false positives in notes that contain shell commands or log output.
                </p>
                <h3>Hash Chain</h3>
                <p>
                    The hash chain is global per case. A block committed on the E001 tab chains to
                    the last block committed anywhere in the case, regardless of which tab or evidence
                    item it was on. The chain is linear across all notes.
                </p>
                <h3>Classification Levels</h3>
                <p>
                    Classification levels are metadata labels only. They are displayed on the case
                    header and included in exports but do not affect encryption strength or access
                    control within the application.
                </p>
            </div>
        ),
    },
];

export default function HelpDialog({ onClose }: HelpDialogProps) {
    const [activeIdx, setActiveIdx] = useState(0);

    useEffect(() => {
        const handler = (e: KeyboardEvent) => {
            if (e.key === 'Escape') onClose();
        };
        window.addEventListener('keydown', handler);
        return () => window.removeEventListener('keydown', handler);
    }, [onClose]);

    const section = helpSections[activeIdx];

    return (
        <div
            className="fixed inset-0 z-50 flex items-center justify-center"
            style={{ backgroundColor: 'rgba(0,0,0,0.65)' }}
            onClick={onClose}
        >
            <div
                className="help-dialog"
                onClick={(e) => e.stopPropagation()}
            >
                {/* Header */}
                <div className="help-header">
                    <span style={{ fontSize: '0.9375rem', fontWeight: 600, color: 'var(--text-primary)' }}>
                        User Guide
                    </span>
                    <button
                        onClick={onClose}
                        aria-label="Close"
                        style={{ color: 'var(--text-muted)', background: 'none', border: 'none', cursor: 'pointer', padding: '2px' }}
                    >
                        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                        </svg>
                    </button>
                </div>

                {/* Body */}
                <div className="help-body">
                    {/* Left Nav */}
                    <nav className="help-nav">
                        {helpSections.map((s, idx) => (
                            <button
                                key={s.title}
                                className={`help-nav-item${activeIdx === idx ? ' active' : ''}`}
                                onClick={() => setActiveIdx(idx)}
                            >
                                {idx + 1}. {s.title}
                            </button>
                        ))}
                    </nav>

                    {/* Content */}
                    <div className="help-content">
                        <h2 style={{
                            fontSize: '1rem',
                            fontWeight: 600,
                            color: 'var(--text-primary)',
                            marginBottom: '12px',
                            paddingBottom: '8px',
                            borderBottom: '1px solid var(--border-color)',
                        }}>
                            {section.title}
                        </h2>
                        {section.content}
                    </div>
                </div>
            </div>
        </div>
    );
}
