export interface Diagnostic {
  /** Severity indicates the diagnostic level ("error" or "warning") */
  severity: "error" | "warning";

  /** Summary is a short description of the diagnostic */
  summary: string;

  /** Detail provides additional context about the diagnostic */
  detail: string;

  /** PropPath optionally specifies which property the diagnostic relates to */
  propPath?: string[];
}

/** Diagnostics contains any warnings or errors to display to the user. */
export interface Diagnostics {
  diagnostics?: Diagnostic[];
}

// deno-lint-ignore no-explicit-any
export function isDiagnostics(value: any): value is Diagnostics {
  return value && typeof value === "object" && "diagnostics" in value;
}
