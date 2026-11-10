import type { ItemView, models } from "./api";

export class FormModel {
  loaded = $state<ItemView | null>(null);
  name = $state("");
  category = $state("Electronics");
  price = $state("");
  quantity = $state("0");
  status = $state("Active");
  ratingTenths = $state(0);
  featured = $state(false);
  restock = $state("");
  notes = $state("");
  errors = $state<Record<string, string>>({});
  saving = $state(false);

  load(it: ItemView | null): void {
    this.loaded = it;
    this.errors = {};
    if (!it) {
      this.name = this.price = this.restock = this.notes = "";
      this.quantity = "0";
      this.ratingTenths = 0;
      this.featured = false;
      return;
    }
    this.name = it.name;
    this.category = it.category;
    this.price = it.priceText;
    this.quantity = String(it.quantity);
    this.status = it.status;
    this.ratingTenths = Math.round(it.rating * 10);
    this.featured = it.featured;
    this.restock = it.restockDate ?? "";
    this.notes = it.notes;
  }

  reset(): void {
    this.load(this.loaded);
  }

  get dirty(): boolean {
    const l = this.loaded;
    if (!l) return false;
    return (
      this.name !== l.name ||
      this.category !== l.category ||
      this.price !== l.priceText ||
      Number(this.quantity) !== l.quantity ||
      this.quantity.trim() === "" ||
      this.status !== l.status ||
      this.ratingTenths !== Math.round(l.rating * 10) ||
      this.featured !== l.featured ||
      this.restock !== (l.restockDate ?? "") ||
      this.notes !== l.notes
    );
  }

  toEdit(): models.ItemEdit {
    return {
      id: this.loaded?.id ?? 0,
      name: this.name,
      category: this.category,
      price: this.price,
      quantity: Math.trunc(Number(this.quantity) || 0),
      status: this.status,
      rating: this.ratingTenths / 10,
      featured: this.featured,
      restockDate: this.restock,
      notes: this.notes,
    };
  }

  setErrors(errs: models.FieldErr[]): void {
    const m: Record<string, string> = {};
    for (const e of errs) m[e.field] = m[e.field] ? m[e.field] + " " + e.message : e.message;
    this.errors = m;
  }
}
