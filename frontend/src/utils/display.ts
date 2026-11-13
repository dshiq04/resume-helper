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

/** Returns true when the value points to an openable web page. */
export function isWebLink(link: string): boolean {
  const trimmed = link.trim();
  if (!trimmed || /\s/.test(trimmed)) {
    return false;
  }
  if (/^[a-z][a-z\d+.-]*:/i.test(trimmed)) {
    try {
      const url = new URL(trimmed);
      return url.protocol === "http:" || url.protocol === "https:";
    } catch {
      return false;
    }
  }
  return /^(?:[\w-]+\.)+[a-z]{2,}(?::\d+)?(?:[/?#].*)?$/i.test(trimmed);
}

/** Adds https:// to bare domains so they can be opened externally. */
export function withProtocol(link: string): string {
  const trimmed = link.trim();
  return /^[a-z][a-z\d+.-]*:/i.test(trimmed) ? trimmed : `https://${trimmed}`;
}
