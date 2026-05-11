import * as AppBindings from "../../wailsjs/go/app/App";
import * as RuntimeBindings from "../../wailsjs/runtime/runtime";

type EventHandler = (...data: any[]) => void;

const defaultSettings = {
  model: "deepseek-v4-pro",
  mode: "code",
  portable: false,
  minimizeToTray: false,
  maxTokens: 32768,
  temperature: 0.7,
  baseUrl: "https://api.deepseek.com",
  apiTimeout: 120,
  apiMaxRetries: 3,
  apiProxyUrl: "",
  thinkingEnabled: false,
  reasoningDisplay: "collapse",
  autoCowork: false,
  toolMode: "confirm",
  toolOverrides: {} as Record<string, string>,
  bashBlocklist: [] as string[],
  initialPrompt: "",
  roleCard: "",
  worldBook: "",
  ocrEnabled: false,
  ocrProvider: "openai_compatible",
  ocrBaseUrl: "",
  ocrModel: "",
  ocrPrompt: "Extract all readable text from this image. Preserve line breaks when useful. If there is no readable text, briefly describe the visible content.",
  ocrTimeout: 60,
};

function hasWailsBridge() {
  return Boolean((window as any)?.go?.app?.App && (window as any)?.runtime);
}

function invoke<T>(fn: () => Promise<T>, fallback: () => T | Promise<T>): Promise<T> {
  if (hasWailsBridge()) {
    return fn();
  }
  return Promise.resolve(fallback());
}

export function CreateSession(name: string) {
  return invoke(
    () => AppBindings.CreateSession(name),
    () => ({
      id: `preview-${Date.now()}`,
      name,
      model: defaultSettings.model,
      createdAt: new Date().toISOString(),
    })
  );
}

export function DeleteSession(id: string) {
  return invoke(() => AppBindings.DeleteSession(id), () => undefined);
}

export function ListSessions() {
  return invoke(() => AppBindings.ListSessions(), () => []);
}

export function SendMessage(req: any) {
  return invoke(() => AppBindings.SendMessage(req), () => undefined);
}

export function UpdateMessage(req: any) {
  return invoke(() => AppBindings.UpdateMessage(req), () => undefined);
}

export function DeleteMessageAt(req: any) {
  return invoke(() => AppBindings.DeleteMessage(req), () => undefined);
}

export function RegenerateMessage(req: any) {
  return invoke(() => AppBindings.RegenerateMessage(req), () => undefined);
}

export function BranchSession(req: any) {
  return invoke(
    () => AppBindings.BranchSession(req),
    () => ({
      id: `preview-branch-${Date.now()}`,
      name: "Branch",
      model: defaultSettings.model,
      createdAt: new Date().toISOString(),
    })
  );
}

export function AbortMessage(sessionId: string) {
  return invoke(() => AppBindings.AbortMessage(sessionId), () => undefined);
}

export function ContinueLastResponse(sessionId: string) {
  return invoke(() => AppBindings.ContinueLastResponse(sessionId), () => undefined);
}

export function GetContextSummary(sessionId: string) {
  return invoke(
    () => AppBindings.GetContextSummary(sessionId),
    () => ({ summary: "", tokens: 0 })
  );
}

export function UpdateContextSummary(req: { sessionId: string; summary: string }) {
  return invoke(() => AppBindings.UpdateContextSummary(req), () => undefined);
}

export function ArchiveSession(sessionId: string) {
  return invoke(() => AppBindings.ArchiveSession(sessionId), () => undefined);
}

export function GetHistory(sessionId: string) {
  return invoke(() => AppBindings.GetHistory(sessionId), () => []);
}

export function ListTools() {
  return invoke(
    () => AppBindings.ListTools(),
    () => [
      { name: "read_file", description: "Read a file from the filesystem." },
      { name: "write_file", description: "Write a file to the filesystem." },
      { name: "edit_file", description: "Perform exact string replacements in a file." },
      { name: "glob", description: "Find files by glob pattern." },
      { name: "grep", description: "Search files with a regular expression." },
      { name: "bash", description: "Execute a shell command." },
      { name: "web_fetch", description: "Fetch a web page." },
      { name: "web_search", description: "Search the web." },
    ]
  );
}

export function ListDirectory(dirPath: string) {
  return invoke(() => AppBindings.ListDirectory(dirPath), () => []);
}

export function ReadFileContent(path: string) {
  return invoke(() => AppBindings.ReadFileContent(path), () => "");
}

