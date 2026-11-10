// Pendant of React mount and unmount. A Base UI part mounts when React renders
// it and unmounts when React removes it, and a portaled part stays mounted as
// long as the element that rendered it. Here one MutationObserver does that
// for every component: init(el) runs once when an element matching the
// selector appears, destroy(el) once when it is gone.
//
// GGUIX local change: the stock version re-ran document.querySelectorAll for
// every registered selector on every DOM mutation batch, which grows with the
// size of the whole document. With 1,000 server-rendered rows in the page that
// took hundreds of milliseconds per swap in WebKit. This version only looks
// inside the nodes a mutation batch added; init and destroy semantics are the
// same. See results/gotth/friction.md.
(function () {
  "use strict";

  const components = [];

  // A portaled subtree lives as long as its declaration site (portal owner),
  // like React unmounts a portal with the component that rendered it.
  function mounted(el) {
    if (!el.isConnected) return false;
    for (let node = el; node; node = node.parentElement) {
      if (node._templPortalOwner && !mounted(node._templPortalOwner)) return false;
    }
    return true;
  }

  function add(component, el) {
    if (component.elements.has(el) || !mounted(el)) return;
    component.elements.add(el);
    component.init?.(el);
  }

  function destroyGone(component) {
    for (const el of component.elements) {
      if (mounted(el)) continue;
      component.elements.delete(el);
      component.destroy?.(el);
    }
  }

  // Full scan, used once at registration.
  function sync(component) {
    destroyGone(component);
    for (const el of document.querySelectorAll(component.selector)) add(component, el);
  }

  // register(selector, { init, destroy }) wires every matching element that
  // is in the document now and every one that appears later.
  function register(selector, { init, destroy } = {}) {
    const component = { selector, init, destroy, elements: new Set() };
    components.push(component);
    if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", () => sync(component));
    else sync(component);
  }

  new MutationObserver((records) => {
    const roots = [];
    for (const r of records) {
      for (const n of r.addedNodes) if (n.nodeType === 1 && n.isConnected) roots.push(n);
    }
    for (const component of components) {
      destroyGone(component);
      for (const root of roots) {
        if (!root.isConnected) continue;
        if (root.matches(component.selector)) add(component, root);
        for (const el of root.querySelectorAll(component.selector)) add(component, el);
      }
    }
  }).observe(document.documentElement, { childList: true, subtree: true });

  window.templ = window.templ || {};
  window.templ.lifecycle = { register };
})();
