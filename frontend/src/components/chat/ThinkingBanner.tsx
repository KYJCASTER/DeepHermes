import { Brain, X } from "lucide-react";
import { useI18n } from "../../stores/i18nStore";

interface Props {
  onDismiss: () => void;
}

export default function ThinkingBanner({ onDismiss }: Props) {
  const { t } = useI18n();
  return (
    <div className="mx-5 mt-3 flex items-center gap-2 rounded border border-yellow/30 bg-yellow/10 px-3 py-2 text-sm">
      <Brain size={14} className="text-yellow shrink-0 animate-pulse" />
      <span className="text-yellow/80 flex-1">{t("chat.thinking")}</span>
      <button
        onClick={onDismiss}
        className="rounded p-0.5 text-dim transition hover:bg-bg hover:text-text"
      >
        <X size={14} />
      </button>
    </div>
  );
}
