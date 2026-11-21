/** Wire types of the DockDuck API (doc/05-api.md). */

export interface Health {
    ok: boolean;
    version: string;
    home: string;
}

export interface ContextInfo {
    name: string;
    description: string;
    endpoint: string;
    /** the Docker CLI's active context */
    current: boolean;
    /** the context DockDuck uses right now */
    selected: boolean;
    available: boolean;
    version?: string;
    error?: string;
}

export interface Root {
    id: string;
    path: string;
}

export type ProjectStatus = "running" | "partial" | "stopped" | "unknown";

export interface ProjectSummary {
    id: string;
    name: string;
    dir: string;
    files: string[];
    env_file?: string;
    /** empty: known only to Docker (outside the scanned roots) */
    root?: string;
    status: ProjectStatus;
    running: number;
    total: number;
}

export interface Port {
    host_ip?: string;
    host_port?: number;
    container_port: number;
    proto: string;
}

export interface ContainerInfo {
    id: string;
    name: string;
    state: string;
    status: string;
    health?: string;
    exit_code: number;
    image: string;
    ports: Port[];
}

export type ServiceStatus =
    | "running"
    | "starting"
    | "restarting"
    | "unhealthy"
    | "paused"
    | "exited"
    | "error"
    | "stopped";

export interface ServiceInfo {
    name: string;
    image: string;
    status: ServiceStatus;
    health?: string;
    containers: ContainerInfo[];
    ports: Port[];
}

export interface ProjectDetail {
    project: ProjectSummary;
    services: ServiceInfo[];
    config_error: string;
    active_job?: { id: string; action: string };
}

export interface ContainerEvent {
    type: "container";
    action: string;
    project: string;
    service: string;
    container: string;
    id: string;
    time: number;
}
