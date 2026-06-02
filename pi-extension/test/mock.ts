import type { ExtensionAPI, ToolRegistration } from "@earendil-works/pi-coding-agent";

export interface RegisteredTool {
    name: string;
    label: string;
    description: string;
    parameters: unknown;
    execute: ToolRegistration["execute"];
}

export interface RegisteredEventHandler {
    event: string;
    handler: (...args: unknown[]) => void | Promise<void>;
}

export function createMockUI() {
    const notifications: Array<{ message: string; level: string }> = [];
    const statuses: Record<string, string> = {};

    return {
        notifications,
        statuses,
        mock: {
            notify(message: string, level: string) {
                notifications.push({ message, level });
            },
            setStatus(key: string, value: string) {
                statuses[key] = value;
            },
        } as any,
    };
}

export function createMockExtensionAPI(): {
    pi: ExtensionAPI;
    registeredTools: RegisteredTool[];
    registeredEvents: RegisteredEventHandler[];
    ui: ReturnType<typeof createMockUI>;
} {
    const ui = createMockUI();
    const registeredTools: RegisteredTool[] = [];
    const registeredEvents: RegisteredEventHandler[] = [];

    const pi = {
        registerTool(reg: ToolRegistration) {
            registeredTools.push({
                name: reg.name,
                label: reg.label,
                description: reg.description,
                parameters: reg.parameters,
                execute: reg.execute,
            });
        },
        on(event: string, handler: (...args: unknown[]) => void | Promise<void>) {
            registeredEvents.push({ event, handler });
        },
    } as unknown as ExtensionAPI;

    return { pi, registeredTools, registeredEvents, ui };
}
