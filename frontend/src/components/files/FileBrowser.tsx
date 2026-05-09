import { useState, useEffect } from "react";
import { Folder, File, ChevronRight, ChevronDown, RefreshCw } from "lucide-react";
import { GetWorkspaceDir, ListDirectory, ReadFileContent } from "../../lib/wails";
import { useI18n } from "../../stores/i18nStore";

interface FileEntry {
  name: string;
  path: string;
  isDir: boolean;
  size: number;
  children?: FileEntry[];
}

export default function FileBrowser() {
  const [isOpen, setIsOpen] = useState(true);
  const [files, setFiles] = useState<FileEntry[]>([]);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [previewPath, setPreviewPath] = useState<string | null>(null);
  const [previewContent, setPreviewContent] = useState("");
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
        setFiles((prev) =>
          prev.map((f) => (f.path === dirPath ? { ...f, children } : f))
        );
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
      // Binary or inaccessible file
    }
  };

  const renderTree = (entries: FileEntry[], depth: number = 0) => {
    return entries.map((entry) => (
      <div key={entry.path}>
        <div
          className={`motion-lift mx-1 flex cursor-pointer items-center gap-1 rounded px-2 py-1 text-xs transition hover:bg-panel/80 ${
            previewPath === entry.path ? "bg-accent/10 text-accent" : "text-dim"
          }`}
          style={{ paddingLeft: 8 + depth * 16 }}
          onClick={() => {
            if (entry.isDir) {
              toggleDir(entry.path);
            } else {
              openFile(entry.path);
            }
          }}
        >
          {entry.isDir ? (
            <>
              {expanded.has(entry.path) ? (
                <ChevronDown size={12} />
              ) : (
                <ChevronRight size={12} />
              )}
              <Folder size={12} className="text-yellow" />
            </>
          ) : (
            <>
              <span className="w-3" />
              <File size={12} />
            </>
          )}
          <span className="truncate">{entry.name}</span>
        </div>
        {entry.isDir && expanded.has(entry.path) && entry.children && (
          <>{renderTree(entry.children, depth + 1)}</>
        )}
      </div>
    ));
  };

  return (
    <aside className={`soft-panel flex shrink-0 flex-col border-l border-border bg-surface/88 transition-all duration-300 ${isOpen ? "w-72" : "w-9"}`}>
      <div
        className="flex cursor-pointer items-center justify-between border-b border-border px-2 py-3"
        onClick={() => setIsOpen(!isOpen)}
      >
        {isOpen && (
          <span className="text-xs font-semibold text-dim uppercase">{t("files.title")}</span>
        )}
        <button
          onClick={(e) => {
            e.stopPropagation();
            refreshFiles();
          }}
          className="motion-lift rounded p-1 text-dim transition hover:bg-panel hover:text-text"
          title="Refresh"
        >
          <RefreshCw size={12} />
        </button>
      </div>

      {isOpen && (
        <>
          <div className="flex-1 overflow-y-auto py-1">{renderTree(files)}</div>

          {previewPath && previewContent && (
            <div className="h-52 overflow-y-auto border-t border-border">
              <div className="px-2 py-1 text-xs text-dim border-b border-border truncate">
                {previewPath.split("/").pop() || previewPath.split("\\").pop()}
              </div>
              <pre className="system-pre p-2 font-mono text-xs text-text whitespace-pre-wrap">
                {previewContent.slice(0, 5000)}
              </pre>
            </div>
          )}
        </>
      )}
    </aside>
  );
}