export function ReadFileSnippet(path: string, maxBytes: number) {
  return invoke(
    () => AppBindings.ReadFileSnippet(path, maxBytes),
    () => ({
      name: path.split(/[\\/]/).pop() || path,
      path,
      size: 0,
      content: "",
      truncated: false,
      binary: false,
    })
  );
}

export function SearchWorkspaceFiles(query: string, limit = 20) {
  return invoke(
    () => AppBindings.SearchWorkspaceFiles(query, limit),
    () => []
  );
}

export function ListOCRPresets() {
  return invoke(
    () => AppBindings.ListOCRPresets(),
    () => []
  );
}

export function OCRImage(req: any) {
  return invoke(
    () => AppBindings.OCRImage(req),
    () => ({
      text: "",
      provider: defaultSettings.ocrProvider,
      model: defaultSettings.ocrModel,
    })
  );
}

export function OCRImageFile(path: string) {
  return invoke(
    () => AppBindings.OCRImageFile(path),
    () => ({
      text: "",
      provider: defaultSettings.ocrProvider,
      model: defaultSettings.ocrModel,
    })
  );
}

export function GetWorkspaceDir() {
  return invoke(() => AppBindings.GetWorkspaceDir(), () => "D:\\DeepHermes");
}

export function OpenFileDialog() {
  return invoke(() => AppBindings.OpenFileDialog(), () => "");
}

export function OpenDirectoryDialog() {
  return invoke(() => AppBindings.OpenDirectoryDialog(), () => "");
}

export function GetSettings() {
  const stored = localStorage.getItem("deephermes.preview.settings");
  return invoke(
    () => AppBindings.GetSettings(),
    () => (stored ? { ...defaultSettings, ...JSON.parse(stored) } : defaultSettings)
  );
}

export function ExportSettings() {
  return invoke(() => AppBindings.ExportSettings(), () => "preview-settings.yaml");
}

export function ImportSettings() {
  return invoke(() => AppBindings.ImportSettings(), () => undefined);
}

export function ImportCharacterCard() {
  return invoke(
    () => AppBindings.ImportCharacterCard(),
    () => ({
      name: "Preview Character",
      roleCard: "## Name\nPreview Character\n\n## Description\nPreview mode character card.",
      worldBook: "",
      source: "preview",
    })
  );
}

export function HideMainWindow() {
  return invoke(() => AppBindings.HideMainWindow(), () => undefined);
}

export function RestoreMainWindow() {
  return invoke(() => AppBindings.RestoreMainWindow(), () => undefined);
}

export function QuitApp() {
  return invoke(() => AppBindings.QuitApp(), () => undefined);
}

export function GetDiagnostics() {
  return invoke<any>(
    () => AppBindings.GetDiagnostics(),
    () => ({
      version: "1.0.0",
      buildCommit: "preview",
      buildDate: "preview",
      goVersion: "preview",
      platform: "browser",
      arch: "wasm",
      configPath: "preview",
      dataDir: "preview",
      sessionsDir: "preview",
      portable: false,
      minimizeToTray: defaultSettings.minimizeToTray,
      model: defaultSettings.model,
      mode: defaultSettings.mode,
      baseUrl: defaultSettings.baseUrl,
      apiKeyStatus: localStorage.getItem("deephermes.preview.apiKeyStatus") || "missing",
      sessionCount: 0,
      memoryDir: "preview",
      recentLogs: [],
    })
  );
}

export function UpdateSettings(settings: typeof defaultSettings) {
  return invoke(
    () => AppBindings.UpdateSettings(settings),
    () => {
      localStorage.setItem("deephermes.preview.settings", JSON.stringify(settings));
    }
  );
}

export function SetAPIKey(key: string) {
  return invoke(
    () => AppBindings.SetAPIKey(key),
    () => {
      localStorage.setItem("deephermes.preview.apiKeyStatus", key.trim() ? "configured" : "missing");
    }
  );
}

export function TestAPIKey(req: any) {
  return invoke(
    () => AppBindings.TestAPIKey(req),
    () => {
      const configured = localStorage.getItem("deephermes.preview.apiKeyStatus") === "configured";
      return {
        ok: Boolean(req?.apiKey || configured),
        message: "Preview mode",
        latencyMs: 0,
      };
    }
  );
}

export function SetOCRAPIKey(key: string) {
  return invoke(
    () => AppBindings.SetOCRAPIKey(key),
    () => {
      localStorage.setItem("deephermes.preview.ocrKeyStatus", key.trim() ? "configured" : "missing");
    }
  );
}

