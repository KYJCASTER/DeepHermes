import * as AppBindings from "../../wailsjs/go/app/App";
import * as RuntimeBindings from "../../wailsjs/runtime/runtime";

type EventHandler = (...data: any[]) => void;

const defaultSettings = {
  model: "deepseek-v4-pro",
  maxTokens: 32768,
  temperature: 0.7,
  baseUrl: "https://api.deepseek.com",
  thinkingEnabled: false,
  autoCowork: false,
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

export function AbortMessage(sessionId: string) {
  return invoke(() => AppBindings.AbortMessage(sessionId), () => undefined);
}

export function GetHistory(sessionId: string) {
  return invoke(() => AppBindings.GetHistory(sessionId), () => []);
}

export function ListTools() {
  return invoke(() => AppBindings.ListTools(), () => []);
}

export function ListDirectory(dirPath: string) {
  return invoke(() => AppBindings.ListDirectory(dirPath), () => []);
}

export function ReadFileContent(path: string) {
  return invoke(() => AppBindings.ReadFileContent(path), () => "");
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

export function GetAPIKeyStatus() {
  return invoke(
    () => AppBindings.GetAPIKeyStatus(),
    () => localStorage.getItem("deephermes.preview.apiKeyStatus") || "missing"
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

export function Quit() {
  if (hasWailsBridge()) RuntimeBindings.Quit();
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
