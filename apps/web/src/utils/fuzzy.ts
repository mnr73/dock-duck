/**
 * Subsequence match for the command palette: every query character must appear in
 * order. A lower score is a better match; `null` means no match.
 *
 * Ranking, best first: exact name, prefix, word initials ("rp" -> "Rescan projects"),
 * substring, then a plain subsequence. Gaps between matched characters add to the score.
 */
export function fuzzyScore(query: string, text: string): number | null {
    const q = query.trim().toLowerCase();
    if (!q) return 0;
    const t = text.toLowerCase();
    if (t === q) return 0;
    if (t.startsWith(q)) return 1 + t.length / 1000;
    if (matchesInitials(q, t)) return 5 + t.length / 1000;
    const at = t.indexOf(q);
    if (at >= 0) return 10 + at + t.length / 1000;

    let score = 100;
    let from = 0;
    let last = -1;
    for (const ch of q) {
        const i = t.indexOf(ch, from);
        if (i < 0) return null;
        const wordStart = i === 0 || /[\s\-_/.]/.test(t[i - 1] ?? "");
        score += (last < 0 ? 0 : i - last - 1) * 2 + (wordStart ? 0 : 3);
        last = i;
        from = i + 1;
    }
    return score + t.length / 1000;
}

/** Every query character is the first letter of a later word, in order. */
function matchesInitials(q: string, t: string): boolean {
    if (q.length < 2) return false;
    const initials = t
        .split(/[\s\-_/.]+/)
        .filter(Boolean)
        .map((w) => w[0]);
    let from = 0;
    for (const ch of q) {
        const i = initials.indexOf(ch, from);
        if (i < 0) return false;
        from = i + 1;
    }
    return true;
}

/** Best score of a query over several fields (name, path, keywords). */
export function bestScore(query: string, fields: string[]): number | null {
    let best: number | null = null;
    for (const f of fields) {
        const s = fuzzyScore(query, f);
        if (s !== null && (best === null || s < best)) best = s;
    }
    return best;
}
