export default function createPiMemoryExtension(pi: any) {
  pi.registerCommand("pi-memory-init", {
    description: "Initialize Pi Memory for the current project",
    handler: async (_args: string, ctx: any) => {
      ctx.ui?.notify?.("pi-memory-init is scaffolded but not implemented yet.", "info");
    },
  });

  pi.registerCommand("pi-memory-status", {
    description: "Show Pi Memory status for the current project",
    handler: async (_args: string, ctx: any) => {
      ctx.ui?.notify?.("pi-memory-status is scaffolded but not implemented yet.", "info");
    },
  });
}
