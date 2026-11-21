/** "0.0.0.0" / "::" / "" mean every interface. */
export function isAnyIP(ip: string | undefined): boolean {
    return !ip || ip === "0.0.0.0" || ip === "::";
}

/** Host part of a published port as shown to the user. */
export function bindLabel(ips: string[]): string {
    if (ips.every((i) => isAnyIP(i))) return "all interfaces";
    if (ips.every((i) => i === "127.0.0.1" || i === "::1"))
        return "localhost only";
    return ips.join(", ");
}

/** URL to open a published HTTP-ish port from this machine. */
export function portUrl(hostPort: number, containerPort?: number): string {
    const https =
        containerPort === 443 || hostPort === 443 || hostPort === 8443;
    return `${https ? "https" : "http"}://localhost:${hostPort}`;
}

export function formatBytes(n: number): string {
    if (!n) return "0 B";
    const units = ["B", "KiB", "MiB", "GiB", "TiB"];
    const i = Math.min(
        units.length - 1,
        Math.floor(Math.log(n) / Math.log(1024)),
    );
    const v = n / 1024 ** i;
    return `${v >= 100 || i === 0 ? Math.round(v) : v.toFixed(1)} ${units[i]}`;
}

export function formatCpu(p: number): string {
    return `${p.toFixed(p >= 10 ? 0 : 1)}%`;
}
