import logoSvg from "@/assets/logo.svg?raw";

/** The two fills of the supplied duck: the body/windows colour and the light tint. */
const BASE = "#F1B74C";
const SOFT = "#F8D996";

function channels(hex: string): [number, number, number] {
    const n = Number.parseInt(hex.slice(1), 16);
    return [(n >> 16) & 255, (n >> 8) & 255, n & 255];
}

/** `hex` mixed with white; `white` is the share of white (0 to 1). */
export function tint(hex: string, white: number): string {
    return (
        "#" +
        channels(hex)
            .map((c) =>
                Math.round(c + (255 - c) * white)
                    .toString(16)
                    .padStart(2, "0"),
            )
            .join("")
    );
}

/**
 * The brand duck in the given accent. `null` keeps the artwork exactly as supplied.
 * The tint share is the one the original uses (F1B74C -> F8D996).
 */
export function brandSvg(accent: string | null): string {
    if (!accent) return logoSvg;
    return logoSvg
        .replaceAll(BASE, accent)
        .replaceAll(SOFT, tint(accent, 0.42));
}

export function brandDataUrl(accent: string | null): string {
    return `data:image/svg+xml,${encodeURIComponent(brandSvg(accent))}`;
}
