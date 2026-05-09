import { useEffect, useState } from "react";
import { ChevronDown, ChevronLeft, ChevronRight, File, Folder, PanelRight, RefreshCw } from "lucide-react";
import type { MouseEvent as ReactMouseEvent } from "react";
import { GetWorkspaceDir, ListDirectory, ReadFileContent } from "../../lib/wails";
import { useI18n } from "../../stores/i18nStore";
import { useLayoutStore } from "../../stores/layoutStore";

interface FileEntry {
  name: string;
  path: string;
  isDir: boolean;
  size: number;
  children?: FileEntry[];
}

function formatBytes(value: number) {
  if (!value) return "";
  if (value < 1024) return `${value} B`;
  if (value < 1024 * 1024) return `${Math.round(value / 1024)} KB`;
  return `${(value / 1024 / 1024).toFixed(1)} MB`;
}

export default function FileBrowser() {
  const [isOpen, setIsOpen] = useState(true);
  const [files, setFiles] = useState<FileEntry[]>([]);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [previewPath, setPreviewPath] = useState<string | null>(null);
  const [previewContent, setPreviewContent] = useState("");
  const filePanelWidth = useLayoutStore((s) => s.filePanelWidth);
  const setFilePanelWidth = useLayoutStore((s) => s.setFilePanelWidth);
  const { t } = useI18n();

  useEffect(() => {
    refreshFiles();
  }, []);

  const refreshFiles = async () => {
    try {
      const dir = await GetWorkspaceDir();
      const entries = await ListDirectory(dir);
      setFiles(entries);
    } catch (e) {
      console.error("Failed to list directory:", e);
    }
  };

  const toggleDir = async (dirPath: string) => {
    const next = new Set(expanded);
    if (next.has(dirPath)) {
      next.delete(dirPath);
    } else {
      next.add(dirPath);
      try {
        const children = await ListDirectory(dirPath);
        setFiles((prev) => prev.map((f) => (f.path === dirPath ? { ...f, children } : f)));
      } catch (e) {
        console.error("Failed to list directory:", e);
      }
    }
    setExpanded(next);
  };

  const openFile = async (path: string) => {
    if (path === previewPath) {
      setPreviewPath(null);
      setPreviewContent("");
      return;
    }
    try {
      const content = await ReadFileContent(path);
      setPreviewPath(path);
      setPreviewContent(content);
    } catch {
      setPreviewPath(null);
      setPreviewContent("");
    }
  };

  const renderTree = (entries: FileEntry[], depth = 0) => {
    return entries.map((entry) => (
      <div key={entry.path}>
        <div
          className={`mx-2 flex cursor-pointer items-center gap-1.5 rounded px-2 py-1.5 text-xs transition hover:bg-panel/80 ${
            previewPath === entry.path ? "bg-accent/10 text-accent" : "text-dim"
          }`}
          style={{ paddingLeft: 8 + depth * 14 }}
          onClick={() => {
            if (entry.isDir) {
              toggleDir(entry.path);
            } else {
              openFile(entry.path);
            }
          }}
        >
          {entry.isDir ? (
            expanded.has(entry.path) ? (
              <ChevronDown size={12} />
            ) : (
              <ChevronRight size={12} />
            )
          ) : (
            <span className="w-3" />
          )}
          {entry.isDir ? <Folder size={13} className="text-yellow" /> : <File size={13} />}
          <span className="min-w-0 flex-1 truncate">{entry.name}</span>
          {!entry.isDir && <span className="text-[10px] text-dim">{formatBytes(entry.size)}</span>}
        </div>
        {entry.isDir && expanded.has(entry.path) && entry.children && renderTree(entry.children, depth + 1)}
      </div>
    ));
  };

  const startResize = (event: ReactMouseEvent) => {
    event.preventDefault();
    const onMove = (moveEvent: MouseEvent) => setFilePanelWidth(window.innerWidth - moveEvent.clientX);
    const onUp = () => {
      window.removeEventListener("mousemove", onMove);
      window.removeEventListener("mouseup", onUp);
    };
    window.addEventListener("mousemove", onMove);
    window.addEventListener("mouseup", onUp);
  };

  return (
    <aside
      className="rail-panel relative flex shrink-0 flex-col border-l border-border transition-all duration-300"
      style={{ width: isOpen ? filePanelWidth : 40 }}
    >
      {isOpen && (
        <div
          onMouseDown={startResize}
          className="resize-handle absolute left-[-3px] top-0 z-10 h-full w-1.5 cursor-col-resize transition"
        />
      )}

      <div className="flex items-center justify-between border-b border-border px-3 py-3">
        {isOpen ? (
          <div className="min-w-0">
            <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-dim">{t("files.title")}</p>
            <p className="mt-1 truncate text-xs text-dim">{previewPath ? previewPath : "Workspace"}</p>
          </div>
        ) : (
          <PanelRight size={15} className="mx-auto text-dim" />
        )}
        <div className="flex items-center gap-1">
          {isOpen && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                refreshFiles();
              }}
              className="icon-button h-7 w-7"
              title="Refresh"
            >
              <RefreshCw size={13} />
            </button>
          )}
          <button
            onClick={() => setIsOpen(!isOpen)}
            className="icon-button h-7 w-7"
            title={isOpen ? "Collapse" : "Expand"}
          >
            {isOpen ? <ChevronRight size={13} /> : <ChevronLeft size={13} />}
          </button>
        </div>
      </div>

      {isOpen && (
        <>
          <div className="flex-1 overflow-y-auto py-2">{renderTree(files)}</div>

          {previewPath && previewContent && (
            <div className="h-56 overflow-y-auto border-t border-border bg-bg/42">
              <div className="truncate border-b border-border px-3 py-2 text-xs font-medium text-text">
                {previewPath.split("/").pop() || previewPath.split("\\").pop()}
              </div>
              <pre className="system-pre whitespace-pre-wrap p-3 font-mono text-xs leading-5 text-text">
                {previewContent.slice(0, 5000)}
              </pre>
            </div>
          )}
        </>
      )}
    </aside>
  );
}
