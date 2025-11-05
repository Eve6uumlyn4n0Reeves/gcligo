export type ModuleFactory = () => Promise<any>;

export class ModuleManager {
  private modules: Record<string, any>;
  private loaders: Record<string, ModuleFactory>;
  private inflight: Map<string, Promise<any>>;

  constructor(
    initialModules: Record<string, any> = {},
    loaders: Record<string, ModuleFactory> = {},
  ) {
    this.modules = { ...initialModules };
    this.loaders = { ...loaders };
    this.inflight = new Map();
  }

  cache(): Record<string, any> {
    return this.modules;
  }

  register(name: string, loader: ModuleFactory): void {
    this.loaders[name] = loader;
  }

  set(name: string, instance: any): void {
    this.modules[name] = instance;
  }

  has(name: string): boolean {
    return Boolean(this.modules[name]);
  }

  async load(name: string): Promise<any> {
    if (this.modules[name]) {
      return this.modules[name];
    }
    const loader = this.loaders[name];
    if (!loader) {
      throw new Error(`No loader registered for module "${name}"`);
    }
    if (this.inflight.has(name)) {
      return this.inflight.get(name)!;
    }
    const promise = (async () => {
      try {
        const mod = await loader();
        this.modules[name] = mod;
        return mod;
      } finally {
        this.inflight.delete(name);
      }
    })();
    this.inflight.set(name, promise);
    return promise;
  }
}
