// The backend API. In the app this is the generated ItemService binding. Tests
// replace it with setApi().
import * as real from "../../bindings/gguix/apps/wails/itemservice.js";
import type * as models from "../../bindings/gguix/apps/wails/models.js";

export type { models };
export type ItemView = models.ItemView;
export type QueryArg = models.QueryArg;
export type PrefsView = models.PrefsView;

export type Api = {
  AppInfo(): PromiseLike<models.AppInfo>;
  CancelExport(): PromiseLike<void>;
  CopyRow(id: number): PromiseLike<models.CopyResult>;
  EmitMetric(event: string, fieldsJSON: string): PromiseLike<void>;
  FindRow(q: models.QueryArg, id: number, maxRows: number): PromiseLike<number>;
  Get(id: number): PromiseLike<models.ItemView>;
  HasLastExport(): PromiseLike<boolean>;
  HostInfo(): PromiseLike<models.HostView>;
  ImportNotes(): PromiseLike<models.ImportResult>;
  LoadPrefs(): PromiseLike<models.PrefsView>;
  Page(req: models.PageArg): PromiseLike<models.PageView>;
  Palette(): PromiseLike<models.PaletteSet>;
  PrefsFlushed(): PromiseLike<void>;
  Quit(): PromiseLike<void>;
  RevealConfigFolder(): PromiseLike<void>;
  RevealLastExport(): PromiseLike<void>;
  SavePrefs(p: models.PrefsView): PromiseLike<void>;
  SetMenuState(s: models.MenuState): PromiseLike<void>;
  StartExport(q: models.QueryArg): PromiseLike<models.ExportStart>;
  Update(e: models.ItemEdit): PromiseLike<models.UpdateResult>;
};

let impl: Api = real as unknown as Api;

export function setApi(a: Api): void {
  impl = a;
}

export const api: Api = new Proxy({} as Api, {
  get: (_t, name: string) => (...args: unknown[]) => (impl as unknown as Record<string, (...a: unknown[]) => unknown>)[name](...args),
});
