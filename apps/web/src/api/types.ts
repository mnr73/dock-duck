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
    /** has a `build:` section (can be built; its image is not pulled from a registry) */
    buildable: boolean;
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

export interface Usage {
    cpu_percent: number;
    mem_usage: string;
    mem_bytes: number;
    mem_percent: number;
}

export type JobStatus = "running" | "succeeded" | "failed";

export interface JobInfo {
    id: string;
    project: string;
    action: string;
    services?: string[];
    status: JobStatus;
    error?: string;
    started_at: string;
    ended_at?: string;
}

export interface DownPreview {
    project: string;
    containers: string[];
    networks: string[];
    volumes: string[];
}

export interface PortEntry {
    host_ips: string[];
    host_port: number;
    container_port: number;
    proto: string;
    project?: string;
    service?: string;
    container: string;
    container_id: string;
    image: string;
}

export interface ContainerDetail {
    id: string;
    name: string;
    image: string;
    created: string;
    command: string;
    entrypoint?: string;
    working_dir?: string;
    user?: string;
    state: string;
    exit_code: number;
    error?: string;
    oom_killed: boolean;
    started_at?: string;
    finished_at?: string;
    restart_count: number;
    restart_policy: string;
    privileged: boolean;
    health?: string;
    health_output?: string;
    mounts: {
        type: string;
        name?: string;
        source: string;
        destination: string;
        rw: boolean;
    }[];
    networks: { name: string; ip: string; aliases?: string[] }[];
    ports: Port[];
}
