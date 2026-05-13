export function errorMessage(error: unknown) {
  if (error instanceof Error) return error.message;
  if (typeof error === "string") return error;
  if (error && typeof error === "object" && "message" in error) {
    const message = (error as { message?: unknown }).message;
    if (typeof message === "string") return message;
  }
  return String(error);
}

export function isWorkspaceBoundaryError(error: unknown) {
  const message = errorMessage(error).toLowerCase();
  return (
    message.includes("outside the allowed workspace") ||
    message.includes("cannot resolve workspace directory") ||
    message.includes("cannot resolve target path")
  );
}

export function friendlyError(error: unknown, workspaceMessage: string) {
  if (isWorkspaceBoundaryError(error)) {
    return workspaceMessage;
  }
  return errorMessage(error);
}
