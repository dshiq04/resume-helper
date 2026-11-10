export function companyInitial(name: string): string {
  return Array.from(name.trim())[0]?.toUpperCase() ?? "?";
}

export function avatarTone(name: string): number {
  let hash = 0;
  for (const character of name) {
    hash = (hash * 31 + (character.codePointAt(0) ?? 0)) >>> 0;
  }
  return hash % 6;
}

export function prettyLink(link: string): string {
  try {
    const url = new URL(link);
    const host = url.hostname.replace(/^www\./, "");
    const path = url.pathname === "/" ? "" : url.pathname;
    const display = `${host}${path}`;
    return display.length > 48 ? `${display.slice(0, 45)}...` : display;
  } catch {
    return link;
  }
}
