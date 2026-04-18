export function formatLines(lines: string[]): string {
  return lines.join("\n");
}

export function formatStatusBlock(title: string, lines: string[]): string {
  return [title, ...lines.map((line) => `- ${line}`)].join("\n");
}