export function GetAPIKeyStatus() {
  return invoke(
    () => AppBindings.GetAPIKeyStatus(),
    () => localStorage.getItem("deephermes.preview.apiKeyStatus") || "missing"
  );
}

export function GetOCRAPIKeyStatus() {
  return invoke(
    () => AppBindings.GetOCRAPIKeyStatus(),
    () => localStorage.getItem("deephermes.preview.ocrKeyStatus") || "missing"
  );
}

export function SetThinking(enabled: boolean) {
  return invoke(() => AppBindings.SetThinking(enabled), () => undefined);
}

export function GetModelInfo() {
  return invoke(
    () => AppBindings.GetModelInfo(),
    () => ({
      current: defaultSettings.model,
      available: ["deepseek-v4-pro", "deepseek-v4-flash", "deepseek-reasoner", "deepseek-chat"],
    })
  );
}

export function SpawnSubAgent(req: any) {
  return invoke(() => AppBindings.SpawnSubAgent(req), () => `preview-subagent-${Date.now()}`);
}

export function CancelSubAgent(id: string) {
  return invoke(() => AppBindings.CancelSubAgent(id), () => undefined);
}

export function GetSubAgents() {
  return invoke(() => AppBindings.GetSubAgents(), () => []);
}

export function ApproveToolCall(id: string) {
  return invoke(() => AppBindings.ApproveToolCall(id), () => undefined);
}

export function RejectToolCall(id: string) {
  return invoke(() => AppBindings.RejectToolCall(id), () => undefined);
}

export function RollbackToolChange(id: string) {
  return invoke(
    () => AppBindings.RollbackToolChange(id),
    () => ({
      restored: true,
      deleted: false,
      path: "",
      message: "Preview rollback",
    })
  );
}

export function WindowMinimise() {
  if (hasWailsBridge()) RuntimeBindings.WindowMinimise();
}

export function WindowMaximise() {
  if (hasWailsBridge()) RuntimeBindings.WindowMaximise();
}

export function WindowUnmaximise() {
  if (hasWailsBridge()) RuntimeBindings.WindowUnmaximise();
}

export function WindowToggleMaximise() {
  if (hasWailsBridge()) RuntimeBindings.WindowToggleMaximise();
}

export function WindowIsMaximised() {
  return invoke(() => RuntimeBindings.WindowIsMaximised(), () => false);
}

export function WindowGetSize() {
  return invoke(() => RuntimeBindings.WindowGetSize(), () => ({ w: 1200, h: 800 }));
}

export function WindowSetSize(width: number, height: number) {
  if (hasWailsBridge()) RuntimeBindings.WindowSetSize(width, height);
}

export function WindowGetPosition() {
  return invoke(() => RuntimeBindings.WindowGetPosition(), () => ({ x: 120, y: 80 }));
}

export function WindowSetPosition(x: number, y: number) {
  if (hasWailsBridge()) RuntimeBindings.WindowSetPosition(x, y);
}

export function Quit() {
  if (hasWailsBridge()) RuntimeBindings.Quit();
}

export function ClipboardSetText(text: string) {
  return invoke(() => RuntimeBindings.ClipboardSetText(text), async () => {
    await navigator.clipboard?.writeText(text);
    return true;
  });
}

export function OnFileDrop(callback: (x: number, y: number, paths: string[]) => void, useDropTarget = true) {
  if (hasWailsBridge()) RuntimeBindings.OnFileDrop(callback, useDropTarget);
}

export function OnFileDropOff() {
  if (hasWailsBridge()) RuntimeBindings.OnFileDropOff();
}

export function EventsOn(eventName: string, callback: EventHandler) {
  if (hasWailsBridge()) {
    return RuntimeBindings.EventsOn(eventName, callback);
  }
  return () => undefined;
}

export function EventsOff(eventName: string, ...additionalEventNames: string[]) {
  if (hasWailsBridge()) RuntimeBindings.EventsOff(eventName, ...additionalEventNames);
}

export function EventsOnce(eventName: string, callback: EventHandler) {
  if (hasWailsBridge()) {
    return RuntimeBindings.EventsOnce(eventName, callback);
  }
  return () => undefined;
}

export function EventsEmit(eventName: string, ...data: any[]) {
  if (hasWailsBridge()) RuntimeBindings.EventsEmit(eventName, ...data);
}

export function LogPrint(message: string) {
  if (hasWailsBridge()) RuntimeBindings.LogPrint(message);
}

export function LogError(message: string) {
  if (hasWailsBridge()) RuntimeBindings.LogError(message);
}

export function getApp() {
  return (window as any)?.go?.app?.App;
}
